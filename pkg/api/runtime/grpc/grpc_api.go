package grpc

import (
	"context"
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
)

func NewGrpcAPI(server v1.RusiServer, port string, serverOptions ...grpc.ServerOption) runtime.Api {
	return &grpcApi{port, server, serverOptions}
}

type grpcApi struct {
	port          string
	server        v1.RusiServer
	serverOptions []grpc.ServerOption
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

func NewRusiServer(publishHandler func(request messaging.PublishRequest) error,
	subscribeHandler func(request messaging.SubscribeRequest) (messaging.UnsubscribeFunc, error)) v1.RusiServer {
	return &server{publishHandler, subscribeHandler}
}

type server struct {
	publishHandler   func(request messaging.PublishRequest) error
	subscribeHandler func(request messaging.SubscribeRequest) (messaging.UnsubscribeFunc, error)
}

// Subscribe creates a subscription
func (srv *server) Subscribe(request *v1.SubscribeRequest, subscribeServer v1.Rusi_SubscribeServer) error {
	ctx, cancel := context.WithCancel(subscribeServer.Context())
	defer cancel()

	unsub, err := srv.subscribeHandler(messaging.SubscribeRequest{
		PubsubName: request.GetPubsubName(),
		Topic:      request.GetTopic(),
		Handler: func(env *messaging.MessageEnvelope) error {
			data, err := serdes.Marshal(env)
			if err != nil {
				return err
			}
			return subscribeServer.Send(&v1.ReceivedMessage{
				Data:     data,
				Metadata: env.Headers,
			})
		},
	})

	if err != nil {
		return err
	}
	defer unsub()

	<-ctx.Done()
	return ctx.Err()
}

func (srv *server) Publish(ctx context.Context, request *v1.PublishRequest) (*emptypb.Empty, error) {

	if request.PubsubName == "" {
		err := status.Error(codes.InvalidArgument, runtime.ErrPubsubEmpty)
		klog.V(4).Info(err)
		return &emptypb.Empty{}, err
	}

	if request.Topic == "" {
		err := status.Errorf(codes.InvalidArgument, runtime.ErrTopicEmpty, request.PubsubName)
		klog.V(4).Info(err)
		return &emptypb.Empty{}, err
	}

	err := srv.publishHandler(messaging.PublishRequest{
		PubsubName: request.GetPubsubName(),
		Topic:      request.GetTopic(),
		Data:       request.GetData(),
		Metadata:   request.GetMetadata(),
	})

	if err != nil {
		klog.V(4).Info(err)
		err = status.Errorf(codes.Unknown, err.Error())
	}
	return &emptypb.Empty{}, err
}
