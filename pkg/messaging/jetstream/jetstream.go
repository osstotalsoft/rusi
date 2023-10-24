package jetstream

import (
	"context"
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/stan.go"
	"rusi/pkg/healthcheck"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// compulsory options
const (
	natsURL = "natsURL"
)

// subscription options (optional)
const (
	deliverAll     = "deliverAll"
	deliverNew     = "deliverNew"
	ackWaitTime    = "ackWaitTime"
	maxInFlight    = "maxInFlight"
	connectWait    = "connectWait"
	commandsStream = "commandsStream"
	eventsStream   = "eventsStream"
)

// valid values for subscription options
const (
	deliverAllTrue = "true"
	deliverNewTrue = "true"
)

const (
	consumerID = "consumerID" // passed in by rusi runtime
)

type jetStreamPubSub struct {
	options  options
	natsConn *nats.Conn
	closed   bool

	ctx    context.Context
	cancel context.CancelFunc
}

// NewJetStreamPubSub returns a new NATS JetStream pub-sub implementation
func NewJetStreamPubSub() messaging.PubSub {
	return &jetStreamPubSub{}
}

func parseNATSStreamingMetadata(properties map[string]string) (options, error) {
	m := options{}
	m.connectWait = stan.DefaultConnectWait

	if val, ok := properties[natsURL]; ok && val != "" {
		m.natsURL = val
	} else {
		return m, errors.New("jetStream error: missing nats URL")
	}

	if val, ok := properties[consumerID]; ok && val != "" {
		m.durableSubscriptionName = val
	} else {
		return m, errors.New("jetStream error: missing consumer ID")
	}

	if val, ok := properties[ackWaitTime]; ok && val != "" {
		dur, err := time.ParseDuration(properties[ackWaitTime])
		if err != nil {
			return m, fmt.Errorf("jetStream error %s ", err)
		}
		m.ackWaitTime = dur
	}
	if val, ok := properties[maxInFlight]; ok && val != "" {
		maxInFlight, err := strconv.ParseInt(properties[maxInFlight], 10, 32)
		if err != nil {
			return m, fmt.Errorf("jetStream error in parsemetadata for maxInFlight: %s ", err)
		}
		if maxInFlight < 1 {
			return m, errors.New("jetStream error: maxInFlight should be equal to or more than 1")
		}
		m.maxInFlight = int(maxInFlight)
	}

	if val, ok := properties[connectWait]; ok && val != "" {
		wait, err := time.ParseDuration(properties[connectWait])
		if err != nil {
			return m, fmt.Errorf("jetStream error %s ", err)
		}
		m.connectWait = wait
	}

	if val, ok := properties[commandsStream]; ok && val != "" {
		m.commandsStream = val
	} else {
		return m, errors.New("jetStream error: missing commandsStream")
	}

	if val, ok := properties[eventsStream]; ok && val != "" {
		m.eventsStream = val
	} else {
		return m, errors.New("jetStream error: missing eventsStream")
	}

	//nolint:nestif
	// subscription options - only one can be used
	if val, ok := properties[deliverAll]; ok {
		// only valid value is true
		if val == deliverAllTrue {
			m.deliverAll = val
		} else {
			return m, errors.New("jetStream error: valid value for deliverAll is true")
		}
	} else if val, ok := properties[deliverNew]; ok {
		// only valid value is true
		if val == deliverNewTrue {
			m.deliverNew = val
		} else {
			return m, errors.New("jetStream error: valid value for deliverNew is true")
		}
	}

	return m, nil
}

func (n *jetStreamPubSub) Init(properties map[string]string) error {
	m, err := parseNATSStreamingMetadata(properties)
	if err != nil {
		return err
	}
	n.options = m

	n.natsConn, err = nats.Connect(m.natsURL, nats.Timeout(n.options.connectWait))
	if err != nil {
		return fmt.Errorf("jetStream: error connecting to nats server %s: %s", m.natsURL, err)
	}
	klog.Infof("connected to jetStream at %s", m.natsURL)

	n.ctx, n.cancel = context.WithCancel(context.Background())

	n.natsConn.SetReconnectHandler(func(conn *nats.Conn) {
		klog.Info("jetStream is reconnecting ...")
	})
	n.natsConn.SetClosedHandler(func(conn *nats.Conn) {
		klog.Info("jetStream connection is closed")
		n.closed = true
	})
	n.natsConn.SetDisconnectErrHandler(func(conn *nats.Conn, err error) {
		klog.ErrorS(err, "jetStream is disconnected")
	})

	return nil
}

func (n *jetStreamPubSub) Publish(topic string, msg *messaging.MessageEnvelope) error {

	msgBytes, err := serdes.MarshalMessageEnvelope(msg)
	if err != nil {
		return err
	}

	err = n.natsConn.Publish(topic, msgBytes)
	if err != nil {
		return fmt.Errorf("jetStream: error from publish: %s", err)
	}
	klog.V(4).InfoS("Published message to JetStream", "topic", topic, "message", *msg)
	return nil
}

func (n *jetStreamPubSub) Subscribe(topic string, handler messaging.Handler, options *messaging.SubscriptionOptions) (messaging.CloseFunc, error) {
	mergedOptions, err := mergeGlobalAndSubscriptionOptions(n.options, options)
	if err != nil {
		return nil, fmt.Errorf("jetStream: error getting subscription options %s", err)
	}

	js, _ := jetstream.New(n.natsConn)
	cc := jetstream.ConsumerConfig{AckPolicy: jetstream.AckExplicitPolicy}
	if mergedOptions.durableSubscriptionName != "" {
		cc.Durable = mergedOptions.durableSubscriptionName
	}
	// check if set the ack options.
	if mergedOptions.ackWaitTime > (1 * time.Nanosecond) {
		cc.AckWait = mergedOptions.ackWaitTime
	}
	if mergedOptions.maxInFlight >= 1 {
		cc.MaxAckPending = mergedOptions.maxInFlight
	}
	cc.FilterSubject = topic

	natsMsgHandler := func(natsMsg jetstream.Msg) {
		msg, err := serdes.UnmarshalMessageEnvelope(natsMsg.Data())
		if err != nil {
			klog.ErrorS(err, "Error unmarshaling message", "topic", natsMsg.Subject(), "data", natsMsg.Data())
		}
		if msg.Id == "" {
			m, _ := natsMsg.Metadata()
			msg.Id = strconv.FormatUint(m.Sequence.Stream, 10)
		}
		klog.InfoS("Received message", "topic", natsMsg.Subject(), "Id", msg.Id)

		//run handler concurrently
		go func() {
			err = handler(n.ctx, &msg)
			if err == nil {
				// we only send a successful ACK if there is no error
				_ = natsMsg.Ack()
				klog.V(4).InfoS("Manual ack", "topic", natsMsg.Subject(), "Id", msg.Id)
			} else {
				klog.ErrorS(err, "Error running subscriber pipeline, message was not ACK", "topic", natsMsg.Subject())
			}
		}()
	}

	isCommand := strings.Contains(strings.ToLower(topic), "commands.")
	stream := mergedOptions.eventsStream
	if isCommand {
		stream = mergedOptions.commandsStream
	}
	consumer, err := js.CreateOrUpdateConsumer(n.ctx, stream, cc)
	subs, err := consumer.Consume(natsMsgHandler)

	if err != nil {
		klog.ErrorS(err, "jetStream: subscribe error", "topic", topic)
	}

	if err != nil || subs == nil {
		return nil, fmt.Errorf("jetStream: subscribe error %s", err)
	}

	logSubscribe(cc, topic)

	return func() error {
		klog.Infof("jetStream: unsubscribed from topic %s", topic)
		subs.Stop()
		return nil
	}, nil
}

func logSubscribe(cc jetstream.ConsumerConfig, topic string) {
	klog.InfoS("jetStream: subscribed to", "ConsumerConfig", cc, "topic", topic)
}

func (n *jetStreamPubSub) Close() error {
	n.cancel()
	n.natsConn.Close()
	return nil
}

func (n *jetStreamPubSub) IsHealthy(ctx context.Context) healthcheck.HealthResult {
	if n.closed ||
		n.natsConn == nil ||
		n.natsConn.Status() != nats.CONNECTED {
		return healthcheck.HealthResult{
			Status:      healthcheck.Unhealthy,
			Description: "jetStream pubsub connection is closed",
		}
	}
	return healthcheck.HealthyResult
}

func mergeGlobalAndSubscriptionOptions(globalOptions options, subscriptionOptions *messaging.SubscriptionOptions) (options, error) {
	if subscriptionOptions == nil {
		return globalOptions, nil
	}

	mergedOptions := globalOptions

	if subscriptionOptions.Durable != nil && *subscriptionOptions.Durable {
		if mergedOptions.durableSubscriptionName == "" {
			return mergedOptions, errors.New("jetStream error: missing durable subscription name")
		}
	} else {
		mergedOptions.durableSubscriptionName = ""
	}

	if subscriptionOptions.MaxConcurrentMessages != nil {
		mergedOptions.maxInFlight = int(*subscriptionOptions.MaxConcurrentMessages)
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
