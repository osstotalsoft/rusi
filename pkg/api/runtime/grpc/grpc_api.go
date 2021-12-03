package grpc

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net"
	"rusi/pkg/api/runtime"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	v1 "rusi/pkg/proto/runtime/v1"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/klog/v2"
)

func NewGrpcAPI(port int, serverOptions ...grpc.ServerOption) runtime.Api {
	return &grpcApi{port: port, serverOptions: serverOptions}
}

type grpcApi struct {
	port             int
	server           *rusiServerImpl
	serverOptions    []grpc.ServerOption
	publishHandler   messaging.PublishRequestHandler
	subscribeHandler messaging.SubscribeRequestHandler
}

func (srv *grpcApi) SetPublishHandler(publishHandler messaging.PublishRequestHandler) {
	srv.publishHandler = publishHandler
}
func (srv *grpcApi) SetSubscribeHandler(subscribeHandler messaging.SubscribeRequestHandler) {
	srv.subscribeHandler = subscribeHandler
}
func (srv *grpcApi) Refresh() error {
	if srv.server != nil {
		return srv.server.Refresh()
	}
	return nil
}

func (srv *grpcApi) Serve(ctx context.Context) error {
	srv.server = &rusiServerImpl{
		refreshChannels:  []chan bool{},
		mainCtx:          ctx,
		publishHandler:   srv.publishHandler,
		subscribeHandler: srv.subscribeHandler,
	}
	grpcServer := grpc.NewServer(srv.serverOptions...)
	v1.RegisterRusiServer(grpcServer, srv.server)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", srv.port))
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}

	go func() {
		select {
		case <-ctx.Done():
			//grpcServer.GracefulStop() should also work
			grpcServer.Stop()
		}
	}()

	return grpcServer.Serve(lis)
}

type rusiServerImpl struct {
	mu               sync.RWMutex
	mainCtx          context.Context
	refreshChannels  []chan bool
	publishHandler   messaging.PublishRequestHandler
	subscribeHandler messaging.SubscribeRequestHandler
}

func (srv *rusiServerImpl) Refresh() error {
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	for _, channel := range srv.refreshChannels {
		channel <- true
	}
	return nil
}

func (srv *rusiServerImpl) createRefreshChan() chan bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	c := make(chan bool)
	srv.refreshChannels = append(srv.refreshChannels, c)
	return c
}

func (srv *rusiServerImpl) removeRefreshChan(refreshChan chan bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	var s []chan bool
	for _, channel := range srv.refreshChannels {
		if channel != refreshChan {
			s = append(s, channel)
		}
	}
	srv.refreshChannels = s
}

type internalSubscription struct {
	subscriptionOptions *messaging.SubscriptionOptions
	pubsubName          string
	topic               string
	handler             func(buildCtx context.Context) messaging.Handler
	closeFunc           messaging.CloseFunc
}

type subAck struct {
	ackHandler messaging.AckHandler
	errCh      chan error
}

// Subscribe creates a subscription
func (srv *rusiServerImpl) Subscribe(stream v1.Rusi_SubscribeServer) error {

	eg, ctx := errgroup.WithContext(stream.Context())
	ackLock := &sync.RWMutex{}
	subRequestChan := make(chan *v1.SubscriptionRequest)
	subAckMap := map[string]*subAck{}

	eg.Go(func() error {
		return startStreamListener(ctx, stream, subRequestChan, subAckMap, ackLock)
	})

	eg.Go(func() error {
		return startSubscriptionRequestsListener(ctx, stream, subRequestChan, srv.subscribeHandler, subAckMap, ackLock)
	})

	return eg.Wait()
}

func startStreamListener(ctx context.Context, stream v1.Rusi_SubscribeServer, subsChan chan *v1.SubscriptionRequest) error {

	//wait for ack or subscriptionRequest from the client app
	for {
		select {
		case <-ctx.Done():
			klog.V(4).ErrorS(ctx.Err(), "stopping stream listener")
			return ctx.Err()
		default:
			r, err := stream.Recv() //blocks
			if err != nil {
				klog.V(4).ErrorS(err, "stream reading error or stream is closed")
				return err
			}
			if r.GetRequestType() == nil {
				klog.V(4).InfoS("invalid message received")
			}
			if r.GetAckRequest() != nil {
				err = handleAck(r.GetAckRequest(), subAckMap, mu)
			}
			if r.GetSubscriptionRequest() != nil {
				subsChan <- r.GetSubscriptionRequest()
			}
			if err != nil {
				return err
			}
		}
	}
}

