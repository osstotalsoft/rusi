package messaging

type MessageEnvelope struct {
	Headers map[string]string `json:"headers"`
	Payload interface{}       `json:"payload"`
}

type UnsubscribeFunc func() error

type Handler func(env *MessageEnvelope) error

type PublishRequest struct {
	PubsubName string
	Topic      string
	Data       []byte
	Metadata   map[string]string
}

type SubscribeRequest struct {
	PubsubName string
	Topic      string
	Handler    Handler
}
