package jetstream

import (
	"github.com/nats-io/nats.go"
	"reflect"
	"testing"
)

func Test_parseNATSStreamingMetadata(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]string
		want       options
		wantErr    bool
	}{
		{
			"missing url",
			map[string]string{},
			options{connectWait: nats.DefaultTimeout},
			true,
		},
		{
			"missing consumer id",
			map[string]string{
				natsURL: "nats://foo.bar:4222",
			},
			options{natsURL: "nats://foo.bar:4222", connectWait: nats.DefaultTimeout},
			true,
		},
		{
			"missing commands stream",
			map[string]string{
				natsURL:    "nats://foo.bar:4222",
				consumerID: "consumerID",
			},
			options{natsURL: "nats://foo.bar:4222", connectWait: nats.DefaultTimeout,
				durableSubscriptionName: "consumerID"},
			true,
		},
		{
			"missing events stream",
			map[string]string{
				natsURL:        "nats://foo.bar:4222",
				consumerID:     "consumerID",
				commandsStream: "commandsStream",
			},
			options{natsURL: "nats://foo.bar:4222", connectWait: nats.DefaultTimeout,
				durableSubscriptionName: "consumerID", commandsStream: "commandsStream"},
			true,
		},
		{
			"should parse ok",
			map[string]string{
				natsURL:        "nats://foo.bar:4222",
				consumerID:     "consumerID",
				commandsStream: "commandsStream",
				eventsStream:   "eventsStream",
			},
			options{natsURL: "nats://foo.bar:4222", connectWait: nats.DefaultTimeout,
				durableSubscriptionName: "consumerID", commandsStream: "commandsStream",
				eventsStream: "eventsStream"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNATSStreamingMetadata(tt.properties)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNATSStreamingMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseNATSStreamingMetadata() got = %v, want %v", got, tt.want)
			}
		})
	}
}
