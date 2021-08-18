package runtime

import "rusi/pkg/components/pubsub"

type (
	// runtimeOpts encapsulates the components to include in the runtime.
	runtimeOpts struct {
		pubsubs []pubsub.PubSub
	}

	// Option is a function that customizes the runtime.
	Option func(o *runtimeOpts)
)

// WithPubSubs adds pubsub store components to the runtime.
func WithPubSubs(pubsubs ...pubsub.PubSub) Option {
	return func(o *runtimeOpts) {
		o.pubsubs = append(o.pubsubs, pubsubs...)
	}
}
