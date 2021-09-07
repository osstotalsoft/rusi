package tracing

import (
	"context"
	"go.opentelemetry.io/otel"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
	"time"
)

func SetDefaultTracerProvider(tp trace.TracerProvider) {
	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
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
