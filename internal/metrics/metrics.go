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
	pubsubMeter          metric.Meter
	publishCount         metric.Int64Counter
	subscribeDurationMs  metric.Int64Histogram
	subscribeDurationSec metric.Float64Histogram
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

	//TODO should be removed
	subscribeDurationMs, _ := pubsubM.Int64Histogram("rusi.pubsub.processing.duration",
		metric.WithDescription("The duration of a message execution"),
		metric.WithUnit("milliseconds"))

	subscribeDurationSec, _ := pubsubM.Float64Histogram("rusi.pubsub.processing.duration.seconds",
		metric.WithDescription("The duration of a message execution"),
		metric.WithUnit("seconds"))

	return &serviceMetrics{
		pubsubMeter:          pubsubM,
		publishCount:         publishCount,
		subscribeDurationMs:  subscribeDurationMs,
		subscribeDurationSec: subscribeDurationSec,
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
	s.subscribeDurationMs.Record(ctx, elapsed.Milliseconds(), opt)
	s.subscribeDurationSec.Record(ctx, elapsed.Seconds(), opt)
}

func SetNoopMeterProvider() {
	otel.SetMeterProvider(noop.NewMeterProvider())
}
