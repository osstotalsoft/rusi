package main

import (
	"context"
	"k8s.io/klog/v2"
	"rusi/pkg/custom-resource/components/middleware"
	"rusi/pkg/custom-resource/components/pubsub"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/jetstream"
	natsstreaming "rusi/pkg/messaging/nats"
	"rusi/pkg/runtime"
)

func RegisterComponentFactories() (result []runtime.Option) {
	result = append(result,

		runtime.WithPubSubs(
			pubsub.New("natsstreaming", func() messaging.PubSub {
				return natsstreaming.NewNATSStreamingPubSub()
			}),
			pubsub.New("jetstream", func() messaging.PubSub {
				return jetstream.NewJetStreamPubSub()
			}),
		),
		runtime.WithPubsubMiddleware(
			middleware.New("uppercase", func(properties map[string]string) messaging.Middleware {
				return func(next messaging.Handler) messaging.Handler {
					return func(ctx context.Context, msg *messaging.MessageEnvelope) error {
						klog.V(4).InfoS("uppercase middleware hit")
						return next(ctx, msg)
					}
				}
			}),
		),
	)
	return
}
