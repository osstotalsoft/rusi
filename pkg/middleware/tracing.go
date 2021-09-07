package middleware

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"k8s.io/klog/v2"
	"rusi/pkg/messaging"
)

func TracingMiddleware() messaging.Middleware {
	tr := otel.Tracer("tracing-middleware")

	return func(next messaging.RequestHandler) messaging.RequestHandler {
		return func(ctx context.Context, msg *messaging.MessageEnvelope) {
			_, span := tr.Start(ctx, "Publisher new incoming message")
			span.SetAttributes(attribute.Key("message").String(fmt.Sprintf("%v", *msg)))
			defer span.End()
			klog.V(4).InfoS("tracing middleware hit")
			next(ctx, msg)
		}
	}
}
