package main

import (
	"flag"
	"k8s.io/klog/v2"
	"rusi/pkg/api/runtime/grpc"
	"rusi/pkg/components/pubsub"
	"rusi/pkg/kube"
	"rusi/pkg/messaging"
	natsstreaming "rusi/pkg/messaging/nats"
	"rusi/pkg/runtime"
)

func main() {
	//https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md
	klog.InitFlags(nil)
	kube.InitFlags(nil)
	defer klog.Flush()

	cfg := runtime.NewRuntimeConfig()
	cfg.AttachCmdFlags(flag.StringVar, flag.BoolVar)
	flag.Parse()

	rusiService := grpc.NewRusiServer()
	api := grpc.NewGrpcAPI(rusiService, cfg.RusiGRPCPort)
	rt := runtime.NewRuntime(cfg, api)

	err := rt.ConfigureOptions(
		runtime.WithPubSubs(
			pubsub.New("natsstreaming", func() messaging.PubSub {
				return natsstreaming.NewNATSStreamingPubSub()
			}),
		))
	if err != nil {
		klog.Error(err)
	}

	klog.Infof("Rusid is starting on port %s", cfg.RusiGRPCPort)
	err = api.Serve()
	if err != nil {
		klog.Error(err)
	}
}
