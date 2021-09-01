package main

import (
	"context"
	"flag"
	"k8s.io/klog/v2"
	"rusi/pkg/api/runtime/grpc"
	components_loader "rusi/pkg/custom-resource/components/loader"
	"rusi/pkg/custom-resource/components/middleware"
	"rusi/pkg/custom-resource/components/pubsub"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
	"rusi/pkg/kube"
	"rusi/pkg/messaging"
	natsstreaming "rusi/pkg/messaging/nats"
	middleware_pubsub "rusi/pkg/middleware/pubsub"
	"rusi/pkg/modes"
	"rusi/pkg/operator"
	"rusi/pkg/runtime"
	"strings"
)

func main() {
	//https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md
	klog.InitFlags(nil)
	kube.InitFlags(nil)
	defer klog.Flush()

	cfgBuilder := runtime.NewRuntimeConfigBuilder()
	cfgBuilder.AttachCmdFlags(flag.StringVar, flag.BoolVar)
	flag.Parse()
	cfg, err := cfgBuilder.Build()
	if err != nil {
		klog.Error(err)
		return
	}

	compLoader := components_loader.LoadLocalComponents(cfg.ComponentsPath)
	configLoader := configuration_loader.LoadStandaloneConfiguration
	if cfg.Mode == modes.KubernetesMode {
		compLoader = operator.ListComponents
		configLoader = operator.GetConfiguration
	}
	rt := runtime.NewRuntime(cfg, compLoader, configLoader)
	if rt == nil {
		return
	}
	rusiGrpcServer := grpc.NewRusiServer(rt.PublishHandler, rt.SubscribeHandler)
	api := grpc.NewGrpcAPI(rusiGrpcServer, cfg.RusiGRPCPort)
	err = rt.Load(
		runtime.WithPubSubs(
			pubsub.New("natsstreaming", func() messaging.PubSub {
				return natsstreaming.NewNATSStreamingPubSub()
			})),
		runtime.WithPubsubMiddleware(
			middleware.New("uppercase", func(properties map[string]string) middleware_pubsub.Middleware {
				return func(next middleware_pubsub.RequestHandler) middleware_pubsub.RequestHandler {
					return func(ctx context.Context, msg *messaging.MessageEnvelope) {
						body := msg.Payload.(string)
						msg.Payload = strings.ToUpper(body)
						next(ctx, msg)
					}
				}
			}),
		))
	if err != nil {
		klog.Error(err)
		return
	}

	klog.Infof("Rusid is starting on port %s", cfg.RusiGRPCPort)
	klog.Infof("Rusid selected mode %s", cfg.Mode)
	err = api.Serve()
	if err != nil {
		klog.Error(err)
	}
}
