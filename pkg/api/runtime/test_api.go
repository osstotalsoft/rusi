package runtime

import (
	"context"
	"rusi/pkg/messaging"
)

type TestApi struct {
	PublishRequestHandler   messaging.PublishRequestHandler
	SubscribeRequestHandler messaging.SubscribeRequestHandler
	RefreshChan             chan bool
}

func NewTestApi() *TestApi {
	return &TestApi{RefreshChan: make(chan bool)}
}

func (TestApi) Serve(ctx context.Context) error {
	return nil
}

func (d *TestApi) Refresh() error {
	d.RefreshChan <- true
	return nil
}

func (d *TestApi) SetPublishHandler(publishRequestHandler messaging.PublishRequestHandler) {
	d.PublishRequestHandler = publishRequestHandler
}

func (d *TestApi) SetSubscribeHandler(subscribeRequestHandler messaging.SubscribeRequestHandler) {
	d.SubscribeRequestHandler = subscribeRequestHandler

}
