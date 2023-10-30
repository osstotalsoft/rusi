package metrics

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"sync"
	"time"
)

type serviceMetrics struct {
	pubsubMeter       metric.Meter
	publishCount      metric.Int64Counter
	subscribeDuration metric.Int64Histogram
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

	publishCount, _ := pubsubM.Int64Counter("rusi.pubsub.publish.count",
		metric.WithDescription("The number of publishes"))

	subscribeDuration, _ := pubsubM.Int64Histogram("rusi.pubsub.processing.duration",
		metric.WithDescription("The duration of a message execution"),
		metric.WithUnit("milliseconds"))

	return &serviceMetrics{
		pubsubMeter:       pubsubM,
		publishCount:      publishCount,
		subscribeDuration: subscribeDuration,
	}
}

func (s *serviceMetrics) RecordPublishMessage(ctx context.Context, topic string, success bool) {
	opt := metric.WithAttributes(
		attribute.String("topic", topic),
		attribute.Bool("success", success),
	)
	s.publishCount.Add(ctx, 1, opt)
}

func (s *serviceMetrics) RecordSubscriberProcessingTime(ctx context.Context, topic string, success bool, elapsed time.Duration) {
	opt := metric.WithAttributes(
		attribute.String("topic", topic),
		attribute.Bool("success", success),
	)
	s.subscribeDuration.Record(ctx, elapsed.Milliseconds(), opt)
}

func SetNoopMeterProvider() {
	otel.SetMeterProvider(noop.NewMeterProvider())
}
