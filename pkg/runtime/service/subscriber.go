package service

import "rusi/pkg/messaging"

type SubscriberService struct {
}

func (srv *SubscriberService) StartSubscribing(pubsubName string, pubsub messaging.PubSub) error {
	return nil
}
