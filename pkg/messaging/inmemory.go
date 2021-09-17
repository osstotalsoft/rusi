package messaging

import "context"

type inMemoryBus struct {
	handlers map[string][]Handler
}

func NewInMemoryBus() inMemoryBus {
	return inMemoryBus{handlers: map[string][]Handler{}}
}

func (c inMemoryBus) Publish(topic string, env *MessageEnvelope) error {
	go runHandlers(c.handlers[topic], env)
	return nil
}

func (c inMemoryBus) Subscribe(topic string, handler Handler, options *SubscriptionOptions) (UnsubscribeFunc, error) {
	c.handlers[topic] = append(c.handlers[topic], handler)
	return func() error {
		return nil
	}, nil
}

func (inMemoryBus) Init(properties map[string]string) error {
	return nil
}

func (inMemoryBus) Close() error {
	return nil
}

func runHandlers(handlers []Handler, env *MessageEnvelope) {
	ctx := context.Background()
	for _, h := range handlers {
		h(ctx, env)
	}
}
