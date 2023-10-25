package middleware

import (
	"context"
	"fmt"
	"rusi/pkg/messaging"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings"
)

func PublisherTracingMiddleware() messaging.Middleware {
	tr := otel.Tracer("tracing-middleware")

	return func(next messaging.Handler) messaging.Handler {
		return func(ctx context.Context, msg *messaging.MessageEnvelope) error {
			bags, spanCtx := Extract(ctx, msg.Headers)
			ctx = baggage.ContextWithBaggage(ctx, bags)
			topic := ctx.Value(messaging.TopicKey).(string)

			// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/messaging.md
			ctx, span := tr.Start(
				trace.ContextWithRemoteSpanContext(ctx, spanCtx),
				fmt.Sprintf("%s send", topic),
				trace.WithSpanKind(trace.SpanKindProducer),
				trace.WithAttributes(
					attribute.String("component", "Rusi"),
					semconv.MessagingDestinationName(topic),
					semconv.MessagingDestinationKindTopic))

			defer span.End()

			Inject(ctx, msg.Headers)
			klog.V(4).InfoS("publisher tracing middleware")
			err := next(ctx, msg)

			if err == nil {
				span.SetStatus(codes.Ok, "")
			} else {
				span.SetStatus(codes.Error, fmt.Sprintf("%v", err))
				span.RecordError(err, trace.WithStackTrace(true))
			}

			return err
		}
	}
}

func SubscriberTracingMiddleware() messaging.Middleware {
	tr := otel.Tracer("tracing-middleware")

	return func(next messaging.Handler) messaging.Handler {
		return func(ctx context.Context, msg *messaging.MessageEnvelope) error {

			bags, spanCtx := Extract(ctx, msg.Headers)
			ctx = baggage.ContextWithBaggage(ctx, bags)
			topic := ctx.Value(messaging.TopicKey).(string)

			// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/messaging.md
			ctx, span := tr.Start(
				trace.ContextWithRemoteSpanContext(ctx, spanCtx),
				fmt.Sprintf("%s receive", topic),
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					attribute.String("component", "Rusi"),
					semconv.MessagingDestinationName(topic),
					semconv.MessagingDestinationKindTopic,
					semconv.MessagingOperationReceive))

			span.AddEvent("new message received",
				trace.WithAttributes(attribute.String("headers", fmt.Sprintf("%v", msg.Headers))))

			if msg.Payload != nil {
				str := fmt.Sprintf("%v", msg.Payload)
				span.SetAttributes(attribute.Key("messaging.message_payload").String(strings.ShortenString(str, 500)))
			}
			span.SetAttributes(attribute.Key("messaging.message_id").String(msg.Id))

			Inject(ctx, msg.Headers)

			defer span.End()
			klog.V(4).InfoS("subscriber tracing middleware")
			err := next(ctx, msg)

			if err == nil {
				span.SetStatus(codes.Ok, "")
			} else {
				span.SetStatus(codes.Error, fmt.Sprintf("%v", err))
				span.RecordError(err, trace.WithStackTrace(true))
			}

			return err
		}
	}
}

type mapHeaderCarrier struct {
	innerMap map[string]string
}

func (i *mapHeaderCarrier) Get(key string) string {
	return i.innerMap[key]
}
func (i *mapHeaderCarrier) Set(key string, value string) {
	i.innerMap[key] = value
}
func (i *mapHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(i.innerMap))
	for k := range i.innerMap {
		keys = append(keys, k)
	}
	return keys
}

// Inject injects correlation context and span context into the gRPC
// metadata object. This function is meant to be used on outgoing
// requests.
func Inject(ctx context.Context, headers map[string]string) {
	otel.GetTextMapPropagator().Inject(ctx, &mapHeaderCarrier{headers})
}

// Extract returns the correlation context and span context that
// another service encoded in the gRPC metadata object with Inject.
// This function is meant to be used on incoming requests.
func Extract(ctx context.Context, headers map[string]string) (baggage.Baggage, trace.SpanContext) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, &mapHeaderCarrier{headers})
	return baggage.FromContext(ctx), trace.SpanContextFromContext(ctx)
}
