package loader

import (
	"rusi/pkg/custom-resource/components"
)

type ComponentsLoader func() ([]components.Spec, error)
