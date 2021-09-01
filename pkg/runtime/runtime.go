package runtime

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	runtime_api "rusi/pkg/api/runtime"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/custom-resource/components/pubsub"
	"rusi/pkg/messaging"
	"rusi/pkg/runtime/service"
	"strings"
)

type ComponentProviderFunc func() ([]components.Spec, error)

type runtime struct {
	config                Config
	pubsubFactory         *pubsub.Factory
	componentProviderFunc ComponentProviderFunc
}

func NewRuntime(config Config, componentProviderFunc ComponentProviderFunc) *runtime {
	return &runtime{
		config:                config,
		pubsubFactory:         pubsub.NewPubSubFactory(config.AppID),
		componentProviderFunc: componentProviderFunc,
	}
}

func (rt *runtime) Load(opts ...Option) error {
	var runtimeOpts runtimeOpts
	for _, opt := range opts {
		opt(&runtimeOpts)
	}

	rt.pubsubFactory.Register(runtimeOpts.pubsubs...)

	err := rt.initComponents()
	if err != nil {
		klog.Errorf("Error loading components %s", err.Error())
		return err
	}

	klog.Infof("app id: %s", rt.config.AppID)

	return nil
}

func (rt *runtime) initComponents() error {
	comps, err := rt.componentProviderFunc()
	if err != nil {
		return err
	}

	for _, item := range comps {
		err = rt.initComponent(item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rt *runtime) initComponent(spec components.Spec) error {

	klog.InfoS("loading component", "name", spec.Name, "type", spec.Type, "version", spec.Version)
	categ := extractComponentCategory(spec)
	switch categ {
	case components.BindingsComponent:
		return nil
	case components.PubsubComponent:
		return rt.initPubSub(spec)
	case components.SecretStoreComponent:
		return nil
	}
	return nil
}

func (rt *runtime) initPubSub(spec components.Spec) error {
	_, _, err := rt.pubsubFactory.Create(spec)
	return err
}

func (rt *runtime) PublishHandler(request messaging.PublishRequest) error {
	publisher := rt.pubsubFactory.GetPublisher(request.PubsubName)
	if publisher == nil {
		return errors.New(runtime_api.ErrPubsubNotFound)
	}
	return publisher.Publish(request.Topic, &messaging.MessageEnvelope{
		Headers: request.Metadata,
		Payload: string(request.Data),
	})
}

func (rt *runtime) SubscribeHandler(request messaging.SubscribeRequest) (messaging.UnsubscribeFunc, error) {
	subs := rt.pubsubFactory.GetSubscriber(request.PubsubName)
	srv := service.NewSubscriberService(subs)
	return srv.StartSubscribing(request.Topic, request.Handler)
}

func extractComponentCategory(spec components.Spec) components.ComponentCategory {
	for _, category := range components.ComponentCategories {
		if strings.HasPrefix(spec.Type, fmt.Sprintf("%s.", category)) {
			return category
		}
	}
	return ""
}
