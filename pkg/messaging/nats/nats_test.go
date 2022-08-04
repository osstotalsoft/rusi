package natsstreaming

import (
	"reflect"
	"rusi/pkg/messaging"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseNATSStreamingForMetadataMandatoryOptionsMissing(t *testing.T) {
	type test struct {
		name       string
		properties map[string]string
	}
	tests := []test{
		{"nats URL missing", map[string]string{
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
		}},
		{"consumer ID missing", map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			subscriptionType:       "topic",
		}},
		{"cluster ID missing", map[string]string{
			natsURL:          "nats://foo.bar:4222",
			consumerID:       "consumer1",
			subscriptionType: "topic",
		}},
	}
	for _, _test := range tests {
		t.Run(_test.name, func(t *testing.T) {
			_, err := parseNATSStreamingMetadata(_test.properties)
			assert.NotEmpty(t, err)
		})
	}
}

func TestParseNATSStreamingMetadataForInvalidSubscriptionOptions(t *testing.T) {
	type test struct {
		name       string
		properties map[string]string
	}

	tests := []test{
		{"invalid value (less than 1) for startAtSequence", map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
			startAtSequence:        "0",
		}},
		{"non integer value for startAtSequence", map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
			startAtSequence:        "foo",
		}},
		{"startWithLastReceived is other than true", map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
			startWithLastReceived:  "foo",
		}},
		{"deliverAll is other than true", map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
			deliverAll:             "foo",
		}},
		{"deliverNew is other than true", map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
			deliverNew:             "foo",
		}},
		{"invalid value for startAtTimeDelta", map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
			startAtTimeDelta:       "foo",
		}},
		{"startAtTime provided without startAtTimeFormat", map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
			startAtTime:            "foo",
		}},
	}

	for _, _test := range tests {
		t.Run(_test.name, func(t *testing.T) {
			_, err := parseNATSStreamingMetadata(_test.properties)
			assert.NotEmpty(t, err)
		})
	}
}

func TestParseNATSStreamingMetadataForValidSubscriptionOptions(t *testing.T) {
	type test struct {
		name                  string
		properties            map[string]string
		expectedMetadataName  string
		expectedMetadataValue string
	}

	tests := []test{

		{
			"using startWithLastReceived",
			map[string]string{
				natsURL:                "nats://foo.bar:4222",
				natsStreamingClusterID: "testcluster",
				consumerID:             "consumer1",
				subscriptionType:       "topic",
				startWithLastReceived:  "true",
			},
			"startWithLastReceived", "true",
		},

		{
			"using deliverAll",
			map[string]string{
				natsURL:                "nats://foo.bar:4222",
				natsStreamingClusterID: "testcluster",
				consumerID:             "consumer1",
				subscriptionType:       "topic",
				deliverAll:             "true",
			},
			"deliverAll", "true",
		},

		{
			"using deliverNew",
			map[string]string{
				natsURL:                "nats://foo.bar:4222",
				natsStreamingClusterID: "testcluster",
				consumerID:             "consumer1",
				subscriptionType:       "topic",
				deliverNew:             "true",
			},
			"deliverNew", "true",
		},

		{
			"using startAtSequence",
			map[string]string{
				natsURL:                "nats://foo.bar:4222",
				natsStreamingClusterID: "testcluster",
				consumerID:             "consumer1",
				subscriptionType:       "topic",
				startAtSequence:        "42",
			},
			"startAtSequence", "42",
		},

		{
			"using startAtTimeDelta",
			map[string]string{
				natsURL:                "nats://foo.bar:4222",
				natsStreamingClusterID: "testcluster",
				consumerID:             "consumer1",
				subscriptionType:       "topic",
				startAtTimeDelta:       "1h",
			},
			"startAtTimeDelta", "1h",
		},
	}

	for _, _test := range tests {
		t.Run(_test.name, func(t *testing.T) {
			m, err := parseNATSStreamingMetadata(_test.properties)

			assert.NoError(t, err)

			assert.NotEmpty(t, m.natsURL)
			assert.NotEmpty(t, m.natsStreamingClusterID)
			assert.NotEmpty(t, m.subscriptionType)
			assert.NotEmpty(t, m.natsQueueGroupName)
			assert.NotEmpty(t, _test.expectedMetadataValue)

			assert.Equal(t, _test.properties[natsURL], m.natsURL)
			assert.Equal(t, _test.properties[natsStreamingClusterID], m.natsStreamingClusterID)
			assert.Equal(t, _test.properties[subscriptionType], m.subscriptionType)
			assert.Equal(t, _test.properties[consumerID], m.natsQueueGroupName)
			assert.Equal(t, _test.properties[_test.expectedMetadataName], _test.expectedMetadataValue)
		})
	}
}

