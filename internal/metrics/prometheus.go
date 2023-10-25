package metrics

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"k8s.io/klog/v2"
)

func SetupPrometheusMetrics(appId string) *prometheus.Exporter {
	exporter, err := prometheus.New(prometheus.WithoutScopeInfo())
	if err != nil {
		klog.ErrorS(err, "failed to initialize prometheus exporter")
		return nil
	}

	r, _ := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithAttributes(semconv.ServiceName(appId)))

	r, _ = resource.Merge(resource.Default(), r)

	provider := metric.NewMeterProvider(metric.WithResource(r),
		metric.WithReader(exporter),
		//metric.WithView(metric.NewView(
		//	metric.Instrument{
		//		Name: "rusi.pubsub.processing.duration",
		//	},
		//	metric.Stream{
		//		Aggregation: metric.AggregationExplicitBucketHistogram{
		//			Boundaries: []float64{10, 100, 1000, 5000, 10000, 20000, 100000},
		//		},
		//	})),
	)

	otel.SetMeterProvider(provider)

	return exporter
}
