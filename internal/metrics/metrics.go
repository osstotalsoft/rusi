package metrics

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"os"
	"sync"
	"time"
)

type serviceMetrics struct {
	pubsubMeter       metric.Meter
	hostname          string
	publishCount      metric.Int64Counter
	subscribeDuration metric.Int64Histogram
}

var (
	initOnce sync.Once
	s        *serviceMetrics
)

func DefaultMonitoring() *serviceMetrics {
	initOnce.Do(func() {
		s = newServiceMetrics()
	})
	return s
}

func newServiceMetrics() *serviceMetrics {
	meter := global.Meter("rusi.io/pubsub")
	pubsubM := metric.Must(meter)
	hostname, _ := os.Hostname()

	return &serviceMetrics{
		pubsubMeter: meter,
		hostname:    hostname,
		publishCount: pubsubM.NewInt64Counter("pubsub.publish.count",
			metric.WithDescription("The number of publishes")),
		subscribeDuration: pubsubM.NewInt64Histogram("pubsub.subscribe.duration",
			metric.WithDescription("The duration of a message execution"),
			metric.WithUnit("milliseconds")),
	}
}

func (s *serviceMetrics) RecordPublishMessage(ctx context.Context, topic string, success bool) {
	s.pubsubMeter.RecordBatch(
		ctx,
		[]attribute.KeyValue{
			attribute.String("hostname", s.hostname),
			attribute.String("topic", topic),
			attribute.Bool("success", success),
		},
		s.publishCount.Measurement(1))
}

func (s *serviceMetrics) RecordSubscriberProcessingTime(ctx context.Context, topic string, success bool, elapsed time.Duration) {
	s.pubsubMeter.RecordBatch(
		ctx,
		[]attribute.KeyValue{
			attribute.String("hostname", s.hostname),
			attribute.String("topic", topic),
			attribute.Bool("success", success),
		},
		s.subscribeDuration.Measurement(elapsed.Milliseconds()))
}

func SetNoopMeterProvider() {
	global.SetMeterProvider(metric.NewNoopMeterProvider())
}
