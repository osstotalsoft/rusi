package messaging

type Subscriber interface {
	Subscribe(topic string, handler Handler, options *SubscriptionOptions) (CloseFunc, error)
}
