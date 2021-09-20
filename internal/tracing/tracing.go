package tracing

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
	"rusi/pkg/custom-resource/configuration"
	"time"
)

func WatchConfig(mainCtx context.Context, configChan <-chan configuration.Spec,
	tracerFunc func(url, environment, serviceName string) (*tracesdk.TracerProvider, error),
	environment, serviceName string) {

	var (
		err                  error
		prevEndpointAddresss string
		tp                   *tracesdk.TracerProvider
	)
	for cfg := range configChan {
		if prevEndpointAddresss == cfg.TracingSpec.Zipkin.EndpointAddresss {
			continue
		}
		if tp != nil {
			//flush prev logs
			FlushTracer(tp)(mainCtx)
		}
		if cfg.TracingSpec.Zipkin.EndpointAddresss != "" {
			tp, err = tracerFunc(cfg.TracingSpec.Zipkin.EndpointAddresss, environment, serviceName)
			if err != nil {
				klog.Fatal(err)
			}
		}
		prevEndpointAddresss = cfg.TracingSpec.Zipkin.EndpointAddresss
	}
	if tp != nil {
		//flush prev logs
		FlushTracer(tp)(mainCtx)
	}
}

func SetTracing(tp trace.TracerProvider, propagator propagation.TextMapPropagator) {
	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
	//otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)
}

// FlushTracer cleanly shutdown and flush telemetry when the application exits.
func FlushTracer(tp *tracesdk.TracerProvider) func(ctx context.Context) {
	return func(ctx context.Context) {
		// Do not make the application hang when it is shutdown.
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			klog.Fatal(err)
		}
	}
}
