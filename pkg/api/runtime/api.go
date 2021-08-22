package runtime

import (
	"rusi/pkg/messaging"
)

type Api interface {
	SendMessageToApp(env *messaging.MessageEnvelope) error
	Serve() error
}
