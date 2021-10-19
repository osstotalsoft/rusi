package metrics

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/unit"
	"os"
	"time"
)

var (
	// DefaultMonitoring holds service monitoring metrics definitions.
	DefaultMonitoring = newServiceMetrics()
)

type serviceMetrics struct {
	pubsubMeter        metric.Meter
	hostname           string
	publishCount       metric.Int64Counter
	processingDuration metric.Int64Histogram
}

func newServiceMetrics() *serviceMetrics {
	meter := global.Meter("rusi.io/pubsub")
	pubsubM := metric.Must(meter)
	hostname, _ := os.Hostname()

	return &serviceMetrics{
		pubsubMeter:        meter,
		hostname:           hostname,
		publishCount:       pubsubM.NewInt64Counter("rusi.io.pubsub.publishcount"),
		processingDuration: pubsubM.NewInt64Histogram("rusi.io.pubsub.processingduration", metric.WithUnit(unit.Milliseconds)),
	}
}

func (s *serviceMetrics) RecordPublishMessage(ctx context.Context, topic string) {
	s.pubsubMeter.RecordBatch(
		ctx,
		[]attribute.KeyValue{
			attribute.String("hostname", s.hostname),
			attribute.String("topic", topic),
			attribute.Bool("success", true),
		},
		s.publishCount.Measurement(1))
}

func (s *serviceMetrics) RecordSubscriberProcessingTime(ctx context.Context, topic string, elapsed time.Duration) {
	s.pubsubMeter.RecordBatch(
		ctx,
		[]attribute.KeyValue{
			attribute.String("hostname", s.hostname),
			attribute.String("topic", topic),
			attribute.Bool("success", true),
		},
		s.processingDuration.Measurement(elapsed.Milliseconds()))
}
