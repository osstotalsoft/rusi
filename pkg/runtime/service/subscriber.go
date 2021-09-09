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

	//insert tracing by default
	srv.pipeline.UseMiddleware(middleware.SubscriberTracingMiddleware())
	pipe := srv.pipeline.Build(handler)

	return srv.subscriber.Subscribe(topic, func(ctx context.Context, env *messaging.MessageEnvelope) error {
		klog.InfoS("message received on", "topic", topic,
			"payload", env.Payload, "headers", env.Headers)

		ctx = context.WithValue(ctx, "topic", topic)
		err := pipe(ctx, env)
		if err != nil {
			klog.ErrorS(err, "error calling handler")
		}
		return err
	})
}
