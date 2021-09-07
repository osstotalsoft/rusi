package service

import (
	"context"
	"k8s.io/klog/v2"
	"rusi/pkg/messaging"
	"rusi/pkg/middleware"
)

type subscriberService struct {
	subscriber messaging.Subscriber
	pipeline   messaging.Pipeline
}

func NewSubscriberService(subscriber messaging.Subscriber, pipeline messaging.Pipeline) *subscriberService {
	return &subscriberService{subscriber, pipeline}
}

func (srv *subscriberService) StartSubscribing(topic string, handler messaging.Handler) (messaging.UnsubscribeFunc, error) {

	ctx := context.Background()

	//insert tracing by default
	srv.pipeline.UseMiddleware(middleware.TracingMiddleware())

	pipe := srv.pipeline.Build(func(ctx context.Context, env *messaging.MessageEnvelope) {
		err := handler(env)
		if err != nil {
			klog.ErrorS(err, "error calling handler")
		}
	})

	return srv.subscriber.Subscribe(topic, func(env *messaging.MessageEnvelope) error {
		klog.InfoS("message received on", "topic", topic,
			"payload", env.Payload, "headers", env.Headers)

		pipe(ctx, env)
		return nil
	})
}
