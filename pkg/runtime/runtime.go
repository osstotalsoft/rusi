package runtime

import (
	"rusi/pkg/api"
	"rusi/pkg/components/pubsub"
)

type runtime struct {
	config         Config
	api            api.Api
	pubSubRegistry pubsub.Registry
}

func NewRuntime(config Config, api api.Api) *runtime {
	return &runtime{
		config:         config,
		api:            api,
		pubSubRegistry: pubsub.NewRegistry(),
	}
}

func (rt *runtime) Run(opts ...Option) error {

	var runtimeOpts runtimeOpts
	for _, opt := range opts {
		opt(&runtimeOpts)
	}

	rt.pubSubRegistry.Register(runtimeOpts.pubsubs...)

	return rt.api.Serve()
}
