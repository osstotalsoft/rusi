package middleware

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
	"rusi/pkg/messaging"
)

func TracingMiddleware() messaging.Middleware {
	tr := otel.Tracer("tracing-middleware")

	return func(next messaging.Handler) messaging.Handler {
		return func(ctx context.Context, msg *messaging.MessageEnvelope) error {
			_, span := tr.Start(ctx, "Publisher new incoming message")
			span.AddEvent("new message received",
				trace.WithAttributes(attribute.String("headers", fmt.Sprintf("%v", msg.Headers))))
			span.SetAttributes(attribute.Key("message").String(fmt.Sprintf("%v", *msg)))
			defer span.End()
			klog.V(4).InfoS("tracing middleware hit")
			return next(ctx, msg)
		}
	}
}
