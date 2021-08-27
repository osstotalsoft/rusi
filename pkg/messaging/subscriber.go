package messaging

type Subscriber interface {
	Subscribe(topic string, handler Handler) (UnsubscribeFunc, error)
}
