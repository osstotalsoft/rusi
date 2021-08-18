package messaging

type Publisher interface {
	Publish(topic string, env *MessageEnvelope) error
}
