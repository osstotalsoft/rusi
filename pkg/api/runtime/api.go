package runtime

import (
	"context"
	"rusi/pkg/messaging"
)

type Api interface {
	Serve(ctx context.Context) error
	Refresh() error
	SetPublishHandler(messaging.PublishRequestHandler)
	SetSubscribeHandler(messaging.SubscribeRequestHandler)
}
