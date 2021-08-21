package runtime

import (
	"fmt"
	"k8s.io/klog/v2"
	runtime_api "rusi/pkg/api/runtime"
	"rusi/pkg/components"
	"rusi/pkg/components/pubsub"
	"rusi/pkg/messaging"
	"strings"
)

type ComponentProviderFunc func() ([]components.Spec, error)

type runtime struct {
	config         Config
	api            runtime_api.Api
	pubSubRegistry pubsub.Registry
	pubSubs        map[string]messaging.PubSub

	componentProviderFunc ComponentProviderFunc
}

func NewRuntime(config Config, api runtime_api.Api,
	componentProviderFunc ComponentProviderFunc) *runtime {
	return &runtime{
		config:         config,
		api:            api,
		pubSubRegistry: pubsub.NewRegistry(),
		pubSubs:        map[string]messaging.PubSub{},

		componentProviderFunc: componentProviderFunc,
	}
}

func (rt *runtime) Run(opts ...Option) error {

	var runtimeOpts runtimeOpts
	for _, opt := range opts {
		opt(&runtimeOpts)
	}

	rt.pubSubRegistry.Register(runtimeOpts.pubsubs...)
	err := rt.initComponents()
	if err != nil {
		klog.Errorf("Error loading components %s", err.Error())
		return err
	}
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
	pubSub, err := rt.pubSubRegistry.Create(spec.Type, spec.Version)
	if err != nil {
		klog.Warningf("error creating pub sub %s (%s/%s): %s", spec.Name, spec.Type, spec.Version, err)
		return err
	}

	pubSub.Init(spec.Metadata)
	if err != nil {
		klog.Warningf("error initializing pub sub %s/%s: %s", spec.Type, spec.Version, err)
		return err
	}
	rt.pubSubs[spec.Name] = pubSub

	return nil
}

func (rt *runtime) GetPubSub(pubsubName string) messaging.PubSub {
	return rt.pubSubs[pubsubName]
}

func extractComponentCategory(spec components.Spec) components.ComponentCategory {
	for _, category := range components.ComponentCategories {
		if strings.HasPrefix(spec.Type, fmt.Sprintf("%s.", category)) {
			return category
		}
	}
	return ""
}
