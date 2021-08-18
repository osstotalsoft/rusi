package messaging

type PubSub interface {
	Publisher
	Subscriber
}
