package runtime

import (
	"rusi/pkg/messaging"
)

type Api interface {
	Publish(env *messaging.MessageEnvelope) error
}
