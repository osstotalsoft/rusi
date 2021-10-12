// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// RusiOperatorClient is the client API for RusiOperator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RusiOperatorClient interface {
	WatchConfiguration(ctx context.Context, in *WatchConfigurationRequest, opts ...grpc.CallOption) (RusiOperator_WatchConfigurationClient, error)
	WatchComponents(ctx context.Context, in *WatchComponentsRequest, opts ...grpc.CallOption) (RusiOperator_WatchComponentsClient, error)
}

type rusiOperatorClient struct {
	cc grpc.ClientConnInterface
}

func NewRusiOperatorClient(cc grpc.ClientConnInterface) RusiOperatorClient {
	return &rusiOperatorClient{cc}
}

func (c *rusiOperatorClient) WatchConfiguration(ctx context.Context, in *WatchConfigurationRequest, opts ...grpc.CallOption) (RusiOperator_WatchConfigurationClient, error) {
	stream, err := c.cc.NewStream(ctx, &RusiOperator_ServiceDesc.Streams[0], "/rusi.proto.operator.v1.RusiOperator/WatchConfiguration", opts...)
	if err != nil {
		return nil, err
	}
	x := &rusiOperatorWatchConfigurationClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type RusiOperator_WatchConfigurationClient interface {
	Recv() (*GenericItem, error)
	grpc.ClientStream
}

type rusiOperatorWatchConfigurationClient struct {
	grpc.ClientStream
}

func (x *rusiOperatorWatchConfigurationClient) Recv() (*GenericItem, error) {
	m := new(GenericItem)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *rusiOperatorClient) WatchComponents(ctx context.Context, in *WatchComponentsRequest, opts ...grpc.CallOption) (RusiOperator_WatchComponentsClient, error) {
	stream, err := c.cc.NewStream(ctx, &RusiOperator_ServiceDesc.Streams[1], "/rusi.proto.operator.v1.RusiOperator/WatchComponents", opts...)
	if err != nil {
		return nil, err
	}
	x := &rusiOperatorWatchComponentsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type RusiOperator_WatchComponentsClient interface {
	Recv() (*GenericItem, error)
	grpc.ClientStream
}

type rusiOperatorWatchComponentsClient struct {
	grpc.ClientStream
}

func (x *rusiOperatorWatchComponentsClient) Recv() (*GenericItem, error) {
	m := new(GenericItem)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// RusiOperatorServer is the server API for RusiOperator service.
// All implementations should embed UnimplementedRusiOperatorServer
// for forward compatibility
type RusiOperatorServer interface {
	WatchConfiguration(*WatchConfigurationRequest, RusiOperator_WatchConfigurationServer) error
	WatchComponents(*WatchComponentsRequest, RusiOperator_WatchComponentsServer) error
}

// UnimplementedRusiOperatorServer should be embedded to have forward compatible implementations.
type UnimplementedRusiOperatorServer struct {
}

func (UnimplementedRusiOperatorServer) WatchConfiguration(*WatchConfigurationRequest, RusiOperator_WatchConfigurationServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchConfiguration not implemented")
}
func (UnimplementedRusiOperatorServer) WatchComponents(*WatchComponentsRequest, RusiOperator_WatchComponentsServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchComponents not implemented")
}

// UnsafeRusiOperatorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RusiOperatorServer will
// result in compilation errors.
type UnsafeRusiOperatorServer interface {
	mustEmbedUnimplementedRusiOperatorServer()
}

func RegisterRusiOperatorServer(s grpc.ServiceRegistrar, srv RusiOperatorServer) {
	s.RegisterService(&RusiOperator_ServiceDesc, srv)
}

func _RusiOperator_WatchConfiguration_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchConfigurationRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(RusiOperatorServer).WatchConfiguration(m, &rusiOperatorWatchConfigurationServer{stream})
}

type RusiOperator_WatchConfigurationServer interface {
	Send(*GenericItem) error
	grpc.ServerStream
}

type rusiOperatorWatchConfigurationServer struct {
	grpc.ServerStream
}

func (x *rusiOperatorWatchConfigurationServer) Send(m *GenericItem) error {
	return x.ServerStream.SendMsg(m)
}

func _RusiOperator_WatchComponents_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchComponentsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(RusiOperatorServer).WatchComponents(m, &rusiOperatorWatchComponentsServer{stream})
}

type RusiOperator_WatchComponentsServer interface {
	Send(*GenericItem) error
	grpc.ServerStream
}

type rusiOperatorWatchComponentsServer struct {
	grpc.ServerStream
}

func (x *rusiOperatorWatchComponentsServer) Send(m *GenericItem) error {
	return x.ServerStream.SendMsg(m)
}

// RusiOperator_ServiceDesc is the grpc.ServiceDesc for RusiOperator service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RusiOperator_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rusi.proto.operator.v1.RusiOperator",
	HandlerType: (*RusiOperatorServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "WatchConfiguration",
			Handler:       _RusiOperator_WatchConfiguration_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "WatchComponents",
			Handler:       _RusiOperator_WatchComponents_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/operator/v1/rusi_operator.proto",
}
