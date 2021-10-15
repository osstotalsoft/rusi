package main

import (
	"context"
	"flag"
	"k8s.io/klog/v2"
	"rusi/internal/tracing"
	grpc_api "rusi/pkg/api/runtime/grpc"
	components_loader "rusi/pkg/custom-resource/components/loader"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
	"rusi/pkg/healthcheck"
	"rusi/pkg/modes"
	"rusi/pkg/operator"
	"rusi/pkg/runtime"
	"time"
)

func main() {
	mainCtx := context.Background()

	//https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md
	klog.InitFlags(nil)
	defer klog.Flush()

	cfgBuilder := runtime.NewRuntimeConfigBuilder()
	cfgBuilder.AttachCmdFlags(flag.StringVar, flag.BoolVar, flag.IntVar)
	flag.Parse()
	cfg, err := cfgBuilder.Build()
	if err != nil {
		klog.Error(err)
		return
	}
	compLoader := components_loader.LoadLocalComponents(cfg.ComponentsPath)
	configLoader := configuration_loader.LoadStandaloneConfiguration
	if cfg.Mode == modes.KubernetesMode {
		compLoader = operator.GetComponentsWatcher(cfg.ControlPlaneAddress)
		configLoader = operator.GetConfigurationWatcher(cfg.ControlPlaneAddress)
	}

	configChan, err := configLoader(mainCtx, cfg.Config)
	if err != nil {
		klog.Fatal(err)
	}

	//setup tracing
	go tracing.WatchConfig(mainCtx, configChan, tracing.SetJaegerTracing, "dev", cfg.AppID)

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

	//setup HealthzServer
	startHealthzServer(cfg.HealthzPort,
		// WithTimeout allows you to set a max overall timeout.
		healthcheck.WithTimeout(5*time.Second),
		healthcheck.WithChecker("component manager", compManager))

	err = rt.Run()
	if err != nil {
		klog.Error(err)
	}
}

func startHealthzServer(healthzPort int, options ...healthcheck.Option) {
	if err := healthcheck.Run(context.Background(), healthzPort, options...); err != nil {
		klog.Fatalf("failed to start healthz server: %s", err)
	}
}
