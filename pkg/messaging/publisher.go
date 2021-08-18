package messaging

type Publisher interface {
	Publish(env *MessageEnvelope) error
}
