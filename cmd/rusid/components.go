package main

import (
	"context"
	"rusi/pkg/custom-resource/components/middleware"
	"rusi/pkg/custom-resource/components/pubsub"
	"rusi/pkg/messaging"
	natsstreaming "rusi/pkg/messaging/nats"
	middleware_pubsub "rusi/pkg/middleware/pubsub"
	"rusi/pkg/runtime"
	"strings"
)

func RegisterComponentFactories() (result []runtime.Option) {
	result = append(result,

		runtime.WithPubSubs(
			pubsub.New("natsstreaming", func() messaging.PubSub {
				return natsstreaming.NewNATSStreamingPubSub()
			}),
		),
		runtime.WithPubsubMiddleware(
			middleware.New("uppercase", func(properties map[string]string) middleware_pubsub.Middleware {
				return func(next middleware_pubsub.RequestHandler) middleware_pubsub.RequestHandler {
					return func(ctx context.Context, msg *messaging.MessageEnvelope) {
						body := msg.Payload.(string)
						msg.Payload = strings.ToUpper(body)
						next(ctx, msg)
					}
				}
			}),
		),
	)
	return
}
