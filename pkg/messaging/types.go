package messaging

type MessageEnvelope struct {
	Headers map[string]string      `json:"headers"`
	Payload map[string]interface{} `json:"payload"`
}

type Handler func(env *MessageEnvelope) error