func startSubscriptionRequestsListener(ctx context.Context, stream v1.Rusi_SubscribeServer,
	subsChan chan *v1.SubscriptionRequest, subscribeHandler messaging.SubscribeRequestHandler,
	subAckMap map[string]*subAck, ackLock *sync.RWMutex) error {

	subscriptions := map[string]internalSubscription{}
	hCtx, hCancel := context.WithCancel(ctx)
	defer hCancel()

	for {

		//blocks until done or refresh
		select {
		case request := <-subsChan:
			key := fmt.Sprintf("%s_%s", request.GetPubsubName(), request.GetTopic())
			sub := internalSubscription{
				subscriptionOptions: messagingSubscriptionOptions(request.GetOptions()),
				pubsubName:          request.GetPubsubName(),
				topic:               request.GetTopic(),
			}

			sub.handler = buildSubscribeHandler(stream, sub.acks)

			unsub, err := subscribeHandler(ctx, messaging.SubscribeRequest{
				PubsubName: subscriptions[key].pubsubName,
				Topic:      subscriptions[key].topic,
				Handler:    subscriptions[key].handler(hCtx),
				Options:    subscriptions[key].subscriptionOptions,
			})
			sub.closeFunc = unsub
			subscriptions[key] = sub

			if err != nil {
				hCancel()
				return err
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func buildSubscribeHandler(stream v1.Rusi_SubscribeServer, acks map[string]*subAck) func(context.Context) messaging.Handler {

	//used to synchronize sending on grpc
	mu := &sync.Mutex{}

	return func(buildCtx context.Context) messaging.Handler {
		return func(ctx context.Context, env *messaging.MessageEnvelope) error {
			if env.Id == "" {
				return errors.New("message id is missing")
			}

			errChan := make(chan error)
			mu.Lock()
			acks[env.Id] = &subAck{nil, errChan}
			mu.Unlock()
			//cleanup
			defer func() {
				mu.Lock()
				delete(acks, env.Id)
				mu.Unlock()
			}()

			//send message to GRPC
			data, err := serdes.Marshal(env.Payload)
			if err != nil {
				return err
			}
			err = stream.Send(&v1.ReceivedMessage{
				Id:       env.Id,
				Data:     data,
				Metadata: env.Headers,
			})
			if err != nil {
				return err
			}
			klog.V(4).InfoS("Message sent to grpc, waiting for ack", "topic", env.Subject, "id", env.Id)

			select {
			//handler builder closed context
			case <-buildCtx.Done():
				klog.V(4).InfoS("Context done before ack", "message", buildCtx.Err())
				return buildCtx.Err()
			//subscriber context is done
			case <-ctx.Done():
				klog.V(4).InfoS("Context done before ack", "message", ctx.Err())
				return ctx.Err()
			case err = <-errChan:
				klog.V(4).InfoS("Ack sent to pubsub", "topic", env.Subject, "Id", env.Id, "error", err)
				return err
			}
		}
	}
}

func handleAck(ackRequest *v1.AckRequest, subAckMap map[string]*subAck, mu *sync.RWMutex) error {
	if ackRequest.GetError() != "" {
		err = errors.New(ackRequest.GetError())
	}

	mu.RLock()
	mid := ackRequest.GetMessageId()
	klog.V(4).InfoS("Ack received for message", "Id", mid)
	for id, ack := range subAckMap {
		if id == mid {
			if ack.ackHandler != nil {
				ack.ackHandler(mid, err)
			}
			if ack.errCh != nil {
				ack.errCh <- err
			}
			break
		}
	}
	mu.RUnlock()
}

func (srv *rusiServerImpl) Publish(ctx context.Context, request *v1.PublishRequest) (*emptypb.Empty, error) {

	if request.Topic == "" {
		err := status.Errorf(codes.InvalidArgument, runtime.ErrTopicEmpty, request.PubsubName)
		klog.ErrorS(err, "missing topic")
		return &emptypb.Empty{}, err
	}

	metadata := request.GetMetadata()
	if metadata == nil {
		metadata = make(map[string]string)
	}

	var data interface{}
	err := serdes.Unmarshal(request.GetData(), &data)
	if err != nil {
		klog.ErrorS(err, "error unmarshalling", "payload", request.GetData())
		return &emptypb.Empty{}, err
	}

	err = srv.publishHandler(ctx, messaging.PublishRequest{
		PubsubName:      request.GetPubsubName(),
		Topic:           request.GetTopic(),
		DataContentType: request.GetDataContentType(),
		Data:            data,
		Type:            request.GetType(),
		Metadata:        metadata,
	})

	if err != nil {
		klog.ErrorS(err, "error on publishing")
		err = status.Errorf(codes.Unknown, err.Error())
	}
	return &emptypb.Empty{}, err
}

func messagingSubscriptionOptions(o *v1.SubscriptionOptions) *messaging.SubscriptionOptions {
	if o == nil {
		return nil
	}

	so := new(messaging.SubscriptionOptions)

	if o.Durable != nil {
		durable := o.Durable.Value
		so.Durable = &durable
	}
	if o.QGroup != nil {
		qGroup := o.QGroup.Value
		so.QGroup = &qGroup
	}
	if o.MaxConcurrentMessages != nil {
		maxConcurrentMessages := o.MaxConcurrentMessages.Value
		so.MaxConcurrentMessages = &maxConcurrentMessages
	}
	if o.DeliverNewMessagesOnly != nil {
		deliverNewMessagesOnly := o.DeliverNewMessagesOnly.Value
		so.DeliverNewMessagesOnly = &deliverNewMessagesOnly
	}
	if o.AckWaitTime != nil {
		duration := o.AckWaitTime.AsDuration()
		so.AckWaitTime = &duration
	}

	return so
}