func TestParseNATSStreamingMetadata(t *testing.T) {
	t.Run("mandatory metadata provided", func(t *testing.T) {
		fakeProperties := map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
		}
		m, err := parseNATSStreamingMetadata(fakeProperties)

		assert.NoError(t, err)
		assert.NotEmpty(t, m.natsURL)
		assert.NotEmpty(t, m.natsStreamingClusterID)
		assert.NotEmpty(t, m.natsQueueGroupName)
		assert.Equal(t, fakeProperties[natsURL], m.natsURL)
		assert.Equal(t, fakeProperties[natsStreamingClusterID], m.natsStreamingClusterID)
		assert.Equal(t, fakeProperties[consumerID], m.natsQueueGroupName)
	})

	t.Run("subscription type missing", func(t *testing.T) {
		fakeProperties := map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
		}
		_, err := parseNATSStreamingMetadata(fakeProperties)
		assert.Empty(t, err)
	})
	t.Run("invalid value for subscription type", func(t *testing.T) {
		fakeProperties := map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "baz",
		}
		_, err := parseNATSStreamingMetadata(fakeProperties)
		assert.NotEmpty(t, err)
	})
	t.Run("more than one subscription option provided", func(t *testing.T) {
		fakeProperties := map[string]string{
			natsURL:                "nats://foo.bar:4222",
			natsStreamingClusterID: "testcluster",
			consumerID:             "consumer1",
			subscriptionType:       "topic",
			startAtSequence:        "42",
			startWithLastReceived:  "true",
			deliverAll:             "true",
		}
		m, err := parseNATSStreamingMetadata(fakeProperties)
		assert.NoError(t, err)
		assert.NotEmpty(t, m.natsURL)
		assert.NotEmpty(t, m.natsStreamingClusterID)
		assert.NotEmpty(t, m.subscriptionType)
		assert.NotEmpty(t, m.natsQueueGroupName)
		assert.NotEmpty(t, m.startAtSequence)
		// startWithLastReceived ignored
		assert.Empty(t, m.startWithLastReceived)
		// deliverAll will be ignored
		assert.Empty(t, m.deliverAll)

		assert.Equal(t, fakeProperties[natsURL], m.natsURL)
		assert.Equal(t, fakeProperties[natsStreamingClusterID], m.natsStreamingClusterID)
		assert.Equal(t, fakeProperties[subscriptionType], m.subscriptionType)
		assert.Equal(t, fakeProperties[consumerID], m.natsQueueGroupName)
		assert.Equal(t, fakeProperties[startAtSequence], strconv.FormatUint(m.startAtSequence, 10))
	})
}

