package runtime

import (
	"rusi/pkg/messaging"
)

type dummyApi struct {
	publishRequestHandler   messaging.PublishRequestHandler
	subscribeRequestHandler messaging.SubscribeRequestHandler
}

func NewDummyApi() *dummyApi {
	return &dummyApi{}
}

func (dummyApi) Serve() error {
	return nil
}

func (dummyApi) Refresh() error {
	return nil
}

func (d *dummyApi) SetPublishHandler(publishRequestHandler messaging.PublishRequestHandler) {
	d.publishRequestHandler = publishRequestHandler
}

func (d *dummyApi) SetSubscribeHandler(subscribeRequestHandler messaging.SubscribeRequestHandler) {
	d.subscribeRequestHandler = subscribeRequestHandler

}
