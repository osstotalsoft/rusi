package main

import (
	"flag"
	"k8s.io/klog/v2"
	runtime_grpc "rusi/pkg/api/grpc"
	"rusi/pkg/runtime"
)

func main() {

	klog.InitFlags(flag.CommandLine)
	defer klog.Flush()

	cfg := runtime.NewRuntimeConfig()
	cfg.AttachCmdFlags(flag.StringVar, flag.BoolVar)

	flag.Parse()

	rusiService := runtime_grpc.NewRusiServer()
	api := runtime_grpc.NewGrpcAPI(rusiService, cfg.RusiGRPCPort)
	rt := runtime.NewRuntime(cfg, api)

	klog.Infof("Rusid is starting on port %s", cfg.RusiGRPCPort)

	if err := rt.Run(); err != nil {
		klog.Error(err)
	}
}