func TestSubscriptionOptionsForValidOptions(t *testing.T) {
	type test struct {
		name                    string
		options                 options
		expectedNumberOfOptions int
	}

	tests := []test{
		{"using durableSubscriptionName", options{durableSubscriptionName: "foobar"}, 2},
		{"durableSubscriptionName is empty", options{durableSubscriptionName: ""}, 1},
		{"using startAtSequence", options{startAtSequence: uint64(42)}, 2},
		{"using startWithLastReceived", options{startWithLastReceived: startWithLastReceivedTrue}, 2},
		{"using deliverAll", options{deliverAll: deliverAllTrue}, 2},
		{"using startAtTimeDelta", options{startAtTimeDelta: 1 * time.Hour}, 2},
		{"using startAtTime and startAtTimeFormat", options{startAtTime: "Feb 3, 2013 at 7:54pm (PST)", startAtTimeFormat: "Jan 2, 2006 at 3:04pm (MST)"}, 2},
		{"using manual ack with ackWaitTime", options{ackWaitTime: 30 * time.Second}, 2},
		{"using manual ack with maxInFlight", options{maxInFlight: uint64(42)}, 2},
	}

	for _, _test := range tests {
		t.Run(_test.name, func(t *testing.T) {
			natsStreaming := natsStreamingPubSub{options: _test.options}
			opts, err := stanSubscriptionOptions(natsStreaming.options)
			assert.Empty(t, err)
			assert.NotEmpty(t, opts)
			assert.Equal(t, _test.expectedNumberOfOptions, len(opts))
		})
	}
}

func TestSubscriptionOptionsForInvalidOptions(t *testing.T) {
	type test struct {
		name    string
		options options
	}

	tests := []test{
		{"startAtSequence is less than 1", options{startAtSequence: uint64(0)}},
		{"startWithLastReceived is other than true", options{startWithLastReceived: "foo"}},
		{"deliverAll is other than true", options{deliverAll: "foo"}},
		{"deliverNew is other than true", options{deliverNew: "foo"}},
		{"startAtTime is empty", options{startAtTime: "", startAtTimeFormat: "Jan 2, 2006 at 3:04pm (MST)"}},
		{"startAtTimeFormat is empty", options{startAtTime: "Feb 3, 2013 at 7:54pm (PST)", startAtTimeFormat: ""}},
	}

	for _, _test := range tests {
		t.Run(_test.name, func(t *testing.T) {
			natsStreaming := natsStreamingPubSub{options: _test.options}
			opts, err := stanSubscriptionOptions(natsStreaming.options)
			assert.Empty(t, err)
			assert.NotEmpty(t, opts)
			assert.Equal(t, 1, len(opts))
		})
	}
}

func TestSubscriptionOptions(t *testing.T) {
	// general
	t.Run("manual ACK option is present by default", func(t *testing.T) {
		natsStreaming := natsStreamingPubSub{options: options{}}
		opts, err := stanSubscriptionOptions(natsStreaming.options)
		assert.Empty(t, err)
		assert.NotEmpty(t, opts)
		assert.Equal(t, 1, len(opts))
	})

	t.Run("only one subscription option will be honored", func(t *testing.T) {
		m := options{deliverNew: deliverNewTrue, deliverAll: deliverAllTrue, startAtTimeDelta: 1 * time.Hour}
		natsStreaming := natsStreamingPubSub{options: m}
		opts, err := stanSubscriptionOptions(natsStreaming.options)
		assert.Empty(t, err)
		assert.NotEmpty(t, opts)
		assert.Equal(t, 2, len(opts))
	})

	// invalid subscription options

	t.Run("startAtTime is invalid", func(t *testing.T) {
		m := options{startAtTime: "foobar", startAtTimeFormat: "Jan 2, 2006 at 3:04pm (MST)"}
		natsStreaming := natsStreamingPubSub{options: m}
		opts, err := stanSubscriptionOptions(natsStreaming.options)
		assert.NotEmpty(t, err)
		assert.Nil(t, opts)
	})

	t.Run("startAtTimeFormat is invalid", func(t *testing.T) {
		m := options{startAtTime: "Feb 3, 2013 at 7:54pm (PST)", startAtTimeFormat: "foo"}

		natsStreaming := natsStreamingPubSub{options: m}
		opts, err := stanSubscriptionOptions(natsStreaming.options)
		assert.NotEmpty(t, err)
		assert.Nil(t, opts)
	})
}

