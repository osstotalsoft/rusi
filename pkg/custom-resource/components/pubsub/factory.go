package pubsub

import (
	"k8s.io/klog/v2"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/messaging"
)

type Factory struct {
	consumerId     string
	components     []components.Spec
	pubSubRegistry Registry
	pubSubs        map[string]messaging.PubSub
}

func NewPubSubFactory(consumerId string) *Factory {
	return &Factory{
		pubSubs:        map[string]messaging.PubSub{},
		pubSubRegistry: NewRegistry(),
		consumerId:     consumerId,
	}
}

func (fact *Factory) Create(spec components.Spec) (string, messaging.PubSub, error) {
	pubSub, err := fact.pubSubRegistry.Create(spec.Type, spec.Version)
	if err != nil {
		klog.Warningf("error creating pub sub %s (%s/%s): %s", spec.Name, spec.Type, spec.Version, err)
		return spec.Name, nil, err
	}

	spec.Metadata["consumerID"] = fact.consumerId
	err = pubSub.Init(spec.Metadata)
	if err != nil {
		klog.Warningf("error initializing pub sub %s/%s: %s", spec.Type, spec.Version, err)
		return spec.Name, nil, err
	}
	fact.pubSubs[spec.Name] = pubSub

	return spec.Name, pubSub, nil
}

func (fact *Factory) GetPublisher(pubsubName string) messaging.Publisher {
	return fact.pubSubs[pubsubName]
}

func (fact *Factory) GetSubscriber(pubsubName string) messaging.Subscriber {
	return fact.pubSubs[pubsubName]
}

func (fact *Factory) Register(pubsubs ...PubSubDefinition) {
	fact.pubSubRegistry.Register(pubsubs...)
}
