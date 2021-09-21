package loader

import (
	"context"
	"rusi/pkg/custom-resource/configuration"
)

type ConfigurationLoader func(ctx context.Context, name string) (<-chan configuration.Spec, error)
