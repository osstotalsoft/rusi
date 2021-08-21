package messaging

type PubSub interface {
	Publisher
	Subscriber
	Init(properties map[string]string) error
	Close() error
}
