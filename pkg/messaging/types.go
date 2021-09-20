package messaging

import (
	"context"
	"time"
)

const (
	TopicKey = "topic"
)

type MessageEnvelope struct {
	Headers map[string]string `json:"headers"`
	Payload interface{}       `json:"payload"`
}

type UnsubscribeFunc func() error
type Handler func(ctx context.Context, msg *MessageEnvelope) error

type PublishRequest struct {
	PubsubName string
	Topic      string
	Data       interface{}
	Metadata   map[string]string
}

type SubscribeRequest struct {
	PubsubName string
	Topic      string
	Handler    Handler
	Options    *SubscriptionOptions
}

type SubscriptionOptions struct {
	Durable                *bool
	QGroup                 *bool
	MaxConcurrentMessages  *int32
	DeliverNewMessagesOnly *bool
	AckWaitTime            *time.Duration
}

type PublishRequestHandler func(ctx context.Context, request PublishRequest) error
type SubscribeRequestHandler func(ctx context.Context, request SubscribeRequest) (UnsubscribeFunc, error)
