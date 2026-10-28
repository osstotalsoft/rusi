package jetstream

import "time"

type options struct {
	natsURL                 string
	consumerID              string
	durableSubscriptionName string
	queueGroup              bool
	deliverNew              string
	deliverAll              string
	connectWait             time.Duration
	ackWaitTime             time.Duration
	maxInFlight             int
}
