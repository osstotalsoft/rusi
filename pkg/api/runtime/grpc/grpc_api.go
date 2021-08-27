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

func NewGrpcAPI(server v1.RusiServer, port string) runtime.Api {
	return &grpcApi{port, server}
}

type grpcApi struct {
	port   string
	server v1.RusiServer
}

func (srv *grpcApi) SendMessageToApp(env *messaging.MessageEnvelope) error {
	return nil
}

func (srv *grpcApi) Serve() error {
	grpcServer := grpc.NewServer()
	v1.RegisterRusiServer(grpcServer, srv.server)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", srv.port))
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}

	return grpcServer.Serve(lis)
}

func NewRusiServer(getPublisherFunc func(pubsubName string) messaging.Publisher,
	getSubscriberFunc func(pubsubName string) messaging.Subscriber) v1.RusiServer {
	return &server{getPublisherFunc, getSubscriberFunc}
}

type server struct {
	getPublisherFunc  func(pubsubName string) messaging.Publisher
	getSubscriberFunc func(pubsubName string) messaging.Subscriber
}

func (srv *server) Subscribe(request *v1.SubscribeRequest, subscribeServer v1.Rusi_SubscribeServer) error {
	ctx, cancel := context.WithCancel(subscribeServer.Context())
	defer cancel()

	unsub, err := srv.getSubscriberFunc(request.PubsubName).
		Subscribe(request.Topic, func(env *messaging.MessageEnvelope) error {
			data, err := serdes.Marshal(env.Payload)
			if err != nil {
				return err
			}
			err = subscribeServer.Send(&v1.ReceivedMessage{
				Data:     data,
				Metadata: env.Headers,
			})
			if err != nil {
				klog.ErrorS(err, "oops")
			}
			return err
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

	publisher := srv.getPublisherFunc(request.PubsubName)
	if publisher == nil {
		err := status.Errorf(codes.InvalidArgument, runtime.ErrPubsubNotFound, request.PubsubName)
		klog.V(4).Info(err)
		return &emptypb.Empty{}, err
	}

	if request.Topic == "" {
		err := status.Errorf(codes.InvalidArgument, runtime.ErrTopicEmpty, request.PubsubName)
		klog.V(4).Info(err)
		return &emptypb.Empty{}, err
	}

	err := publisher.Publish(request.Topic, &messaging.MessageEnvelope{
		Headers: request.Metadata,
		Payload: string(request.Data),
	})

	return &emptypb.Empty{}, err
}
