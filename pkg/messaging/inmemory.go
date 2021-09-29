package messaging

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"sync"
	"time"
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
	h := c.handlers[topic]
	c.mu.RUnlock()
	if env.Headers == nil {
		env.Headers = map[string]string{}
	}
	env.Headers["topic"] = topic
	println("Publish to topic " + topic)

	println("start runHandlers for topic " + topic)
	runHandlers(h, env)
	println("finish runHandlers for topic " + topic)

	return nil
}

func (c *inMemoryBus) Subscribe(topic string, handler Handler, options *SubscriptionOptions) (CloseFunc, error) {
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
	for i, h := range handlers {
		h := h
		i := i
		println(fmt.Sprintf("starting Handler %d with metadata %v", i, env.Headers))
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
			defer cancel()
			err := h(ctx, env)
			if err != nil {
				klog.ErrorS(err, "error")
				if err != context.Canceled {
					panic(err)
				}
			}
		}()
	}
}