func TestGenRandomString(t *testing.T) {
	t.Run("random client ID is not empty", func(t *testing.T) {
		clientID := genRandomString(20)
		assert.NotEmpty(t, clientID)
	})

	t.Run("random client ID is not nil", func(t *testing.T) {
		clientID := genRandomString(20)
		assert.NotNil(t, clientID)
	})

	t.Run("random client ID length is 20", func(t *testing.T) {
		clientID := genRandomString(20)
		assert.NotEmpty(t, clientID)
		assert.NotNil(t, clientID)
		assert.Equal(t, 20, len(clientID))
	})
}

func TestMergeGlobalAndSubscriptionOptions(t *testing.T) {
	globalSubscriptionOptions := options{
		natsURL:                 "natsUrl",
		natsStreamingClusterID:  "natsStreamingClusterID",
		subscriptionType:        "subscriptionType",
		natsQueueGroupName:      "natsQueueGroupName",
		durableSubscriptionName: "durableSubscriptionName",
		deliverNew:              deliverNewTrue,
		ackWaitTime:             time.Second,
		maxInFlight:             1,
	}
	durableFalse, emptyDurableName, qGroupFalse, qGroupTrue, maxConcurrentMessages10, ackWaitTime1Minute := false, "", false, true, int32(10), time.Minute

	type args struct {
		globalOptions       options
		subscriptionOptions *messaging.SubscriptionOptions
	}
	tests := []struct {
		name    string
		args    args
		want    func() options
		wantErr bool
	}{
		{"test nill subscription options", args{globalSubscriptionOptions, nil}, func() options { return globalSubscriptionOptions }, false},
		{"test empty subscription options", args{globalSubscriptionOptions, new(messaging.SubscriptionOptions)}, func() options { return globalSubscriptionOptions }, false},
		{"test subscription options with durable set", args{globalSubscriptionOptions, &messaging.SubscriptionOptions{Durable: &durableFalse}}, func() options {
			result := globalSubscriptionOptions
			result.durableSubscriptionName = emptyDurableName
			return result
		}, false},
		{"test subscription options with qGroup false", args{globalSubscriptionOptions, &messaging.SubscriptionOptions{QGroup: &qGroupFalse}}, func() options {
			result := globalSubscriptionOptions
			result.natsQueueGroupName = ""
			result.subscriptionType = subscriptionTypeTopic
			return result
		}, false},
		{"test subscription options with qGroup true and global options qGroupName set", args{globalSubscriptionOptions, &messaging.SubscriptionOptions{QGroup: &qGroupTrue}}, func() options {
			result := globalSubscriptionOptions
			result.subscriptionType = subscriptionTypeQueueGroup
			return result
		}, false},
		{"test subscription options with qGroup true and global options qGroupName not set", args{options{natsQueueGroupName: ""}, &messaging.SubscriptionOptions{QGroup: &qGroupTrue}}, func() options {
			return options{}
		}, true},
		{"test subscription options with maxConcurrentMessages set", args{globalSubscriptionOptions, &messaging.SubscriptionOptions{MaxConcurrentMessages: &maxConcurrentMessages10}}, func() options {
			result := globalSubscriptionOptions
			result.maxInFlight = uint64(maxConcurrentMessages10)
			return result
		}, false},
		{"test subscription options with ackWaitTime set", args{globalSubscriptionOptions, &messaging.SubscriptionOptions{AckWaitTime: &ackWaitTime1Minute}}, func() options {
			result := globalSubscriptionOptions
			result.ackWaitTime = ackWaitTime1Minute
			return result
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeGlobalAndSubscriptionOptions(tt.args.globalOptions, tt.args.subscriptionOptions)
			if err != nil && !tt.wantErr {
				t.Errorf("mergeGlobalAndSubscriptionOptions() error = %v", err)
				return
			}
			want := tt.want()
			if !reflect.DeepEqual(got, want) {
				t.Errorf("mergeGlobalAndSubscriptionOptions() got = %v, want %v", got, want)
			}
		})
	}
}
