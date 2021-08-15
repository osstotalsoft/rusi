package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"net"
	"rusi/pkg/runtime"
)

func main() {

	klog.InitFlags(flag.CommandLine)

	cfg := runtime.NewRuntimeConfig()
	cfg.AttachCmdFlags(flag.StringVar, flag.BoolVar)

	flag.Parse()

	rt := runtime.NewRuntime(cfg)

	klog.Infof("Rusid is starting on port %s", rt.Config.RusiGRPCPort)
	defer klog.Flush()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", rt.Config.RusiGRPCPort))
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	if err = grpcServer.Serve(lis); err != nil {
		klog.Error(err)
	}
}
