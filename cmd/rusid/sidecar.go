package main

import (
	"flag"
	"k8s.io/klog/v2"
	"rusi/pkg/api/grpc"
	"rusi/pkg/components/pubsub"
	"rusi/pkg/messaging"
	natsstreaming "rusi/pkg/messaging/nats"
	"rusi/pkg/runtime"
)

func main() {

	klog.InitFlags(flag.CommandLine)
	defer klog.Flush()

	cfg := runtime.NewRuntimeConfig()
	cfg.AttachCmdFlags(flag.StringVar, flag.BoolVar)

	flag.Parse()

	rusiService := grpc.NewRusiServer()
	api := grpc.NewGrpcAPI(rusiService, cfg.RusiGRPCPort)
	rt := runtime.NewRuntime(cfg, api)

	klog.Infof("Rusid is starting on port %s", cfg.RusiGRPCPort)
	err := rt.Run(
		runtime.WithPubSubs(
			pubsub.New("natsstreaming", func() messaging.PubSub {
				return natsstreaming.NewNATSStreamingPubSub()
			}),
		))
	if err != nil {
		klog.Error(err)
	}
}
