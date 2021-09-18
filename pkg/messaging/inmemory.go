package messaging

import (
	"context"
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
	go func(hh []Handler) {
		runHandlers(hh, env)
	}(h)
	return nil
}

func (c *inMemoryBus) Subscribe(topic string, handler Handler, options *SubscriptionOptions) (UnsubscribeFunc, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[topic] = append(c.handlers[topic], handler)
	return func() error {
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
	for _, h := range handlers {
		h(ctx, env)
	}
}
