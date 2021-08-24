package messaging

type MessageEnvelope struct {
	Headers map[string]string `json:"headers"`
	Payload interface{}       `json:"payload"`
}

type Handler func(env *MessageEnvelope) error
