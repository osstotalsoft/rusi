package main

import (
	"context"
	"rusi/pkg/custom-resource/components/middleware"
	"rusi/pkg/custom-resource/components/pubsub"
	"rusi/pkg/messaging"
	natsstreaming "rusi/pkg/messaging/nats"
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
			middleware.New("uppercase", func(properties map[string]string) messaging.Middleware {
				return func(next messaging.Handler) messaging.Handler {
					return func(ctx context.Context, msg *messaging.MessageEnvelope) error {
						body := msg.Payload.(string)
						msg.Payload = strings.ToUpper(body)
						return next(ctx, msg)
					}
				}
			}),
		),
	)
	return
}
