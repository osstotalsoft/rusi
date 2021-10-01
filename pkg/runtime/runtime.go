package runtime

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"reflect"
	runtime_api "rusi/pkg/api/runtime"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/custom-resource/configuration"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
	"rusi/pkg/messaging"
	"rusi/pkg/middleware"
	"rusi/pkg/runtime/service"
	"time"
)

type runtime struct {
	ctx   context.Context
	appID string
	api   runtime_api.Api

	appConfig         configuration.Spec
	componentsManager *ComponentsManager

	configurationUpdatesChan <-chan configuration.Spec
}

func NewRuntime(ctx context.Context, config Config, api runtime_api.Api,
	configurationLoader configuration_loader.ConfigurationLoader,
	manager *ComponentsManager) (*runtime, error) {

	configChan, err := configurationLoader(ctx, config.Config)
	if err != nil {
		klog.ErrorS(err, "error loading application config", "name",
			config.Config, "mode", config.Mode)
		return nil, err
	}

	//block until config arrives
	rt := &runtime{
		api:   api,
		ctx:   ctx,
		appID: config.AppID,

		configurationUpdatesChan: configChan,
		appConfig:                <-configChan,
		componentsManager:        manager,
	}

	api.SetPublishHandler(rt.PublishHandler)
	api.SetSubscribeHandler(rt.SubscribeHandler)

	go rt.watchComponentsUpdates()
	go rt.watchConfigurationUpdates()

	return rt, nil
}

func (rt *runtime) watchConfigurationUpdates() {
	for update := range rt.configurationUpdatesChan {
		if reflect.DeepEqual(rt.appConfig, update) {
			return
		}
		klog.V(4).InfoS("configuration changed",
			"old config", rt.appConfig, "new config", update)
		rt.appConfig = update
		err := rt.api.Refresh()
		if err != nil {
			klog.V(4).ErrorS(err, "error refreshing subscription")
		}
	}
}

func (rt *runtime) watchComponentsUpdates() {
	for update := range rt.componentsManager.Watch() {
		klog.V(4).InfoS("component changed", "operation", update.Operation,
			"name", update.ComponentSpec.Name, "type", update.ComponentSpec.Type)

		switch {
		case update.ComponentCategory == components.PubsubComponent && update.Operation == components.Update:
			err := rt.api.Refresh()
			if err != nil {
				klog.V(4).ErrorS(err, "error refreshing subscription")
			}
		}
	}
}

func (rt *runtime) buildSubscriberPipeline() (pipeline messaging.Pipeline, err error) {
	for _, middlewareSpec := range rt.appConfig.SubscriberPipelineSpec.Handlers {
		midlw, err := rt.componentsManager.GetMiddleware(middlewareSpec)
		if err != nil {
			return pipeline, err
		}
		klog.Infof("enabled %s/%s middleware", middlewareSpec.Type, middlewareSpec.Version)
		pipeline.UseMiddleware(midlw)
	}
	return pipeline, nil
}

func (rt *runtime) PublishHandler(ctx context.Context, request messaging.PublishRequest) error {
	publisher := rt.componentsManager.GetPublisher(request.PubsubName)
	if publisher == nil {
		return errors.New(fmt.Sprintf(runtime_api.ErrPubsubNotFound, request.PubsubName))
	}
	env := &messaging.MessageEnvelope{
		Id:              uuid.New().String(),
		Time:            time.Now(),
		Subject:         request.Topic,
		Type:            request.Type,
		DataContentType: request.DataContentType,
		Headers:         request.Metadata,
		Payload:         request.Data,
	}

	ctx = context.WithValue(ctx, messaging.TopicKey, request.Topic)
	midl := middleware.PublisherTracingMiddleware()

	return midl(func(ctx context.Context, msg *messaging.MessageEnvelope) error {
		return publisher.Publish(request.Topic, msg)
	})(ctx, env)
}

func (rt *runtime) SubscribeHandler(ctx context.Context, request messaging.SubscribeRequest) (messaging.CloseFunc, error) {
	subs := rt.componentsManager.GetSubscriber(request.PubsubName)
	if subs == nil {
		err := errors.New(fmt.Sprintf("cannot find PubsubName named %s", request.PubsubName))
		klog.ErrorS(err, err.Error())
		return nil, err
	}
	pipeline, err := rt.buildSubscriberPipeline()
	if err != nil {
		klog.ErrorS(err, "error building pipeline")
		return nil, err
	}
	srv := service.NewSubscriberService(subs, pipeline)
	return srv.StartSubscribing(request)
}

func (rt *runtime) Run() error {
	return rt.api.Serve()
}
