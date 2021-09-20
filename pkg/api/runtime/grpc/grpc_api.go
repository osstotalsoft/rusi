package grpc

import (
	"context"
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

func NewGrpcAPI(port string, serverOptions ...grpc.ServerOption) runtime.Api {

	srv := &rusiServerImpl{
		refreshChannels:  []chan bool{},
		publishHandler:   nil,
		subscribeHandler: nil,
	}
	return &grpcApi{port, srv, serverOptions}
}

type grpcApi struct {
	port          string
	server        *rusiServerImpl
	serverOptions []grpc.ServerOption
}

func (srv *grpcApi) SetPublishHandler(publishHandler messaging.PublishRequestHandler) {
	srv.server.publishHandler = publishHandler
}
func (srv *grpcApi) SetSubscribeHandler(subscribeHandler messaging.SubscribeRequestHandler) {
	srv.server.subscribeHandler = subscribeHandler
}
func (srv *grpcApi) Refresh() error {
	return srv.server.Refresh()
}

func (srv *grpcApi) Serve() error {
	grpcServer := grpc.NewServer(srv.serverOptions...)
	v1.RegisterRusiServer(grpcServer, srv.server)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", srv.port))
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}

	return grpcServer.Serve(lis)
}

type rusiServerImpl struct {
	mu               sync.RWMutex
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

// Subscribe creates a subscription
func (srv *rusiServerImpl) Subscribe(request *v1.SubscribeRequest, subscribeServer v1.Rusi_SubscribeServer) error {
	ctx, cancel := context.WithCancel(subscribeServer.Context())
	defer cancel()
	exit := false
	refreshChan := srv.createRefreshChan()
	for {
		unsub, err := srv.subscribeHandler(ctx, messaging.SubscribeRequest{
			PubsubName: request.GetPubsubName(),
			Topic:      request.GetTopic(),
			Handler: func(_ context.Context, env *messaging.MessageEnvelope) error {
				data, err := serdes.Marshal(env.Payload)
				if err != nil {
					return err
				}
				return subscribeServer.Send(&v1.ReceivedMessage{
					Data:     data,
					Metadata: env.Headers,
				})
			},
			Options: messagingSubscriptionOptions(request.GetOptions()),
		})

		if err != nil {
			return err
		}

		//blocks until done or refresh
		select {
		case <-ctx.Done():
			exit = true
		case <-refreshChan:
			exit = false
		}
		err = unsub()
		if err != nil {
			return err
		}
		if exit {
			srv.removeRefreshChan(refreshChan)
			return ctx.Err()
		}
	}
}

func (srv *rusiServerImpl) Publish(ctx context.Context, request *v1.PublishRequest) (*emptypb.Empty, error) {

	if request.PubsubName == "" {
		err := status.Error(codes.InvalidArgument, runtime.ErrPubsubEmpty)
		klog.ErrorS(err, "missing pubsub name")
		return &emptypb.Empty{}, err
	}

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
		PubsubName: request.GetPubsubName(),
		Topic:      request.GetTopic(),
		Data:       data,
		Metadata:   metadata,
	})

	if err != nil {
		klog.V(4).Info(err)
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
