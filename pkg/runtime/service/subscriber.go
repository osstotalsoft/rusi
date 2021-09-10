package service

import (
	"context"
	"rusi/pkg/messaging"
	"rusi/pkg/middleware"

	"k8s.io/klog/v2"
)

type subscriberService struct {
	subscriber messaging.Subscriber
	pipeline   messaging.Pipeline
}

func NewSubscriberService(subscriber messaging.Subscriber, pipeline messaging.Pipeline) *subscriberService {
	return &subscriberService{subscriber, pipeline}
}

func (srv *subscriberService) StartSubscribing(request messaging.SubscribeRequest) (messaging.UnsubscribeFunc, error) {

	//insert tracing by default
	srv.pipeline.UseMiddleware(middleware.SubscriberTracingMiddleware())
	pipe := srv.pipeline.Build(request.Handler)

	return srv.subscriber.Subscribe(request.Topic, func(ctx context.Context, env *messaging.MessageEnvelope) error {
		klog.InfoS("message received on", "topic", request.Topic,
			"payload", env.Payload, "headers", env.Headers)

		ctx = context.WithValue(ctx, "topic", request.Topic)
		err := pipe(ctx, env)
		if err != nil {
			klog.ErrorS(err, "error calling handler")
		}
		return err
	}, request.Options)
}
