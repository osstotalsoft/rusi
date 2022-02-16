package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"k8s.io/klog/v2"
)

func SetupPrometheusMetrics(appId string) *prometheus.Exporter {
	config := prometheus.Config{DefaultHistogramBoundaries: []float64{10, 100, 1000, 10000, 100000}}
	r, _ := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithAttributes(semconv.ServiceNameKey.String(appId)))

	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
		controller.WithResource(r),
		controller.WithCollectPeriod(10*time.Second), //default - 10 sec
	)
	exporter, err := prometheus.New(config, c)
	if err != nil {
		klog.ErrorS(err, "failed to initialize prometheus exporter")
		return nil
	}
	global.SetMeterProvider(exporter.MeterProvider())
	return exporter
}
