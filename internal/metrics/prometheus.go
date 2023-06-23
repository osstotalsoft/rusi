package metrics

import (
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"k8s.io/klog/v2"
)

func SetupPrometheusMetrics(appId string) *prometheus.Exporter {
	exporter, err := prometheus.New()
	if err != nil {
		klog.ErrorS(err, "cannot create prometheus exporter ")
	}
	return exporter
}

func CustomAggregationSelector(ik metric.InstrumentKind) aggregation.Aggregation {
	switch ik {
	case metric.InstrumentKindHistogram:
		return aggregation.ExplicitBucketHistogram{
			Boundaries: []float64{10, 100, 1000, 10000, 100000},
			NoMinMax:   false,
		}
	}
	return metric.DefaultAggregationSelector(ik)
}
