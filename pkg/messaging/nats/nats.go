package natsstreaming

import (
	"context"
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	"math/rand"
	"rusi/pkg/healthcheck"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	"strconv"
	"time"

	"k8s.io/klog/v2"

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
	consumerID       = "consumerID" // passed in by rusi runtime
	subscriptionType = "subscriptionType"
)

type natsStreamingPubSub struct {
	options          options
	natStreamingConn stan.Conn
	closed           bool

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

	natStreamingConn, err := stan.Connect(m.natsStreamingClusterID, clientID, stan.NatsURL(m.natsURL),
		stan.SetConnectionLostHandler(func(conn stan.Conn, err error) {
			klog.ErrorS(err, "connection lost")
			n.closed = true
		}))

	if err != nil {
		return fmt.Errorf("nats-streaming: error connecting to nats streaming server %s: %s", m.natsStreamingClusterID, err)
	}
	klog.Infof("connected to natsstreaming at %s", m.natsURL)

	n.ctx, n.cancel = context.WithCancel(context.Background())
	n.natStreamingConn = natStreamingConn

	n.natStreamingConn.NatsConn().SetReconnectHandler(func(conn *nats.Conn) {
		klog.Info("nats is reconnecting ...")
	})
	n.natStreamingConn.NatsConn().SetClosedHandler(func(conn *nats.Conn) {
		klog.Info("nats connection is closed")
		n.closed = true
	})
	n.natStreamingConn.NatsConn().SetDisconnectErrHandler(func(conn *nats.Conn, err error) {
		klog.ErrorS(err, "nats is disconnected")
	})

	return nil
}

func (n *natsStreamingPubSub) Publish(topic string, msg *messaging.MessageEnvelope) error {

	msgBytes, err := serdes.Marshal(msg)
	if err != nil {
		return err
	}

	klog.V(4).InfoS("Publishing message to NATS")
	err = n.natStreamingConn.Publish(topic, msgBytes)
	if err != nil {
		return fmt.Errorf("nats-streaming: error from publish: %s", err)
	}
	klog.V(4).InfoS("Published message to NATS", "topic", topic, "message", *msg)
	return nil
}

func (n *natsStreamingPubSub) Subscribe(topic string, handler messaging.Handler, options *messaging.SubscriptionOptions) (messaging.CloseFunc, error) {
	mergedOptions, err := mergeGlobalAndSubscriptionOptions(n.options, options)
	if err != nil {
		return nil, fmt.Errorf("nats-streaming: error getting subscription options %s", err)
	}
	stanOptions, err := stanSubscriptionOptions(mergedOptions)
	if err != nil {
		return nil, fmt.Errorf("nats-streaming: error getting stan subscription options %s", err)
	}

	natsMsgHandler := func(natsMsg *stan.Msg) {
		msg := messaging.MessageEnvelope{}
		err := serdes.Unmarshal(natsMsg.Data, &msg)
		if err != nil {
			klog.ErrorS(err, "Error unmarshaling message")
		}
		if msg.Id == "" {
			msg.Id = strconv.FormatUint(natsMsg.Sequence, 10)
		}
		klog.InfoS("Received message", "topic", natsMsg.Subject, "Id", msg.Id)

		//run handler concurrently
		go func() {
			err = handler(n.ctx, &msg)
			if err == nil {
				// we only send a successful ACK if there is no error
				natsMsg.Ack()
				klog.V(4).InfoS("Manual ack", "topic", natsMsg.Subject, "Id", msg.Id)
			} else {
				klog.ErrorS(err, "Error running subscriber pipeline, message was not ACK", "topic", natsMsg.Subject)
			}
		}()
	}

	var subs stan.Subscription
	if mergedOptions.subscriptionType == subscriptionTypeTopic {
		subs, err = n.natStreamingConn.Subscribe(topic, natsMsgHandler, stanOptions...)
	} else if mergedOptions.subscriptionType == subscriptionTypeQueueGroup {
		subs, err = n.natStreamingConn.QueueSubscribe(topic, n.options.natsQueueGroupName, natsMsgHandler, stanOptions...)
	}

	if err != nil || subs == nil {
		return nil, fmt.Errorf("nats-streaming: subscribe error %s", err)
	}

	logSubscribe(stanOptions, mergedOptions.subscriptionType, n.options.natsQueueGroupName, topic)

	return func() error {
		klog.Infof("nats: unsubscribed from topic %s", topic)
		return subs.Close()
	}, nil
}

