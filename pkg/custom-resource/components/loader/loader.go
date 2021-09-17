package loader

import (
	"context"
	"rusi/pkg/custom-resource/components"
)

type ComponentsLoader func(ctx context.Context) (<-chan components.Spec, error)
