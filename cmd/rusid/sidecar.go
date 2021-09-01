package main

import (
	"flag"
	"k8s.io/klog/v2"
	"rusi/pkg/api/runtime/grpc"
	"rusi/pkg/custom-resource/components/pubsub"
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
	//compProviderFunc := localMachine.ListComponents
	rt := runtime.NewRuntime(cfg, compProviderFunc)
	rusiGrpcServer := grpc.NewRusiServer(rt.PublishHandler, rt.SubscribeHandler)
	api := grpc.NewGrpcAPI(rusiGrpcServer, cfg.RusiGRPCPort)
	err := rt.Load(
		runtime.WithPubSubs(
			pubsub.New("natsstreaming", func() messaging.PubSub {
				return natsstreaming.NewNATSStreamingPubSub()
			}),
		))
	if err != nil {
		klog.Error(err)
		return
	}

	klog.Infof("Rusid is starting on port %s", cfg.RusiGRPCPort)
	err = api.Serve()
	if err != nil {
		klog.Error(err)
	}
}
