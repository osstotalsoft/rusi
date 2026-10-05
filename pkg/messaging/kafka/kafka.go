package kafka

import (
	"context"
	"errors"
	"fmt"
	"rusi/pkg/healthcheck"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

// compulsory options
const (
	bootstrapServers = "bootstrapServers"
)

// optional options
const (
	groupID    = "groupId"
	consumerID = "consumerID" // passed in by rusi runtime
)

// streamIDHeader is the envelope header used as the Kafka message key (per-stream ordering)
const streamIDHeader = "nbb-streamId"

type kafkaPubSub struct {
	options  options
	config   *sarama.Config
	client   sarama.Client
	producer sarama.SyncProducer

	mu            sync.Mutex
	subscriptions map[int]messaging.CloseFunc
	nextSubID     int
}

// NewKafkaPubSub returns a new Kafka pub-sub implementation
func NewKafkaPubSub() messaging.PubSub {
	return &kafkaPubSub{}
}

func parseKafkaMetadata(properties map[string]string) (options, error) {
	m := options{useGroup: true, deliverNewOnly: true}

	if val, ok := properties[bootstrapServers]; ok && val != "" {
		for _, b := range strings.Split(val, ",") {
			if b = strings.TrimSpace(b); b != "" {
				m.brokers = append(m.brokers, b)
			}
		}
	}
	if len(m.brokers) == 0 {
		return m, errors.New("kafka error: missing bootstrapServers")
	}

	if val, ok := properties[groupID]; ok && val != "" {
		m.groupID = val
	} else if val, ok := properties[consumerID]; ok && val != "" {
		m.groupID = val
	} else {
		return m, errors.New("kafka error: missing groupId (or runtime consumerID)")
	}

	return m, nil
}

func newSaramaConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.AutoCommit.Enable = true
	return config
}

func (k *kafkaPubSub) Init(properties map[string]string) error {
	m, err := parseKafkaMetadata(properties)
	if err != nil {
		return err
	}
	k.options = m
	k.config = newSaramaConfig()

	client, err := sarama.NewClient(m.brokers, k.config)
	if err != nil {
		return fmt.Errorf("kafka: error connecting to %v: %w", m.brokers, err)
	}
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("kafka: error creating producer: %w", err)
	}
	k.client = client
	k.producer = producer
	klog.Infof("connected to kafka at %v", m.brokers)
	return nil
}

func (k *kafkaPubSub) Publish(topic string, msg *messaging.MessageEnvelope) error {
	msgBytes, err := serdes.MarshalMessageEnvelope(msg)
	if err != nil {
		return err
	}

	pm := &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(msgBytes)}
	if streamID, ok := msg.Headers[streamIDHeader]; ok && streamID != "" {
		pm.Key = sarama.StringEncoder(streamID)
	}

	klog.V(4).InfoS("Publishing message to Kafka", "topic", topic)
	if _, _, err = k.producer.SendMessage(pm); err != nil {
		return fmt.Errorf("kafka: error from publish: %w", err)
	}
	klog.V(4).InfoS("Published message to Kafka", "topic", topic, "message", *msg)
	return nil
}

