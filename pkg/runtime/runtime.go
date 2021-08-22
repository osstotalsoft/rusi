package runtime

import (
	"fmt"
	"k8s.io/klog/v2"
	runtime_api "rusi/pkg/api/runtime"
	"rusi/pkg/components"
	"rusi/pkg/components/pubsub"
	"rusi/pkg/messaging"
	"rusi/pkg/runtime/service"
	"strings"
)

type ComponentProviderFunc func() ([]components.Spec, error)

type runtime struct {
	config                Config
	api                   runtime_api.Api
	pubsubFactory         *pubsub.Factory
	componentProviderFunc ComponentProviderFunc
}

func NewRuntime(config Config, api runtime_api.Api, componentProviderFunc ComponentProviderFunc,
	pubsubFactory *pubsub.Factory) *runtime {
	return &runtime{
		config:                config,
		api:                   api,
		pubsubFactory:         pubsubFactory,
		componentProviderFunc: componentProviderFunc,
	}
}

func (rt *runtime) Run(opts ...Option) error {
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

	return rt.api.Serve()
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
	name, ps, err := rt.pubsubFactory.Create(spec)
	if err != nil {
		return err
	}
	if ps != nil {
		err = rt.startSubscribing(name, ps)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rt *runtime) startSubscribing(name string, pubSub messaging.PubSub) error {

	subscriberService := service.SubscriberService{}
	if err := subscriberService.StartSubscribing(name, pubSub); err != nil {
		klog.Errorf("error occurred while subscribing to pubsub %s: %s", name, err)
		return err
	}
	return nil
}

func extractComponentCategory(spec components.Spec) components.ComponentCategory {
	for _, category := range components.ComponentCategories {
		if strings.HasPrefix(spec.Type, fmt.Sprintf("%s.", category)) {
			return category
		}
	}
	return ""
}
