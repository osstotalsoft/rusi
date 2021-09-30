package messaging

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"sync"
	"sync/atomic"
	"time"
)

type inMemoryBus struct {
	handlers       map[string][]Handler
	mu             sync.RWMutex
	workingCounter *int32
}

func NewInMemoryBus() *inMemoryBus {
	return &inMemoryBus{handlers: map[string][]Handler{}, workingCounter: new(int32)}
}

func (c *inMemoryBus) Publish(topic string, env *MessageEnvelope) error {
	atomic.AddInt32(c.workingCounter, 1)
	defer atomic.AddInt32(c.workingCounter, -1)
	c.mu.RLock()
	h := c.handlers[topic]
	c.mu.RUnlock()
	if env.Headers == nil {
		env.Headers = map[string]string{}
	}
	env.Headers["topic"] = topic
	println("Publish to topic " + topic)

	println("start runHandlers for topic " + topic)
	c.runHandlers(h, env)
	println("finish runHandlers for topic " + topic)

	return nil
}

func (c *inMemoryBus) Subscribe(topic string, handler Handler, options *SubscriptionOptions) (CloseFunc, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[topic] = append(c.handlers[topic], handler)
	position := len(c.handlers[topic]) - 1
	println("Subscribed to topic " + topic)

	return func() error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.handlers[topic] = append(c.handlers[topic][:position], c.handlers[topic][position+1:]...)

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

func (c *inMemoryBus) IsDoneWorking() bool {
	return atomic.LoadInt32(c.workingCounter) == 0
}
func (c *inMemoryBus) GetSubscribersCount(topic string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.handlers[topic])
}

func (c *inMemoryBus) runHandlers(handlers []Handler, env *MessageEnvelope) {
	sg := sync.WaitGroup{}
	sg.Add(len(handlers))
	c.mu.RLock()
	for i, h := range handlers {
		h := h
		i := i
		println(fmt.Sprintf("starting Handler %d with metadata %v", i, env.Headers))
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
			defer cancel()
			defer sg.Done()
			err := h(ctx, env)
			if err != nil {
				klog.ErrorS(err, "error")
			}
		}()
	}
	c.mu.RUnlock()
	sg.Wait()
}
