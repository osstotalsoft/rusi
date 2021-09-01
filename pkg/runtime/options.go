package runtime

import (
	"rusi/pkg/custom-resource/components/middleware"
	"rusi/pkg/custom-resource/components/pubsub"
)

type (
	// runtimeOpts encapsulates the components to include in the runtime.
	runtimeOpts struct {
		pubsubs          []pubsub.PubSubDefinition
		pubsubMiddleware []middleware.Middleware
	}

	// Option is a function that customizes the runtime.
	Option func(o *runtimeOpts)
)

// WithPubSubs adds pubsub store components to the runtime.
func WithPubSubs(pubsubs ...pubsub.PubSubDefinition) Option {
	return func(o *runtimeOpts) {
		o.pubsubs = append(o.pubsubs, pubsubs...)
	}
}

// WithPubsubMiddleware adds Pubsub middleware components to the runtime.
func WithPubsubMiddleware(middleware ...middleware.Middleware) Option {
	return func(o *runtimeOpts) {
		o.pubsubMiddleware = append(o.pubsubMiddleware, middleware...)
	}
}
