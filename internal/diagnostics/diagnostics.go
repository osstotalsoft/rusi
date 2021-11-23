package diagnostics

import (
	"context"
	"rusi/pkg/custom-resource/configuration"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"

	"k8s.io/klog/v2"
)

func WatchConfig(ctx context.Context, configLoader configuration_loader.ConfigurationLoader,
	tracerFunc func(url string, useAgent bool) (func(), error)) {

	var (
		prevConf       configuration.Spec
		tracingStopper func()
	)

	configChan, err := configLoader(ctx)
	if err != nil {
		klog.ErrorS(err, "error loading application config")
	}

	for cfg := range configChan {
		enabledAgent := !prevConf.TracingSpec.Jaeger.UseAgent && cfg.TracingSpec.Jaeger.UseAgent
		changedCollectorUrl := !cfg.TracingSpec.Jaeger.UseAgent && prevConf.TracingSpec.Zipkin.EndpointAddresss != cfg.TracingSpec.Zipkin.EndpointAddresss
		if enabledAgent || changedCollectorUrl {
			if tracingStopper != nil {
				//flush prev logs
				tracingStopper()
			}
			if enabledAgent || (changedCollectorUrl && cfg.TracingSpec.Zipkin.EndpointAddresss != "") {
				tracingStopper, err = tracerFunc(cfg.TracingSpec.Zipkin.EndpointAddresss, cfg.TracingSpec.Jaeger.UseAgent)
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
