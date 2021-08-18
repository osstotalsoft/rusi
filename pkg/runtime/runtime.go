package runtime

import (
	"rusi/pkg/api"
)

type runtime struct {
	config Config
	api    api.Api
}

func NewRuntime(config Config, api api.Api) *runtime {
	return &runtime{
		config, api,
	}
}

func (rt *runtime) Run() error {
	return rt.api.Serve()
}
