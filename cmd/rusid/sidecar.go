package main

import (
	"flag"
	"k8s.io/klog/v2"
	"rusi/pkg/api/runtime/grpc"
	"rusi/pkg/components/pubsub"
	"rusi/pkg/kube"
	"rusi/pkg/messaging"
	natsstreaming "rusi/pkg/messaging/nats"
	"rusi/pkg/operator"
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
	if err := cfg.Validate(); err != nil {
		klog.Error(err)
		return
	}

	compProviderFunc := operator.ListComponents

	pubsubFactory := pubsub.NewPubSubFactory(cfg.AppID)
	rusiGrpcServer := grpc.NewRusiServer(pubsubFactory.GetPublisher)
	api := grpc.NewGrpcAPI(rusiGrpcServer, cfg.RusiGRPCPort)
	rt := runtime.NewRuntime(cfg, api, compProviderFunc, pubsubFactory)

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
