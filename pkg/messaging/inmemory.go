package messaging

import (
	"context"
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"
	"sync"
	"sync/atomic"
	"time"
)

type inMemoryBus struct {
	handlers       map[string][]*Handler
	mu             sync.RWMutex
	workingCounter *int32
}

func NewInMemoryBus() *inMemoryBus {
	return &inMemoryBus{handlers: map[string][]*Handler{}, workingCounter: new(int32)}
}

func (c *inMemoryBus) Publish(topic string, env *MessageEnvelope) error {
	c.mu.RLock()
	h := c.handlers[topic]
	c.mu.RUnlock()
	if env.Headers == nil {
		env.Headers = map[string]string{}
	}
	env.Headers["topic"] = topic
	klog.InfoS("Publish to topic " + topic)

	go c.runHandlers(h, env)
	return nil
}

func (c *inMemoryBus) Subscribe(topic string, handler Handler, options *SubscriptionOptions) (CloseFunc, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	handlerP := &handler
	c.handlers[topic] = append(c.handlers[topic], handlerP)
	klog.InfoS("Subscribed to topic " + topic)

	return func() error {
		c.mu.Lock()
		defer c.mu.Unlock()
		for i, h := range c.handlers[topic] {
			if h == handlerP {
				c.handlers[topic][i] = c.handlers[topic][len(c.handlers[topic])-1]
				c.handlers[topic] = c.handlers[topic][:len(c.handlers[topic])-1]
				break
			}
		}
		klog.InfoS("unSubscribe from topic " + topic)
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

func (c *inMemoryBus) runHandlers(handlers []*Handler, env *MessageEnvelope) error {
	atomic.AddInt32(c.workingCounter, 1)
	defer atomic.AddInt32(c.workingCounter, -1)

	klog.InfoS("start runHandlers for topic " + env.Subject)
	eg := errgroup.Group{}
	c.mu.RLock()
	for i, h := range handlers {
		h := h
		i := i
		klog.Infof("starting Handler %d with metadata %v", i, env.Headers)
		eg.Go(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			return (*h)(ctx, env) //run handler blocks
		})
	}
	c.mu.RUnlock()
	err := eg.Wait()
	if err != nil {
		klog.ErrorS(err, "error running handlers")
	} else {
		klog.InfoS("finish runHandlers for topic " + env.Subject)
	}
	return err
}
