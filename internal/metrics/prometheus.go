package metrics

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"k8s.io/klog/v2"
)

func SetupPrometheusMetrics(appId string) *prometheus.Exporter {
	exporter, err := prometheus.New()
	if err != nil {
		klog.ErrorS(err, "failed to initialize prometheus exporter")
		return nil
	}

	r, _ := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithAttributes(semconv.ServiceNameKey.String(appId)))

	provider := metric.NewMeterProvider(metric.WithResource(r), metric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	return exporter
}
