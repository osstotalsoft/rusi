package middleware

import (
	"context"
	"rusi/internal/metrics"
	"rusi/pkg/messaging"
	"time"
)

func SubscriberMetricsMiddleware() messaging.Middleware {
	return func(next messaging.Handler) messaging.Handler {
		return func(ctx context.Context, msg *messaging.MessageEnvelope) error {
			start := time.Now()
			err := next(ctx, msg)
			metrics.DefaultMonitoring.RecordSubscriberProcessingTime(ctx, msg.Subject, err == nil, time.Since(start))
			return err
		}
	}
}

func PublisherMetricsMiddleware() messaging.Middleware {
	return func(next messaging.Handler) messaging.Handler {
		return func(ctx context.Context, msg *messaging.MessageEnvelope) error {
			err := next(ctx, msg)
			metrics.DefaultMonitoring.RecordPublishMessage(ctx, msg.Subject, err == nil)
			return err
		}
	}
}
