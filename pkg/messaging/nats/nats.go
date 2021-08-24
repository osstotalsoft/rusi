package natsstreaming

import (
	"context"
	"errors"
	"fmt"
	"k8s.io/klog/v2"
	"math/rand"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	"strconv"
	"time"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"github.com/nats-io/stan.go/pb"
)

// compulsory options
const (
	natsURL                = "natsURL"
	natsStreamingClusterID = "natsStreamingClusterID"
)

// subscription options (optional)
const (
	durableSubscriptionName = "durableSubscriptionName"
	startAtSequence         = "startAtSequence"
	startWithLastReceived   = "startWithLastReceived"
	deliverAll              = "deliverAll"
	deliverNew              = "deliverNew"
	startAtTimeDelta        = "startAtTimeDelta"
	startAtTime             = "startAtTime"
	startAtTimeFormat       = "startAtTimeFormat"
	ackWaitTime             = "ackWaitTime"
	maxInFlight             = "maxInFlight"
)

// valid values for subscription options
const (
	subscriptionTypeQueueGroup = "queue"
	subscriptionTypeTopic      = "topic"
	startWithLastReceivedTrue  = "true"
	deliverAllTrue             = "true"
	deliverNewTrue             = "true"
)

const (
	consumerID       = "consumerID" // passed in by Dapr runtime
	subscriptionType = "subscriptionType"
)

type natsStreamingPubSub struct {
	options          options
	natStreamingConn stan.Conn

	ctx    context.Context
	cancel context.CancelFunc
}

// NewNATSStreamingPubSub returns a new NATS Streaming pub-sub implementation
func NewNATSStreamingPubSub() messaging.PubSub {
	return &natsStreamingPubSub{}
}

func parseNATSStreamingMetadata(properties map[string]string) (options, error) {
	m := options{}
	if val, ok := properties[natsURL]; ok && val != "" {
		m.natsURL = val
	} else {
		return m, errors.New("nats-streaming error: missing nats URL")
	}
	if val, ok := properties[natsStreamingClusterID]; ok && val != "" {
		m.natsStreamingClusterID = val
	} else {
		return m, errors.New("nats-streaming error: missing nats streaming cluster ID")
	}

	if val, ok := properties[subscriptionType]; ok {
		if val == subscriptionTypeTopic || val == subscriptionTypeQueueGroup {
			m.subscriptionType = val
		} else {
			return m, errors.New("nats-streaming error: valid value for subscriptionType is topic or queue")
		}
	}

	if val, ok := properties[consumerID]; ok && val != "" {
		m.natsQueueGroupName = val
	} else {
		return m, errors.New("nats-streaming error: missing queue group name")
	}

	if val, ok := properties[durableSubscriptionName]; ok && val != "" {
		m.durableSubscriptionName = val
	}

	if val, ok := properties[ackWaitTime]; ok && val != "" {
		dur, err := time.ParseDuration(properties[ackWaitTime])
		if err != nil {
			return m, fmt.Errorf("nats-streaming error %s ", err)
		}
		m.ackWaitTime = dur
	}
	if val, ok := properties[maxInFlight]; ok && val != "" {
		max, err := strconv.ParseUint(properties[maxInFlight], 10, 64)
		if err != nil {
			return m, fmt.Errorf("nats-streaming error in parsemetadata for maxInFlight: %s ", err)
		}
		if max < 1 {
			return m, errors.New("nats-streaming error: maxInFlight should be equal to or more than 1")
		}
		m.maxInFlight = max
	}

	//nolint:nestif
	// subscription options - only one can be used
	if val, ok := properties[startAtSequence]; ok && val != "" {
		// nats streaming accepts a uint64 as sequence
		seq, err := strconv.ParseUint(properties[startAtSequence], 10, 64)
		if err != nil {
			return m, fmt.Errorf("nats-streaming error %s ", err)
		}
		if seq < 1 {
			return m, errors.New("nats-streaming error: startAtSequence should be equal to or more than 1")
		}
		m.startAtSequence = seq
	} else if val, ok := properties[startWithLastReceived]; ok {
		// only valid value is true
		if val == startWithLastReceivedTrue {
			m.startWithLastReceived = val
		} else {
			return m, errors.New("nats-streaming error: valid value for startWithLastReceived is true")
		}
	} else if val, ok := properties[deliverAll]; ok {
		// only valid value is true
		if val == deliverAllTrue {
			m.deliverAll = val
		} else {
			return m, errors.New("nats-streaming error: valid value for deliverAll is true")
		}
	} else if val, ok := properties[deliverNew]; ok {
		// only valid value is true
		if val == deliverNewTrue {
			m.deliverNew = val
		} else {
			return m, errors.New("nats-streaming error: valid value for deliverNew is true")
		}
	} else if val, ok := properties[startAtTimeDelta]; ok && val != "" {
		dur, err := time.ParseDuration(properties[startAtTimeDelta])
		if err != nil {
			return m, fmt.Errorf("nats-streaming error %s ", err)
		}
		m.startAtTimeDelta = dur
	} else if val, ok := properties[startAtTime]; ok && val != "" {
		m.startAtTime = val
		if val, ok := properties[startAtTimeFormat]; ok && val != "" {
			m.startAtTimeFormat = val
		} else {
			return m, errors.New("nats-streaming error: missing value for startAtTimeFormat")
		}
	}

	return m, nil
}

