package service

import (
	"k8s.io/klog/v2"
	"rusi/pkg/messaging"
)

type SubscriberService struct {
}

func (srv *SubscriberService) StartSubscribing(pubsubName string, pubsub messaging.PubSub) error {

	return nil

	pubsub.Subscribe("TS1858.dapr_test_topic", func(env *messaging.MessageEnvelope) error {
		klog.InfoS("message received on topic dapr_test_topic",
			"payload", env.Payload, "headers", env.Headers)

		return nil
	})

	return nil
}
