package diagnostics

import (
	"context"
	"rusi/pkg/custom-resource/configuration"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"

	"k8s.io/klog/v2"
)

func WatchConfig(ctx context.Context, configLoader configuration_loader.ConfigurationLoader,
	tracerFunc func(url string, propagator configuration.TelemetryPropagator) (func(), error)) {

	var (
		prevConf       configuration.Spec
		tracingStopper func()
	)

	configChan, err := configLoader(ctx)
	if err != nil {
		klog.ErrorS(err, "error loading application config")
	}

	for cfg := range configChan {
		changed := cfg.Telemetry.CollectorEndpoint != prevConf.Telemetry.CollectorEndpoint || cfg.Telemetry.Tracing != prevConf.Telemetry.Tracing
		validConfig := cfg.Telemetry.CollectorEndpoint != "" && cfg.Telemetry.Tracing.Propagator != ""
		if changed {
			if tracingStopper != nil {
				//flush prev logs
				tracingStopper()
			}
			if validConfig {
				tracingStopper, err = tracerFunc(cfg.Telemetry.CollectorEndpoint, cfg.Telemetry.Tracing.Propagator)
				if err != nil {
					klog.ErrorS(err, "error creating tracer")
				}
			}
		}

		prevConf = cfg
	}

	if tracingStopper != nil {
		//flush logs
		tracingStopper()
	}
}
