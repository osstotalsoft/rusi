package jetstream

import (
	"context"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"reflect"
	"rusi/pkg/messaging"
	"sync"
	"testing"
	"time"
)

func Ptr[T any](v T) *T {
	return &v
}

func Test_jetStreamPubSub_parseMetadata(t *testing.T) {
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
			"should parse ok",
			map[string]string{
				natsURL:    "nats://foo.bar:4222",
				consumerID: "consumerID",
			},
			options{natsURL: "nats://foo.bar:4222", connectWait: nats.DefaultTimeout,
				durableSubscriptionName: "consumerID"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMetadata(tt.properties)
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

func Test_jetStreamPubSub_Subscribe_serial_messages(t *testing.T) {
	srv := RunBasicJetStreamServer()
	defer shutdownJSServerAndRemoveStorage(t, srv)
	natsConn, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	defer natsConn.Close()

	js, err := jetstream.New(natsConn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	_, _ = js.CreateStream(ctx, jetstream.StreamConfig{Name: "foo", Subjects: []string{"test.*"}})

	sut := &jetStreamPubSub{
		options: options{
			durableSubscriptionName: "consumerID",
			ackWaitTime:             time.Second,
			maxInFlight:             1,
		},
		natsConn: natsConn,
		closed:   false,
		ctx:      ctx,
		cancel:   cancel,
	}

	subject := "test.topic"
	_ = sut.Publish(subject, &messaging.MessageEnvelope{Payload: "test1"})
	_ = sut.Publish(subject, &messaging.MessageEnvelope{Payload: "test2"})

	done := false
	var mu sync.Mutex
	wg := &sync.WaitGroup{}
	wg.Add(2)
	unsub, err := sut.Subscribe(subject, func(ctx context.Context, msg *messaging.MessageEnvelope) error {
		switch msg.Payload {
		case "test1":
			time.Sleep(time.Millisecond * 100)
			mu.Lock()
			done = true
			mu.Unlock()
		case "test2":
			mu.Lock()
			if !done {
				t.Errorf("Message1 not processed yet")
			}
			mu.Unlock()
		default:
			t.Errorf("Unexpected message payload: %s", msg.Payload)
		}
		wg.Done()
		return nil
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	doneCh := make(chan struct{})
	go func() { wg.Wait(); close(doneCh) }()
	select {
	case <-doneCh:
	case <-ctx.Done():
		t.Errorf("timeout waiting for message processing")
		return
	}

	err = unsub()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = sut.Close()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func Test_jetStreamPubSub_Subscribe_two_concurrent_messages(t *testing.T) {
	srv := RunBasicJetStreamServer()
	defer shutdownJSServerAndRemoveStorage(t, srv)
	natsConn, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	defer natsConn.Close()

	js, err := jetstream.New(natsConn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	_, _ = js.CreateStream(ctx, jetstream.StreamConfig{Name: "foo", Subjects: []string{"test.*"}})

	sut := &jetStreamPubSub{
		options: options{
			durableSubscriptionName: "consumerID",
			ackWaitTime:             time.Second,
			maxInFlight:             1,
		},
		natsConn: natsConn,
		closed:   false,
		ctx:      ctx,
		cancel:   cancel,
	}

	subject := "test.topic"
	_ = sut.Publish(subject, &messaging.MessageEnvelope{Payload: "test1"})
	_ = sut.Publish(subject, &messaging.MessageEnvelope{Payload: "test2"})
	_ = sut.Publish(subject, &messaging.MessageEnvelope{Payload: "test3"})

	done := false
	wg := &sync.WaitGroup{}
	wg.Add(3)
	var mu sync.Mutex
	unsub, err := sut.Subscribe(subject, func(ctx context.Context, msg *messaging.MessageEnvelope) error {
		switch msg.Payload {
		case "test1":
			time.Sleep(time.Millisecond * 100)
			mu.Lock()
			done = true
			mu.Unlock()
		case "test2":
			mu.Lock()
			if done {
				t.Errorf("Message2 should not wait for message1")
			}
			mu.Unlock()
			time.Sleep(time.Millisecond * 100)
		case "test3":
			mu.Lock()
			if !done {
				t.Errorf("Message3 has to wait for the other 2 to finish")
			}
			mu.Unlock()
		default:
			t.Errorf("Unexpected message payload: %s", msg.Payload)
		}
		wg.Done()
		return nil
	}, &messaging.SubscriptionOptions{MaxConcurrentMessages: Ptr(int32(2))})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	doneCh := make(chan struct{})
	go func() { wg.Wait(); close(doneCh) }()
	select {
	case <-doneCh:
	case <-ctx.Done():
		t.Errorf("timeout waiting for message processing")
		return
	}

	err = unsub()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = sut.Close()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
