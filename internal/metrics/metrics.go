package metrics

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

type serviceMetrics struct {
	pubsubMeter       api.Meter
	publishCount      api.Int64Counter
	subscribeDuration api.Int64Histogram
}

var (
	initOnce sync.Once
	s        *serviceMetrics
)

func DefaultPubSubMetrics() *serviceMetrics {
	initOnce.Do(func() {
		s = newServiceMetrics()
	})
	return s
}

func newServiceMetrics() *serviceMetrics {
	pubsubM := otel.GetMeterProvider().Meter("rusi.io/pubsub")
	publishCount, err := pubsubM.Int64Counter("rusi.pubsub.publish.count",
		api.WithDescription("The number of publishes"))
	if err != nil {
		klog.ErrorS(err, "cannot create Int64Counter")
	}
	subscribeDuration, err := pubsubM.Int64Histogram("rusi.pubsub.processing.duration",
		api.WithDescription("The duration of a message execution"),
		api.WithUnit("milliseconds"))
	if err != nil {
		klog.ErrorS(err, "cannot create Int64Histogram")
	}
	return &serviceMetrics{
		pubsubMeter:       pubsubM,
		publishCount:      publishCount,
		subscribeDuration: subscribeDuration,
	}
}

func (s *serviceMetrics) RecordPublishMessage(ctx context.Context, topic string, success bool) {
	opt := api.WithAttributes(
		attribute.String("topic", topic),
		attribute.Bool("success", success),
	)
	s.publishCount.Add(ctx, 1, opt)
}

func (s *serviceMetrics) RecordSubscriberProcessingTime(ctx context.Context, topic string, success bool, elapsed time.Duration) {
	opt := api.WithAttributes(
		attribute.String("topic", topic),
		attribute.Bool("success", success),
	)
	s.subscribeDuration.Record(ctx, elapsed.Milliseconds(), opt)
}

func CreateMeterProvider(appId string, exporter metric.Reader) {
	r, _ := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithAttributes(semconv.ServiceNameKey.String(appId)))

	otel.SetMeterProvider(
		metric.NewMeterProvider(
			metric.WithReader(exporter),
			metric.WithView(metric.NewView(metric.Instrument{
				Name:  "custom_histogram",
				Scope: instrumentation.Scope{Name: "rusi.io/pubsub"},
			},
				metric.Stream{
					Name: "bar",
					Aggregation: aggregation.ExplicitBucketHistogram{
						Boundaries: []float64{10, 100, 1000, 10000, 100000},
					},
				},
			)),
			metric.WithResource(r)),
	)
}
func SetNoopMeterProvider() {
	otel.SetMeterProvider(noop.NewMeterProvider())
}
