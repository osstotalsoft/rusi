package metrics

import (
	"context"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	sdk_metric "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"k8s.io/klog/v2"
	"time"
)

func SetupPrometheusMetrics(appId string) *prometheus.Exporter {
	config := prometheus.Config{}
	r, _ := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithAttributes(semconv.ServiceNameKey.String(appId)))

	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			sdk_metric.CumulativeExportKindSelector(),
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
