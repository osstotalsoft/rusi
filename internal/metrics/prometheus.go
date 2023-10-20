package metrics

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"k8s.io/klog/v2"
)

func SetupPrometheusMetrics(appId string) *prometheus.Exporter {
	exporter, err := prometheus.New()
	if err != nil {
		klog.ErrorS(err, "failed to initialize prometheus exporter")
		return nil
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	return exporter
}
