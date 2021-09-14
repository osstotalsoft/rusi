package grpc

import (
	"context"
	"net"
	"reflect"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	v1 "rusi/pkg/proto/runtime/v1"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func Test_grpc_to_messaging_subscriptionOptions(t *testing.T) {
	durable, qGroup, maxConcurentMessages, ackWaitTime := true, true, int32(3), 10*time.Second

	type args struct {
		o *v1.SubscriptionOptions
	}
	tests := []struct {
		name string
		args args
		want *messaging.SubscriptionOptions
	}{
		{"test nill subscription options", args{nil}, nil},
		{"test empty subscription options", args{new(v1.SubscriptionOptions)}, new(messaging.SubscriptionOptions)},
		{"some subscription options", args{&v1.SubscriptionOptions{Durable: wrapperspb.Bool(durable)}}, &messaging.SubscriptionOptions{Durable: &durable}},
		{"some subscription options", args{&v1.SubscriptionOptions{Durable: wrapperspb.Bool(durable), QGroup: wrapperspb.Bool(qGroup), MaxConcurrentMessages: wrapperspb.Int32(int32(maxConcurentMessages))}}, &messaging.SubscriptionOptions{Durable: &durable, QGroup: &qGroup, MaxConcurrentMessages: &maxConcurentMessages}},
		{"some subscription options", args{&v1.SubscriptionOptions{Durable: wrapperspb.Bool(durable), QGroup: wrapperspb.Bool(qGroup), MaxConcurrentMessages: wrapperspb.Int32(int32(maxConcurentMessages)), AckWaitTime: durationpb.New(ackWaitTime)}}, &messaging.SubscriptionOptions{Durable: &durable, QGroup: &qGroup, MaxConcurrentMessages: &maxConcurentMessages, AckWaitTime: &ackWaitTime}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := messagingSubscriptionOptions(tt.args.o)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("messagingSubscriptionOptions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_RusiServer_Pubsub(t *testing.T) {

	store := make(map[string]messaging.MessageEnvelope)
	publishHandler := func(ctx context.Context, request messaging.PublishRequest) error {
		store[request.Topic] = messaging.MessageEnvelope{
			Headers: request.Metadata,
			Payload: request.Data,
		}
		return nil
	}
	subscribeHandler := func(ctx context.Context, request messaging.SubscribeRequest) (messaging.UnsubscribeFunc, error) {
		if msg, ok := store[request.Topic]; ok {
			request.Handler(context.Background(), &msg)
		} else {
			return nil, errors.New("invalid topic")
		}
		return func() error {
			delete(store, request.Topic)
			return nil
		}, nil
	}

	srv := NewRusiServer(publishHandler, subscribeHandler)
	startServer(srv, t)
	ctx := context.Background()
	client, close := newClient(ctx, t)
	defer close()

	tests := []struct {
		name             string
		publishRequest   *v1.PublishRequest
		subscribeRequest *v1.SubscribeRequest
		wantData         string
		wantMetadata     map[string]string
		wantErr          bool
	}{
		// TODO: Add test cases.
		{"test pubsub with one message",
			&v1.PublishRequest{
				PubsubName: "p1",
				Topic:      "t1",
				Data:       []byte("\"data1\""),
				Metadata:   nil,
			},
			&v1.SubscribeRequest{
				PubsubName: "p1",
				Topic:      "t1",
			}, "data1", map[string]string(nil), false,
		},
		{"test pubsub with one message and headers",
			&v1.PublishRequest{
				PubsubName: "p1",
				Topic:      "t1",
				Data:       []byte("\"data1\""),
				Metadata:   map[string]string{"ip": "10"},
			},
			&v1.SubscribeRequest{
				PubsubName: "p1",
				Topic:      "t1",
			}, "data1", map[string]string{"ip": "10"}, false,
		},
		{"test pub on topic t1 and sub on t2",
			&v1.PublishRequest{
				PubsubName: "p1",
				Topic:      "t1",
				Data:       []byte(""),
				Metadata:   nil,
			},
			&v1.SubscribeRequest{
				PubsubName: "p1",
				Topic:      "t2",
			}, "", nil, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := client.Publish(ctx, tt.publishRequest)
			if err != nil && !tt.wantErr {
				t.Errorf("Publish() error = %v", err)
				return
			}
			stream, err := client.Subscribe(ctx, tt.subscribeRequest)
			msg, err := stream.Recv() //blocks
			if err != nil && !tt.wantErr {
				t.Errorf("Subscribe() error = %v", err)
				return
			}
			var data string
			_ = serdes.Unmarshal(msg.GetData(), &data)
			assert.Equal(t, tt.wantData, data)
			assert.Equal(t, tt.wantMetadata, msg.GetMetadata())
		})
	}
}

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func startServer(server v1.RusiServer, t *testing.T) {
	lis = bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	v1.RegisterRusiServer(grpcServer, server)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Server exited with error: %v", err)
		}
	}()
}
func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func newClient(ctx context.Context, t *testing.T) (v1.RusiClient, func()) {
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	return v1.NewRusiClient(conn), func() {
		conn.Close()
	}
}
