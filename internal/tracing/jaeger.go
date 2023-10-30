package tracing

import (
	"context"
	"fmt"
	"os"

	jaeger_propagators "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// JaegerTracerProvider returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func jaegerTracerProvider(url string, useAgent bool, serviceName string) (*tracesdk.TracerProvider, error) {
	// Set up a trace exporter
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

func SetJaegerTracing(serviceName string) func(url string, useAgent bool) (func(), error) {
	return func(url string, useAgent bool) (func(), error) {
		tp, err := jaegerTracerProvider(url, useAgent, serviceName)
		if err != nil {
			return nil, err
		}
		SetTracing(tp, jaeger_propagators.Jaeger{})

		return func() {
			FlushTracer(tp)
		}, nil
	}
}
