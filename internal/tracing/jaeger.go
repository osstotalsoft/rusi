package tracing

import (
	jaeger_propagators "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/attribute"
	jaeger_exporters "go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// JaegerTracerProvider returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func JaegerTracerProvider(url string, useAgent bool, environment, serviceName string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	var exp *jaeger_exporters.Exporter
	var err error

	if useAgent {
		exp, err = jaeger_exporters.New(jaeger_exporters.WithAgentEndpoint())
	} else {
		exp, err = jaeger_exporters.New(jaeger_exporters.WithCollectorEndpoint(jaeger_exporters.WithEndpoint(url)))
	}
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", environment),
			//attribute.Int64("ID", id),
		)),
	)
	return tp, nil
}

func SetJaegerTracing(environment, serviceName string) func(url string, useAgent bool) (func(), error) {
	return func(url string, useAgent bool) (func(), error) {
		tp, err := JaegerTracerProvider(url, useAgent, environment, serviceName)
		if err != nil {
			return nil, err
		}
		SetTracing(tp, jaeger_propagators.Jaeger{})

		return func() {
			FlushTracer(tp)
		}, nil
	}
}
