package middleware

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
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

// Inject injects correlation context and span context into the gRPC
// metadata object. This function is meant to be used on outgoing
// requests.
func Inject(ctx context.Context, metadata *metadata.MD, opts ...Option) {
	c := newConfig(opts)
	c.Propagators.Inject(ctx, &metadataSupplier{
		metadata: metadata,
	})
}

// Extract returns the correlation context and span context that
// another service encoded in the gRPC metadata object with Inject.
// This function is meant to be used on incoming requests.
func Extract(ctx context.Context, metadata *metadata.MD, opts ...Option) (baggage.Baggage, trace.SpanContext) {
	c := newConfig(opts)
	ctx = c.Propagators.Extract(ctx, &metadataSupplier{
		metadata: metadata,
	})
	return baggage.FromContext(ctx), trace.SpanContextFromContext(ctx)
}
