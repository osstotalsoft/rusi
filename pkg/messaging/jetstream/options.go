package jetstream

import "time"

type options struct {
	natsURL                 string
	durableSubscriptionName string
	deliverNew              string
	deliverAll              string
	connectWait             time.Duration
	ackWaitTime             time.Duration
	maxInFlight             int
}
