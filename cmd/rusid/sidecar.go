package main

import (
	"context"
	"flag"
	"k8s.io/klog/v2"
	grpc_api "rusi/pkg/api/runtime/grpc"
	components_loader "rusi/pkg/custom-resource/components/loader"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
	"rusi/pkg/kube"
	"rusi/pkg/modes"
	"rusi/pkg/operator"
	"rusi/pkg/runtime"
)

func main() {
	mainCtx := context.Background()

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

	//dynamicConfig, err := configLoader(mainCtx, cfg.Config)
	//if err != nil {
	//	klog.Fatal(err)
	//}
	//
	//if dynamicConfig.TracingSpec.Zipkin.EndpointAddresss != "" {
	//	tp, err := tracing.JaegerTracerProvider(dynamicConfig.TracingSpec.Zipkin.EndpointAddresss,
	//		"dev", cfg.AppID)
	//	if err != nil {
	//		klog.Fatal(err)
	//	}
	//	tracing.SetTracing(tp, jaeger.Jaeger{})
	//	defer tracing.FlushTracer(tp)(mainCtx)
	//}

	compManager, err := runtime.NewComponentsManager(mainCtx, cfg.AppID, compLoader,
		RegisterComponentFactories()...)
	if err != nil {
		klog.Error(err)
		return
	}
	api := grpc_api.NewGrpcAPI(cfg.RusiGRPCPort)
	rt, err := runtime.NewRuntime(mainCtx, cfg, api, configLoader, compManager)
	if err != nil {
		klog.Error(err)
		return
	}

	klog.InfoS("Rusid is starting", "port", cfg.RusiGRPCPort,
		"app id", cfg.AppID, "mode", cfg.Mode)
	klog.InfoS("Rusid is using", "config", cfg)

	err = rt.Run()
	if err != nil {
		klog.Error(err)
	}
}
