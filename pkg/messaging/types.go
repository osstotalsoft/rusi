package messaging

import (
	"context"
	"time"
)

const (
	TopicKey = "topic"
)

//MessageEnvelope should be cloudevent compatible
//spec https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#type
type MessageEnvelope struct {
	Id              string    `json:"id"`
	Type            string    `json:"type"`
	SpecVersion     string    `json:"specversion"`
	DataContentType string    `json:"datacontenttype"`
	Time            time.Time `json:"time"`
	Subject         string    `json:"subject"`
	//data is not used yet
	//Data            interface{} `json:"data"`

	//For backward compatibility
	Headers map[string]string `json:"headers"`
	Payload interface{}       `json:"payload"`
}

type CloseFunc func() error
type AckHandler func(string, error)

type Handler func(ctx context.Context, msg *MessageEnvelope) error
type HandlerWithAck func(ctx context.Context, msg *MessageEnvelope, ackHandler AckHandler) error

type PublishRequest struct {
	PubsubName      string
	Topic           string
	Data            interface{}
	Type            string
	DataContentType string
	Metadata        map[string]string
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
type SubscribeRequestHandler func(ctx context.Context, request SubscribeRequest) (CloseFunc, error)
