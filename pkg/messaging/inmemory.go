package messaging

import (
	"context"
	"fmt"
	"sync"
)

type inMemoryBus struct {
	handlers map[string][]Handler
	mu       sync.RWMutex
}

func NewInMemoryBus() *inMemoryBus {
	return &inMemoryBus{handlers: map[string][]Handler{}}
}

func (c *inMemoryBus) Publish(topic string, env *MessageEnvelope) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	h := c.handlers[topic]
	if env.Headers == nil {
		env.Headers = map[string]string{}
	}
	env.Headers["topic"] = topic
	println("Publish to topic " + topic)

	go func(hh []Handler) {
		println("start runHandlers for topic " + topic)
		runHandlers(hh, env)
		println("finish runHandlers for topic " + topic)

	}(h)
	return nil
}

func (c *inMemoryBus) Subscribe(topic string, handler Handler, options *SubscriptionOptions) (UnsubscribeFunc, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[topic] = append(c.handlers[topic], handler)
	println("Subscribed to topic " + topic)

	return func() error {
		println("unSubscribe from topic " + topic)
		return nil
	}, nil
}

func (*inMemoryBus) Init(properties map[string]string) error {
	return nil
}

func (*inMemoryBus) Close() error {
	return nil
}

func (c *inMemoryBus) GetSubscribersCount(topic string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.handlers[topic])
}

func runHandlers(handlers []Handler, env *MessageEnvelope) {
	ctx := context.Background()
	for i, h := range handlers {
		println(fmt.Sprintf("runHandler %d with metadata %v", i, env.Headers))
		h(ctx, env)
	}
}
