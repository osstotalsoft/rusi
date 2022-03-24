package grpc

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/klog/v2"
	"net"
	"rusi/pkg/api/runtime"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	v1 "rusi/pkg/proto/runtime/v1"
	"sync"
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
	srv.server = newRusiServer(ctx, srv.publishHandler, srv.subscribeHandler)
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

func newRusiServer(ctx context.Context,
	publishHandler messaging.PublishRequestHandler,
	subscribeHandler messaging.SubscribeRequestHandler) *rusiServerImpl {
	return &rusiServerImpl{
		refreshChannels:  []chan bool{},
		mainCtx:          ctx,
		subsWaitGroup:    &sync.WaitGroup{},
		publishHandler:   publishHandler,
		subscribeHandler: subscribeHandler,
	}
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

// Subscribe creates a subscription
func (srv *rusiServerImpl) Subscribe(stream v1.Rusi_SubscribeServer) error {
	srv.subsWaitGroup.Add(1)
	defer srv.subsWaitGroup.Done()

	//block until subscriptionRequest is received
	r, err := stream.Recv()
	if err != nil {
		return err
	}
	request := r.GetSubscriptionRequest()
	if request == nil {
		return errors.New("invalid subscription request")
	}

	exit := false
	refreshChan := srv.createRefreshChan()
	defer srv.removeRefreshChan(refreshChan)
	handler := srv.buildSubscribeHandler(stream)
	for {
		hCtx, hCancel := context.WithCancel(context.Background())
		unsub, err := srv.subscribeHandler(hCtx, messaging.SubscribeRequest{
			PubsubName: request.GetPubsubName(),
			Topic:      request.GetTopic(),
			Handler:    handler(hCtx),
			Options:    messagingSubscriptionOptions(request.GetOptions()),
		})

		if err != nil {
			hCancel()
			return err
		}

		//blocks until done or refresh
		select {
		case <-srv.mainCtx.Done():
			exit = true
			err = srv.mainCtx.Err()
		case <-stream.Context().Done():
			exit = true
			err = stream.Context().Err()
		case <-refreshChan:
			exit = false
			klog.V(4).InfoS("Refresh requested for", "topic", request.Topic)
		}
		hCancel()
		//ignore unsubscribe error
		_ = unsub()
		if exit {
			klog.V(4).InfoS("closing subscription stream", "topic", request.Topic, "error", err)
			return err
		}
	}
}

type subAck struct {
	ackHandler messaging.AckHandler
	errCh      chan error
}

func (srv *rusiServerImpl) buildSubscribeHandler(stream v1.Rusi_SubscribeServer) func(context.Context) messaging.Handler {

	subAckMap := map[string]*subAck{}
	mu := &sync.RWMutex{}

	//monitor incoming ack stream for the current subscription
	go startAckReceiverForStream(subAckMap, mu, stream)

	return func(buildCtx context.Context) messaging.Handler {
		return func(ctx context.Context, env *messaging.MessageEnvelope) error {
			if env.Id == "" {
				return errors.New("message id is missing")
			}

			errChan := make(chan error)
			mu.Lock()
			subAckMap[env.Id] = &subAck{nil, errChan}
			mu.Unlock()
			//cleanup
			defer func() {
				mu.Lock()
				delete(subAckMap, env.Id)
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

func startAckReceiverForStream(subAckMap map[string]*subAck, mu *sync.RWMutex, stream v1.Rusi_SubscribeServer) {

	//wait for ack from the client
	for {
		select {
		case <-stream.Context().Done():
			//klog.V(4).ErrorS(stream.Context().Err(), "stopping ack stream watcher")
			return
		default:
			r, err := stream.Recv() //blocks
			if err != nil {
				klog.V(4).ErrorS(err, "ack stream error")
				break
			}
			if r.GetAckRequest() == nil {
				klog.V(4).InfoS("invalid ack response")
				break
			}
			if r.GetAckRequest().GetError() != "" {
				err = errors.New(r.GetAckRequest().GetError())
			}

			mu.RLock()
			mid := r.GetAckRequest().GetMessageId()
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
	}
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
