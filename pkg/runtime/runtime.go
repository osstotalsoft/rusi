package runtime

import (
	"context"
	"fmt"
	"reflect"
	"rusi/internal/version"
	runtime_api "rusi/pkg/api/runtime"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/custom-resource/configuration"
	configuration_loader "rusi/pkg/custom-resource/configuration/loader"
	"rusi/pkg/healthcheck"
	"rusi/pkg/messaging"
	"rusi/pkg/middleware"
	"rusi/pkg/runtime/service"
	"time"

	"errors"
	"github.com/google/uuid"
	"k8s.io/klog/v2"
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

	klog.InfoS("Loading configuration")
	configChan, err := configurationLoader(ctx)
	if err != nil {
		klog.ErrorS(err, "error loading application config", "name",
			config.Config, "mode", config.Mode)
		return nil, err
	}

	//block until config arrives
	klog.InfoS("Waiting for configuration changes")
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
			klog.V(4).InfoS("configuration not changed")
			continue
		}
		klog.InfoS("configuration changed")
		klog.V(4).InfoS("configuration details", "old", rt.appConfig, "new", update)
		rt.appConfig = update
		err := rt.api.Refresh()
		if err != nil {
			klog.ErrorS(err, "error refreshing subscription")
		}
	}
}

func (rt *runtime) watchComponentsUpdates() {
	for update := range rt.componentsManager.Watch() {
		klog.InfoS("component changed", "operation", update.Operation,
			"name", update.ComponentSpec.Name, "type", update.ComponentSpec.Type)

		switch {
		//case update.ComponentCategory == components.PubsubComponent && update.Operation == components.Update:
		case update.Operation == components.Update:
			err := rt.api.Refresh()
			if err != nil {
				klog.ErrorS(err, "error refreshing subscription")
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
	var pubSubName string
	if request.PubsubName != "" {
		pubSubName = request.PubsubName
	} else {
		pubSubName = rt.appConfig.PubSubSpec.Name
	}
	if pubSubName == "" {
		return errors.New("PubSubName is empty. Please provide a valid pubSub name in the publish request or via configuration.")
	}

	publisher := rt.componentsManager.GetPublisher(pubSubName)
	if publisher == nil {
		return errors.New(fmt.Sprintf(runtime_api.ErrPubsubNotFound, pubSubName))
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

	pipe := messaging.Pipeline{}
	pipe.UseMiddleware(middleware.PublisherTracingMiddleware())
	pipe.UseMiddleware(middleware.PublisherMetricsMiddleware())

	midl := pipe.Build(func(ctx context.Context, msg *messaging.MessageEnvelope) error {
		return publisher.Publish(request.Topic, msg)
	})

	return midl(ctx, env)
}

func (rt *runtime) SubscribeHandler(ctx context.Context, request messaging.SubscribeRequest) (messaging.CloseFunc, error) {
	var pubSubName string
	if request.PubsubName != "" {
		pubSubName = request.PubsubName
	} else {
		pubSubName = rt.appConfig.PubSubSpec.Name
	}
	if pubSubName == "" {
		return nil, errors.New("PubSubName is empty. Please provide a valid pubSub name in the subscribe request or via configuration.")
	}

	subs := rt.componentsManager.GetSubscriber(pubSubName)
	if subs == nil {
		err := errors.New(fmt.Sprintf("cannot find PubsubName named %s", pubSubName))
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

func (rt *runtime) Run(ctx context.Context) error {
	return rt.api.Serve(ctx)
}

func (rt *runtime) IsHealthy(ctx context.Context) healthcheck.HealthResult {
	if rt.appConfig.MinRuntimeVersion != "" && version.Version() < rt.appConfig.MinRuntimeVersion {
		return healthcheck.HealthResult{
			Status:      healthcheck.Unhealthy,
			Description: "a bigger minimum runtime version is required",
		}
	}
	return healthcheck.HealthyResult
}