func (k *kafkaPubSub) Subscribe(topic string, handler messaging.Handler, options *messaging.SubscriptionOptions) (messaging.CloseFunc, error) {
	mergedOptions := mergeGlobalAndSubscriptionOptions(k.options, options)

	group := mergedOptions.groupID
	if !mergedOptions.useGroup {
		// every subscriber instance gets every message
		group = group + "-" + uuid.NewString()
	}

	config := newSaramaConfig()
	if mergedOptions.deliverNewOnly {
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	} else {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	consumerGroup, err := sarama.NewConsumerGroup(mergedOptions.brokers, group, config)
	if err != nil {
		return nil, fmt.Errorf("kafka: subscribe error %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	h := &groupHandler{ctx: ctx, handler: handler}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for err := range consumerGroup.Errors() {
			klog.ErrorS(err, "kafka: consumer group error", "topic", topic, "group", group)
		}
	}()
	go func() {
		defer wg.Done()
		// Consume returns on every (eager) rebalance; loop until cancelled
		for ctx.Err() == nil {
			if err := consumerGroup.Consume(ctx, []string{topic}, h); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}
				klog.ErrorS(err, "kafka: consume error", "topic", topic, "group", group)
				select {
				case <-ctx.Done():
				case <-time.After(time.Second):
				}
			}
		}
	}()

	klog.InfoS("kafka: subscribed to", "topic", topic, "group", group, "deliverNewOnly", mergedOptions.deliverNewOnly)

	k.mu.Lock()
	if k.subscriptions == nil {
		k.subscriptions = map[int]messaging.CloseFunc{}
	}
	id := k.nextSubID
	k.nextSubID++
	var once sync.Once
	closeFn := func() (closeErr error) {
		once.Do(func() {
			cancel()
			closeErr = consumerGroup.Close()
			wg.Wait()
			k.mu.Lock()
			delete(k.subscriptions, id)
			k.mu.Unlock()
			klog.Infof("kafka: unsubscribed from topic %s", topic)
		})
		return
	}
	k.subscriptions[id] = closeFn
	k.mu.Unlock()
	return closeFn, nil
}

type groupHandler struct {
	ctx     context.Context
	handler messaging.Handler
}

func (h *groupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *groupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

// ConsumeClaim handles one partition sequentially, preserving per-partition order.
// Offsets are marked (and auto-committed) after the handler returns.
func (h *groupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case kafkaMsg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			msg, err := serdes.UnmarshalMessageEnvelope(kafkaMsg.Value)
			if err != nil {
				klog.ErrorS(err, "Error unmarshaling message", "topic", kafkaMsg.Topic, "data", kafkaMsg.Value)
			}
			if msg.Id == "" {
				msg.Id = strconv.FormatInt(kafkaMsg.Offset, 10)
			}
			klog.InfoS("Received message", "topic", kafkaMsg.Topic, "Id", msg.Id)

			if err = h.handler(h.ctx, &msg); err != nil {
				// Kafka offsets are per partition: there is no per-message redelivery like NATS ack.
				klog.ErrorS(err, "Error running subscriber pipeline", "topic", kafkaMsg.Topic, "offset", kafkaMsg.Offset)
			}
			session.MarkMessage(kafkaMsg, "")
		}
	}
}

func (k *kafkaPubSub) Close() error {
	var errs []error
	k.mu.Lock()
	subs := make([]messaging.CloseFunc, 0, len(k.subscriptions))
	for _, c := range k.subscriptions {
		subs = append(subs, c)
	}
	k.mu.Unlock()
	for _, c := range subs {
		errs = append(errs, c())
	}
	if k.producer != nil {
		errs = append(errs, k.producer.Close())
	}
	if k.client != nil && !k.client.Closed() {
		errs = append(errs, k.client.Close())
	}
	return errors.Join(errs...)
}

func (k *kafkaPubSub) IsHealthy(ctx context.Context) healthcheck.HealthResult {
	if k.client == nil || k.client.Closed() {
		return healthcheck.HealthResult{
			Status:      healthcheck.Unhealthy,
			Description: "kafka pubsub client is closed",
		}
	}
	if err := k.client.RefreshMetadata(); err != nil {
		return healthcheck.HealthResult{
			Status:      healthcheck.Unhealthy,
			Description: "kafka pubsub metadata refresh failed: " + err.Error(),
		}
	}
	return healthcheck.HealthyResult
}

func mergeGlobalAndSubscriptionOptions(globalOptions options, subscriptionOptions *messaging.SubscriptionOptions) options {
	mergedOptions := globalOptions
	if subscriptionOptions == nil {
		return mergedOptions
	}
	if subscriptionOptions.QGroup != nil {
		mergedOptions.useGroup = *subscriptionOptions.QGroup
	}
	if subscriptionOptions.DeliverNewMessagesOnly != nil {
		mergedOptions.deliverNewOnly = *subscriptionOptions.DeliverNewMessagesOnly
	}
	return mergedOptions
}
