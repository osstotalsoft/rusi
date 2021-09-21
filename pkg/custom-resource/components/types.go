package components

import (
	"time"
)

type ComponentCategory string
type Operation string

const (
	BindingsComponent    ComponentCategory = "bindings"
	PubsubComponent      ComponentCategory = "pubsub"
	SecretStoreComponent ComponentCategory = "secretstores"
	MiddlewareComponent  ComponentCategory = "middleware"

	Insert Operation = "insert"
	Update Operation = "update"
	Delete Operation = "delete"

	DefaultComponentInitTimeout     = time.Second * 5
	DefaultGracefulShutdownDuration = time.Second * 5
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

type ChangeNotification struct {
	ComponentCategory ComponentCategory
	Operation         Operation
	ComponentSpec     Spec
}
