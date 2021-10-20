package tracing

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
	"time"
)

func SetTracing(tp trace.TracerProvider, propagator propagation.TextMapPropagator) {
	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
	//otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)
}

// FlushTracer cleanly shutdown and flush telemetry when the application exits.
func FlushTracer(tp *tracesdk.TracerProvider) func() {
	return func() {
		// Do not make the application hang when it is shutdown.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			klog.ErrorS(err, "Tracer shutdown error")
		}
	}
}
