package natsstreaming

import "time"

type options struct {
	natsURL                 string
	natsStreamingClusterID  string
	subscriptionType        string
	natsQueueGroupName      string
	durableSubscriptionName string
	startAtSequence         uint64
	startWithLastReceived   string
	deliverNew              string
	deliverAll              string
	startAtTimeDelta        time.Duration
	connectWait             time.Duration
	startAtTime             string
	startAtTimeFormat       string
	ackWaitTime             time.Duration
	maxInFlight             uint64
}
