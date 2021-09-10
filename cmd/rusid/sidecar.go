package main

import (
	"context"
	"flag"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"k8s.io/klog/v2"
	"rusi/internal/tracing"
	grpc_api "rusi/pkg/api/runtime/grpc"
	components_loader "rusi/pkg/custom-resource/components/loader"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
	"rusi/pkg/kube"
	"rusi/pkg/modes"
	"rusi/pkg/operator"
	"rusi/pkg/runtime"
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

	dynamicConfig, err := configLoader(cfg.Config)
	if err != nil {
		klog.Fatal(err)
	}

	if dynamicConfig.TracingSpec.Zipkin.EndpointAddresss != "" {
		tp, err := tracing.JaegerTracerProvider(dynamicConfig.TracingSpec.Zipkin.EndpointAddresss,
			"dev", cfg.AppID)
		if err != nil {
			klog.Fatal(err)
		}
		tracing.SetTracing(tp, jaeger.Jaeger{})
		defer tracing.FlushTracer(tp)(context.Background())
	}

	rt := runtime.NewRuntime(cfg, compLoader, configLoader)
	if rt == nil {
		return
	}
	rusiGrpcServer := grpc_api.NewRusiServer(rt.PublishHandler, rt.SubscribeHandler)
	api := grpc_api.NewGrpcAPI(rusiGrpcServer, cfg.RusiGRPCPort)
	err = rt.Load(RegisterComponentFactories()...)
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
