package kafka

import (
	"context"
	"os"
	"rusi/pkg/healthcheck"
	"rusi/pkg/messaging"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKafkaMetadataMandatoryOptionsMissing(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]string
	}{
		{"bootstrapServers missing", map[string]string{consumerID: "consumer1"}},
		{"bootstrapServers blank", map[string]string{bootstrapServers: " , ", consumerID: "consumer1"}},
		{"groupId and consumerID missing", map[string]string{bootstrapServers: "localhost:9092"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseKafkaMetadata(tt.properties)
			assert.Error(t, err)
		})
	}
}

func TestParseKafkaMetadata(t *testing.T) {
	t.Run("brokers split and trimmed, group falls back to consumerID", func(t *testing.T) {
		m, err := parseKafkaMetadata(map[string]string{bootstrapServers: "b1:9092, b2:9092", consumerID: "consumer1"})
		require.NoError(t, err)
		assert.Equal(t, []string{"b1:9092", "b2:9092"}, m.brokers)
		assert.Equal(t, "consumer1", m.groupID)
		assert.True(t, m.useGroup)
		assert.True(t, m.deliverNewOnly)
	})
	t.Run("groupId wins over consumerID", func(t *testing.T) {
		m, err := parseKafkaMetadata(map[string]string{bootstrapServers: "b1:9092", groupID: "g1", consumerID: "consumer1"})
		require.NoError(t, err)
		assert.Equal(t, "g1", m.groupID)
	})
}

func TestMergeGlobalAndSubscriptionOptions(t *testing.T) {
	global := options{brokers: []string{"b1"}, groupID: "g1", useGroup: true, deliverNewOnly: true}
	f, tr := false, true

	assert.Equal(t, global, mergeGlobalAndSubscriptionOptions(global, nil))
	assert.Equal(t, global, mergeGlobalAndSubscriptionOptions(global, &messaging.SubscriptionOptions{}))

	got := mergeGlobalAndSubscriptionOptions(global, &messaging.SubscriptionOptions{QGroup: &f, DeliverNewMessagesOnly: &f})
	assert.False(t, got.useGroup)
	assert.False(t, got.deliverNewOnly)

	got = mergeGlobalAndSubscriptionOptions(options{groupID: "g1"}, &messaging.SubscriptionOptions{QGroup: &tr, DeliverNewMessagesOnly: &tr})
	assert.True(t, got.useGroup)
	assert.True(t, got.deliverNewOnly)
}

// TestPublishSubscribeRoundTrip runs against a live broker: KAFKA_BROKERS=localhost:9092 go test ./pkg/messaging/kafka/...
func TestPublishSubscribeRoundTrip(t *testing.T) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		t.Skip("KAFKA_BROKERS not set")
	}

	ps := NewKafkaPubSub()
	require.NoError(t, ps.Init(map[string]string{bootstrapServers: brokers, consumerID: "rusi-test-" + uuid.NewString()}))
	defer ps.Close()

	assert.EqualValues(t, healthcheck.Healthy, ps.(*kafkaPubSub).IsHealthy(context.Background()).Status)

	topic := "rusi-test-" + uuid.NewString()
	sent := &messaging.MessageEnvelope{Id: uuid.NewString(), Headers: map[string]string{streamIDHeader: "s1"}, Payload: map[string]interface{}{"n": 1.0}}
	require.NoError(t, ps.Publish(topic, sent))

	received := make(chan *messaging.MessageEnvelope, 1)
	deliverAll := false
	closeFn, err := ps.Subscribe(topic, func(ctx context.Context, msg *messaging.MessageEnvelope) error {
		received <- msg
		return nil
	}, &messaging.SubscriptionOptions{DeliverNewMessagesOnly: &deliverAll})
	require.NoError(t, err)
	defer closeFn()

	select {
	case msg := <-received:
		assert.Equal(t, sent.Id, msg.Id)
		assert.Equal(t, "s1", msg.Headers[streamIDHeader])
		assert.Equal(t, sent.Payload, msg.Payload)
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}
