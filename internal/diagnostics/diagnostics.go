package diagnostics

import (
	"context"
	"k8s.io/klog/v2"
	"rusi/pkg/custom-resource/configuration"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
)

func WatchConfig(ctx context.Context, configLoader configuration_loader.ConfigurationLoader,
	tracerFunc func(url string) (func(), error)) {

	var (
		prevConf       configuration.Spec
		tracingStopper func()
	)

	configChan, err := configLoader(ctx)
	if err != nil {
		klog.ErrorS(err, "error loading application config")
	}

	for cfg := range configChan {
		if prevConf.TracingSpec.Zipkin.EndpointAddresss != cfg.TracingSpec.Zipkin.EndpointAddresss {
			if tracingStopper != nil {
				//flush prev logs
				tracingStopper()
			}
			if cfg.TracingSpec.Zipkin.EndpointAddresss != "" {
				tracingStopper, err = tracerFunc(cfg.TracingSpec.Zipkin.EndpointAddresss)
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
