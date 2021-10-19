package main

import (
	"context"
	"flag"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"os/signal"
	"rusi/internal/diagnostics"
	"rusi/internal/metrics"
	"rusi/internal/tracing"
	grpc_api "rusi/pkg/api/runtime/grpc"
	components_loader "rusi/pkg/custom-resource/components/loader"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
	"rusi/pkg/healthcheck"
	"rusi/pkg/modes"
	"rusi/pkg/operator"
	"rusi/pkg/runtime"
	"sync"
	"time"
)

func main() {
	mainCtx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

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
	configLoader := configuration_loader.LoadStandaloneConfiguration(cfg.Config)
	if cfg.Mode == modes.KubernetesMode {
		compLoader = operator.GetComponentsWatcher(mainCtx, cfg.ControlPlaneAddress, wg)
		configLoader = operator.GetConfigurationWatcher(mainCtx, cfg.ControlPlaneAddress, cfg.Config, wg)
	}

	//setup tracing
	go diagnostics.WatchConfig(mainCtx, configLoader,
		tracing.SetJaegerTracing("dev", cfg.AppID),
		metrics.GetPrometheusExporter)

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

	//Start diagnostics server
	go startDiagnosticsServer(mainCtx, wg, cfg.DiagnosticsPort,
		// WithTimeout allows you to set a max overall timeout.
		healthcheck.WithTimeout(5*time.Second),
		healthcheck.WithChecker("component manager", compManager))

	shutdownOnInterrupt(cancel)

	err = rt.Run(mainCtx) //blocks
	if err != nil {
		klog.Error(err)
	}

	wg.Wait() // wait for app to close gracefully
}

func shutdownOnInterrupt(cancel func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		klog.InfoS("Shutdown requested")
		cancel()
	}()

}

func startDiagnosticsServer(ctx context.Context, wg *sync.WaitGroup, healthzPort int, options ...healthcheck.Option) {
	wg.Add(1)
	defer wg.Done()

	router := http.NewServeMux()
	router.Handle("/healthz", healthcheck.HandlerFunc(options...))
	router.Handle("/metrics", metrics.GetPrometheusMetricHandler())

	if err := diagnostics.Run(ctx, healthzPort, router); err != nil {
		if err != http.ErrServerClosed {
			klog.ErrorS(err, "failed to start diagnostics server")
		}
	}
}