func (n *natsStreamingPubSub) Init(properties map[string]string) error {
	m, err := parseNATSStreamingMetadata(properties)
	if err != nil {
		return err
	}
	n.options = m
	clientID := genRandomString(20)
	opts := []nats.Option{nats.Name(clientID)}
	natsConn, err := nats.Connect(m.natsURL, opts...)
	if err != nil {
		return fmt.Errorf("nats-streaming: error connecting to nats server at %s: %s", m.natsURL, err)
	}
	natStreamingConn, err := stan.Connect(m.natsStreamingClusterID, clientID, stan.NatsConn(natsConn))
	if err != nil {
		return fmt.Errorf("nats-streaming: error connecting to nats streaming server %s: %s", m.natsStreamingClusterID, err)
	}
	klog.Infof("connected to natsstreaming at %s", m.natsURL)

	n.ctx, n.cancel = context.WithCancel(context.Background())
	n.natStreamingConn = natStreamingConn

	return nil
}

func (n *natsStreamingPubSub) Publish(topic string, msg *messaging.MessageEnvelope) error {

	msgBytes, err := serdes.Marshal(msg)
	if err != nil {
		return err
	}

	err = n.natStreamingConn.Publish(topic, msgBytes)
	if err != nil {
		return fmt.Errorf("nats-streaming: error from publish: %s", err)
	}

	return nil
}

func (n *natsStreamingPubSub) Subscribe(topic string, handler messaging.Handler) error {
	natStreamingsubscriptionOptions, err := n.subscriptionOptions()
	if err != nil {
		return fmt.Errorf("nats-streaming: error getting subscription options %s", err)
	}

	natsMsgHandler := func(natsMsg *stan.Msg) {

		klog.InfoS("Processing NATS Streaming message", " subject", natsMsg.Subject,
			"Sequence", natsMsg.Sequence)

		msg := messaging.MessageEnvelope{}
		err = serdes.Unmarshal(natsMsg.Data, &msg)
		if err != nil {
			klog.ErrorS(err, "Error unmarshaling message")
		}

		err = handler(&msg)
		if err == nil {
			// we only send a successful ACK if there is no error from Dapr runtime
			natsMsg.Ack()
		}
	}

	if n.options.subscriptionType == subscriptionTypeTopic {
		_, err = n.natStreamingConn.Subscribe(topic, natsMsgHandler, natStreamingsubscriptionOptions...)
	} else if n.options.subscriptionType == subscriptionTypeQueueGroup {
		_, err = n.natStreamingConn.QueueSubscribe(topic, n.options.natsQueueGroupName, natsMsgHandler, natStreamingsubscriptionOptions...)
	}

	if err != nil {
		return fmt.Errorf("nats-streaming: subscribe error %s", err)
	}
	if n.options.subscriptionType == subscriptionTypeTopic {
		klog.Infof("nats: subscribed to subject %s", topic)
	} else if n.options.subscriptionType == subscriptionTypeQueueGroup {
		klog.Infof("nats: subscribed to subject %s with queue group %s", topic, n.options.natsQueueGroupName)
	}

	return nil
}

func (n *natsStreamingPubSub) subscriptionOptions() ([]stan.SubscriptionOption, error) {
	var options []stan.SubscriptionOption

	if n.options.durableSubscriptionName != "" {
		options = append(options, stan.DurableName(n.options.durableSubscriptionName))
	}

	switch {
	case n.options.deliverNew == deliverNewTrue:
		options = append(options, stan.StartAt(pb.StartPosition_NewOnly))
	case n.options.startAtSequence >= 1: // messages index start from 1, this is a valid check
		options = append(options, stan.StartAtSequence(n.options.startAtSequence))
	case n.options.startWithLastReceived == startWithLastReceivedTrue:
		options = append(options, stan.StartWithLastReceived())
	case n.options.deliverAll == deliverAllTrue:
		options = append(options, stan.DeliverAllAvailable())
	case n.options.startAtTimeDelta > (1 * time.Nanosecond): // as long as its a valid time.Duration
		options = append(options, stan.StartAtTimeDelta(n.options.startAtTimeDelta))
	case n.options.startAtTime != "":
		if n.options.startAtTimeFormat != "" {
			startTime, err := time.Parse(n.options.startAtTimeFormat, n.options.startAtTime)
			if err != nil {
				return nil, err
			}
			options = append(options, stan.StartAtTime(startTime))
		}
	}

	// default is auto ACK. switching to manual ACK since processing errors need to be handled
	options = append(options, stan.SetManualAckMode())

	// check if set the ack options.
	if n.options.ackWaitTime > (1 * time.Nanosecond) {
		options = append(options, stan.AckWait(n.options.ackWaitTime))
	}
	if n.options.maxInFlight >= 1 {
		options = append(options, stan.MaxInflight(int(n.options.maxInFlight)))
	}

	return options, nil
}

const inputs = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// generates a random string of length 20
func genRandomString(n int) string {
	b := make([]byte, n)
	s := rand.NewSource(int64(time.Now().Nanosecond()))
	for i := range b {
		b[i] = inputs[s.Int63()%int64(len(inputs))]
	}
	clientID := string(b)

	return clientID
}

func (n *natsStreamingPubSub) Close() error {
	n.cancel()

	return n.natStreamingConn.Close()
}
