package runtime

import (
	runtime_api "rusi/pkg/api/runtime"
	"rusi/pkg/components/pubsub"
)

type runtime struct {
	config         Config
	api            runtime_api.Api
	pubSubRegistry pubsub.Registry
}

func NewRuntime(config Config, api runtime_api.Api) *runtime {
	return &runtime{
		config:         config,
		api:            api,
		pubSubRegistry: pubsub.NewRegistry(),
	}
}

func (rt *runtime) ConfigureOptions(opts ...Option) error {

	var runtimeOpts runtimeOpts
	for _, opt := range opts {
		opt(&runtimeOpts)
	}

	rt.pubSubRegistry.Register(runtimeOpts.pubsubs...)

	return nil
}
