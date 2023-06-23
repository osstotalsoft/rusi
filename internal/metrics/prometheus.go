package metrics

import (
	"go.opentelemetry.io/otel/exporters/prometheus"
	"k8s.io/klog/v2"
)

func SetupPrometheusMetrics(appId string) *prometheus.Exporter {
	exporter, err := prometheus.New()
	if err != nil {
		klog.ErrorS(err, "cannot create prometheus exporter ")
	}
	return exporter
}
