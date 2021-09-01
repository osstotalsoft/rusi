package components

import (
	"time"
)

type ComponentCategory string

const (
	BindingsComponent               ComponentCategory = "bindings"
	PubsubComponent                 ComponentCategory = "pubsub"
	SecretStoreComponent            ComponentCategory = "secretstores"
	MiddlewareComponent             ComponentCategory = "middleware"
	DefaultComponentInitTimeout                       = time.Second * 5
	DefaultGracefulShutdownDuration                   = time.Second * 5
)

var ComponentCategories = []ComponentCategory{
	BindingsComponent,
	PubsubComponent,
	SecretStoreComponent,
	MiddlewareComponent,
}

type Spec struct {
	Name     string
	Type     string
	Version  string            `json:"version" yaml:"version"`
	Metadata map[string]string `json:"metadata" yaml:"metadata"`
	Scopes   []string          `json:"scopes" yaml:"scopes"`
}
