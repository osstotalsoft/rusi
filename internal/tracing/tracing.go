package tracing

import (
	"context"
	"fmt"
	"os"
	configuration "rusi/pkg/custom-resource/configuration"
	"strings"
	"time"

	jaeger_propagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"k8s.io/klog/v2"
)

func SetTracing(serviceName string) func(url string, propagator configuration.TelemetryPropagator) (func(), error) {
	return func(url string, propagator configuration.TelemetryPropagator) (func(), error) {
		tp, err := getTracerProvider(url, serviceName)
		if err != nil {
			return nil, err
		}
		// Register our TracerProvider as the global so any imported
		// instrumentation in the future will default to using it.
		otel.SetTracerProvider(tp)
		if propagator == configuration.TelemetryPropagatorJaeger {
			otel.SetTextMapPropagator(jaeger_propagator.Jaeger{})
		} else {
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
		}

		return func() {
			flushTracer(tp)
		}, nil
	}
}

// FlushTracer cleanly shutdown and flush telemetry when the application exits.
func flushTracer(tp *tracesdk.TracerProvider) func() {
	return func() {

		klog.V(4).InfoS("Trying to stop TracerProvider")

		// Do not make the application hang when it is shutdown.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*6)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			klog.ErrorS(err, "Tracer shutdown error")
		}
	}
}

func getTracerProvider(url string, serviceName string) (*tracesdk.TracerProvider, error) {
	// Set up a trace exporter
	url = strings.TrimPrefix(url, "http://")

	traceExporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(url),
		otlptracegrpc.WithInsecure())

	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	hostName, _ := os.Hostname()

	bsp := tracesdk.NewBatchSpanProcessor(traceExporter)
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.HostNameKey.String(hostName),
		)),
		tracesdk.WithSpanProcessor(bsp),
	)

	return tp, nil
}
