package grpc

import (
	"context"
	"errors"
	"fmt"
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
		subsWaitGroup:    &sync.WaitGroup{},
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
			//force stop streams
			grpcServer.Stop()
		}
	}()

	err = grpcServer.Serve(lis)
	//wait for unsubscribe
	srv.server.subsWaitGroup.Wait()
	return err
}

type rusiServerImpl struct {
	mu               sync.RWMutex
	mainCtx          context.Context
	subsWaitGroup    *sync.WaitGroup
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
	handler             func(buildCtx context.Context, pubsubName string, topic string) messaging.Handler
	closeFunc           messaging.CloseFunc
}

type subAck struct {
	ackHandler messaging.AckHandler
	errCh      chan error
}

// Subscribe creates a subscription
func (srv *rusiServerImpl) Subscribe(stream v1.Rusi_SubscribeServer) (err error) {
	if srv.subsWaitGroup != nil {
		srv.subsWaitGroup.Add(1)
		defer srv.subsWaitGroup.Done()
	}

	refreshChan := srv.createRefreshChan()
	defer srv.removeRefreshChan(refreshChan)
	subAckMap := map[string]*subAck{}
	ackLock := &sync.RWMutex{}
	subscriptions := map[string]*internalSubscription{}
	subsChan := make(chan *v1.SubscriptionRequest)

	innerHandler := buildSubscribeHandler(stream, subAckMap, ackLock)
	handleSubs := handleSubscriptionRequest(srv.subscribeHandler, subscriptions, innerHandler)
	handleAcks := handleAck(subAckMap, ackLock)

	go startStreamListener(stream, handleAcks, subsChan)
	hCtx, hCancel := context.WithCancel(context.Background())

	for {
		exit := false
		//blocks until done or refresh
		select {
		case <-srv.mainCtx.Done():
			exit = true
			err = srv.mainCtx.Err()
		case <-stream.Context().Done():
			exit = true
			err = stream.Context().Err()
		case subReq := <-subsChan:
			err = handleSubs(hCtx, subReq)
		case <-refreshChan:
			klog.V(4).InfoS("Refresh requested")
			hCancel()
			hCtx, hCancel = context.WithCancel(context.Background())
			err = handleRefreshSubscriptions(hCtx, srv.subscribeHandler, subscriptions)
		}

		if exit || err != nil {
			break
		}
	}

	hCancel()

	for _, s := range subscriptions {
		s.closeFunc()
	}
	return err
}

func startStreamListener(stream v1.Rusi_SubscribeServer,
	handleAck func(ack *v1.AckRequest) error,
	subsChan chan *v1.SubscriptionRequest) error {

	//wait for ack or subscriptionRequest from the client app
	for {
		select {
		case <-stream.Context().Done():
			klog.V(4).ErrorS(stream.Context().Err(), "stopping stream listener")
			return stream.Context().Err()
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
				go handleAck(r.GetAckRequest())
			}
			if r.GetSubscriptionRequest() != nil {
				//do subscribe in sync
				subsChan <- r.GetSubscriptionRequest()
			}
			if err != nil {
				return err
			}
		}
	}
}

func handleRefreshSubscriptions(ctx context.Context,
	subscribeHandler messaging.SubscribeRequestHandler,
	subscriptions map[string]*internalSubscription) error {

	for _, sub := range subscriptions {
		sub.closeFunc()

		unsub, err := subscribeHandler(ctx, messaging.SubscribeRequest{
			PubsubName: sub.pubsubName,
			Topic:      sub.topic,
			Handler:    sub.handler(ctx, sub.pubsubName, sub.topic),
			Options:    sub.subscriptionOptions,
		})
		sub.closeFunc = unsub

		if err != nil {
			return err
		}
	}
	return nil
}

func handleSubscriptionRequest(subscribeHandler messaging.SubscribeRequestHandler,
	subscriptions map[string]*internalSubscription,
	sendToGrpcHandler func(context.Context, string, string) messaging.Handler) func(context.Context, *v1.SubscriptionRequest) error {
	return func(ctx context.Context, request *v1.SubscriptionRequest) error {

		key := getMapKey(request.GetPubsubName(), request.GetTopic(), "")
		sub := &internalSubscription{
			subscriptionOptions: messagingSubscriptionOptions(request.GetOptions()),
			pubsubName:          request.GetPubsubName(),
			topic:               request.GetTopic(),
			handler:             sendToGrpcHandler,
		}

		unsub, err := subscribeHandler(ctx, messaging.SubscribeRequest{
			PubsubName: sub.pubsubName,
			Topic:      sub.topic,
			Handler:    sub.handler(ctx, sub.pubsubName, sub.topic),
			Options:    sub.subscriptionOptions,
		})
		sub.closeFunc = unsub
		subscriptions[key] = sub
		return err
	}
}

func buildSubscribeHandler(stream v1.Rusi_SubscribeServer, subAckMap map[string]*subAck,
	ackLock *sync.RWMutex) func(context.Context, string, string) messaging.Handler {

	return func(buildCtx context.Context, pubsubName, topic string) messaging.Handler {
		return func(ctx context.Context, env *messaging.MessageEnvelope) error {
			if env.Id == "" {
				return errors.New("message id is missing")
			}

			key := getMapKey(pubsubName, topic, env.Id)
			errChan := make(chan error)
			ackLock.Lock()
			subAckMap[key] = &subAck{nil, errChan}
			ackLock.Unlock()

			//send message to GRPC
			data, err := serdes.Marshal(env.Payload)
			if err != nil {
				return err
			}
			err = stream.Send(&v1.ReceivedMessage{
				Id:         env.Id,
				Data:       data,
				Topic:      topic,
				PubsubName: pubsubName,
				Metadata:   env.Headers,
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

func handleAck(subAckMap map[string]*subAck, mu *sync.RWMutex) func(ackRequest *v1.AckRequest) error {
	return func(ackRequest *v1.AckRequest) (err error) {
		if ackRequest.GetError() != "" {
			err = errors.New(ackRequest.GetError())
		}

		mid := ackRequest.GetMessageId()
		if mid == "" {
			err = errors.New("invalid ack : missing message id ")
		}
		if ackRequest.GetTopic() == "" {
			err = errors.New("invalid ack : missing topic ")
		}
		if ackRequest.GetPubsubName() == "" {
			err = errors.New("invalid ack : missing pubsubName")
		}

		klog.V(4).InfoS("Ack received for message", "Id", mid)
		key := getMapKey(ackRequest.GetPubsubName(), ackRequest.GetTopic(), mid)
		mu.RLock()
		ack, found := subAckMap[key]
		mu.RUnlock()

		if found {
			if ack.ackHandler != nil {
				ack.ackHandler(mid, err)
			}
			if ack.errCh != nil {
				ack.errCh <- err
			}
			//remove from list
			mu.Lock()
			delete(subAckMap, key)
			mu.Unlock()
		}

		return
	}
}

func getMapKey(pubsubName string, topic string, message_id string) string {
	return fmt.Sprintf("%s_%s_%s", pubsubName, topic, message_id)
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
