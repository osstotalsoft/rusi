package runtime

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	runtime_api "rusi/pkg/api/runtime"
	"rusi/pkg/custom-resource/components"
	components_loader "rusi/pkg/custom-resource/components/loader"
	"rusi/pkg/custom-resource/components/middleware"
	"rusi/pkg/custom-resource/components/pubsub"
	"rusi/pkg/custom-resource/configuration"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
	"rusi/pkg/messaging"
	"rusi/pkg/runtime/service"
	"strings"
)

type runtime struct {
	config           Config
	pubsubFactory    *pubsub.Factory
	componentsLoader components_loader.ComponentsLoader
	appConfig        configuration.Spec
	components       []components.Spec

	subscriptionMiddlewareRegistry middleware.Registry
}

func NewRuntime(config Config,
	componentsLoader components_loader.ComponentsLoader,
	configurationLoader configuration_loader.ConfigurationLoader) *runtime {

	appConfig, err := configurationLoader(config.Config)
	if err != nil {
		klog.ErrorS(err, "error loading application config", "name", config.Config, "mode", config.Mode)
		return nil
	}

	return &runtime{
		config:           config,
		pubsubFactory:    pubsub.NewPubSubFactory(config.AppID),
		componentsLoader: componentsLoader,
		appConfig:        appConfig,

		subscriptionMiddlewareRegistry: middleware.NewRegistry(),
	}
}

func (rt *runtime) Load(opts ...Option) error {
	var runtimeOpts runtimeOpts
	for _, opt := range opts {
		opt(&runtimeOpts)
	}

	rt.pubsubFactory.Register(runtimeOpts.pubsubs...)
	rt.subscriptionMiddlewareRegistry.Register(runtimeOpts.pubsubMiddleware...)

	err := rt.initComponents()
	if err != nil {
		klog.Errorf("Error loading components %s", err.Error())
		return err
	}

	klog.Infof("app id: %s", rt.config.AppID)

	return nil
}

func (rt *runtime) initComponents() error {
	comps, err := rt.componentsLoader()
	if err != nil {
		return err
	}

	for _, item := range comps {
		err = rt.initComponent(item)
		if err != nil {
			return err
		}
	}
	rt.components = comps
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

func (rt *runtime) getComponent(componentType string, name string) (components.Spec, bool) {
	for _, c := range rt.components {
		if c.Type == componentType && c.Name == name {
			return c, true
		}
	}
	return components.Spec{}, false
}

func (rt *runtime) buildSubscriberPipeline() (pipeline messaging.Pipeline, err error) {

	for _, middlewareSpec := range rt.appConfig.SubscriberPipelineSpec.Handlers {
		component, exists := rt.getComponent(middlewareSpec.Type, middlewareSpec.Name)
		if !exists {
			return pipeline,
				errors.Errorf("couldn't find middleware component with name %s and type %s/%s",
					middlewareSpec.Name,
					middlewareSpec.Type,
					middlewareSpec.Version)
		}
		midlw, err := rt.subscriptionMiddlewareRegistry.Create(middlewareSpec.Type, middlewareSpec.Version,
			component.Metadata)
		if err != nil {
			return pipeline, err
		}
		klog.Infof("enabled %s/%s middleware", middlewareSpec.Type, middlewareSpec.Version)
		pipeline.UseMiddleware(midlw)
	}
	return pipeline, nil
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
	pipeline, err := rt.buildSubscriberPipeline()
	if err != nil {
		klog.ErrorS(err, "error building pipeline")
		return nil, err
	}
	srv := service.NewSubscriberService(subs, pipeline)
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
