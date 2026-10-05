package kafka

type options struct {
	brokers        []string
	groupID        string
	useGroup       bool
	deliverNewOnly bool
}
