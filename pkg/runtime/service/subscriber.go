package service

import (
	"k8s.io/klog/v2"
	"rusi/pkg/messaging"
)

type subscriberService struct {
	subscriber messaging.Subscriber
}

func NewSubscriberService(subscriber messaging.Subscriber) *subscriberService {
	return &subscriberService{subscriber}
}

func (srv *subscriberService) StartSubscribing(topic string, handler messaging.Handler) (messaging.UnsubscribeFunc, error) {
	return srv.subscriber.Subscribe(topic, func(env *messaging.MessageEnvelope) error {
		klog.InfoS("message received on", "topic", topic,
			"payload", env.Payload, "headers", env.Headers)

		//TODO prepare and invoke pipeline
		return handler(env)
	})
}
