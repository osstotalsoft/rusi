package runtime

import (
	"rusi/pkg/messaging"
)

type DummyApi struct {
	PublishRequestHandler   messaging.PublishRequestHandler
	SubscribeRequestHandler messaging.SubscribeRequestHandler
	RefreshChan             chan bool
}

func NewDummyApi() *DummyApi {
	return &DummyApi{RefreshChan: make(chan bool)}
}

func (DummyApi) Serve() error {
	return nil
}

func (d *DummyApi) Refresh() error {
	d.RefreshChan <- true
	return nil
}

func (d *DummyApi) SetPublishHandler(publishRequestHandler messaging.PublishRequestHandler) {
	d.PublishRequestHandler = publishRequestHandler
}

func (d *DummyApi) SetSubscribeHandler(subscribeRequestHandler messaging.SubscribeRequestHandler) {
	d.SubscribeRequestHandler = subscribeRequestHandler

}
