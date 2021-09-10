package messaging

type Subscriber interface {
	Subscribe(topic string, handler Handler, options *SubscriptionOptions) (UnsubscribeFunc, error)
}
