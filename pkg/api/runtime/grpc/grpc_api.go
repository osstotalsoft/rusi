package grpc

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/klog/v2"
	"net"
	"rusi/pkg/messaging"
	v1 "rusi/pkg/proto/runtime/v1"
)

func NewGrpcAPI(server v1.RusiServer, port string) *grpcApi {
	return &grpcApi{port, server}
}

type grpcApi struct {
	port   string
	server v1.RusiServer
}

func (srv *grpcApi) Publish(env *messaging.MessageEnvelope) error {
	_, err := srv.server.Publish(context.Background(), &v1.PublishRequest{
		PubsubName:      "",
		Topic:           "",
		Data:            nil,
		DataContentType: "",
		Metadata:        nil,
	})

	return err
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

func NewRusiServer() v1.RusiServer {
	return &server{}
}

type server struct {
}

func (srv *server) Publish(ctx context.Context, request *v1.PublishRequest) (*emptypb.Empty, error) {
	return nil, nil
}
