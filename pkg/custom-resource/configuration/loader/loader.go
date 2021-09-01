package loader

import (
	"rusi/pkg/custom-resource/configuration"
)

type ConfigurationLoader func(name string) (configuration.Spec, error)
