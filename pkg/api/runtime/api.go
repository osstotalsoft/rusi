package runtime

import "rusi/pkg/messaging"

type Api interface {
	Serve() error
	Refresh() error
	SetPublishHandler(messaging.PublishRequestHandler)
	SetSubscribeHandler(messaging.SubscribeRequestHandler)
}