func logSubscribe(stanOptions []stan.SubscriptionOption, subscriptionType, queueGroupName, topic string) {
	opts := stan.SubscriptionOptions{}
	for _, option := range stanOptions {
		_ = option(&opts)
	}

	if subscriptionType == subscriptionTypeTopic {
		klog.InfoS("nats: subscribed to", "subject", topic, "options", opts)
	} else if subscriptionType == subscriptionTypeQueueGroup {
		klog.InfoS("nats: subscribed to", "topic", topic,
			"queue group", queueGroupName, "options", opts)
	}
}

func stanSubscriptionOptions(opts options) ([]stan.SubscriptionOption, error) {
	var options []stan.SubscriptionOption

	if opts.durableSubscriptionName != "" {
		options = append(options, stan.DurableName(opts.durableSubscriptionName))
	}

	switch {
	case opts.deliverNew == deliverNewTrue:
		options = append(options, stan.StartAt(pb.StartPosition_NewOnly))
	case opts.startAtSequence >= 1: // messages index start from 1, this is a valid check
		options = append(options, stan.StartAtSequence(opts.startAtSequence))
	case opts.startWithLastReceived == startWithLastReceivedTrue:
		options = append(options, stan.StartWithLastReceived())
	case opts.deliverAll == deliverAllTrue:
		options = append(options, stan.DeliverAllAvailable())
	case opts.startAtTimeDelta > (1 * time.Nanosecond): // as long as its a valid time.Duration
		options = append(options, stan.StartAtTimeDelta(opts.startAtTimeDelta))
	case opts.startAtTime != "":
		if opts.startAtTimeFormat != "" {
			startTime, err := time.Parse(opts.startAtTimeFormat, opts.startAtTime)
			if err != nil {
				return nil, err
			}
			options = append(options, stan.StartAtTime(startTime))
		}
	}

	// default is auto ACK. switching to manual ACK since processing errors need to be handled
	options = append(options, stan.SetManualAckMode())

	// check if set the ack options.
	if opts.ackWaitTime > (1 * time.Nanosecond) {
		options = append(options, stan.AckWait(opts.ackWaitTime))
	}
	if opts.maxInFlight >= 1 {
		options = append(options, stan.MaxInflight(int(opts.maxInFlight)))
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

func (n *natsStreamingPubSub) IsHealthy(ctx context.Context) healthcheck.HealthResult {
	if n.closed ||
		n.natStreamingConn == nil ||
		n.natStreamingConn.NatsConn() == nil ||
		n.natStreamingConn.NatsConn().Status() != nats.CONNECTED {
		return healthcheck.HealthResult{
			Status:      healthcheck.Unhealthy,
			Description: "nats pubsub connection is closed",
		}
	}
	return healthcheck.HealthyResult
}

func mergeGlobalAndSubscriptionOptions(globalOptions options, subscriptionOptions *messaging.SubscriptionOptions) (options, error) {
	if subscriptionOptions == nil {
		return globalOptions, nil
	}

	mergedOptions := globalOptions

	if subscriptionOptions.Durable != nil {
		if *subscriptionOptions.Durable {
			if mergedOptions.durableSubscriptionName == "" {
				mergedOptions.durableSubscriptionName = "durable"
			}
		} else {
			mergedOptions.durableSubscriptionName = ""
		}
	}

	if subscriptionOptions.QGroup != nil {
		if *subscriptionOptions.QGroup == false {
			mergedOptions.natsQueueGroupName = ""
		} else if mergedOptions.natsQueueGroupName == "" {
			return mergedOptions, errors.New("nats-streaming error: missing queue group name")
		}
	}
	if subscriptionOptions.MaxConcurrentMessages != nil {
		mergedOptions.maxInFlight = uint64(*subscriptionOptions.MaxConcurrentMessages)
	}
	if subscriptionOptions.DeliverNewMessagesOnly != nil {
		if *subscriptionOptions.DeliverNewMessagesOnly {
			mergedOptions.deliverNew = deliverNewTrue
		} else {
			mergedOptions.deliverAll = deliverAllTrue
		}
	}
	if subscriptionOptions.AckWaitTime != nil {
		mergedOptions.ackWaitTime = *subscriptionOptions.AckWaitTime
	}

	return mergedOptions, nil
}
