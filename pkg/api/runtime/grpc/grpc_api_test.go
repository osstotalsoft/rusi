package grpc

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"k8s.io/klog/v2"
	"net"
	"reflect"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	v1 "rusi/pkg/proto/runtime/v1"
	"testing"
	"time"

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
	l := klog.Level(4)
	l.Set("4")
	store := messaging.NewInMemoryBus()
	publishHandler := func(ctx context.Context, request messaging.PublishRequest) error {
		return store.Publish(request.Topic, &messaging.MessageEnvelope{
			Id:              uuid.New().String(),
			Type:            request.Type,
			SpecVersion:     "1.0",
			DataContentType: request.DataContentType,
			Time:            time.Time{},
			Subject:         request.Topic,
			Headers:         request.Metadata,
			Payload:         request.Data,
		})
	}
	subscribeHandler := func(ctx context.Context, request messaging.SubscribeRequest) (messaging.CloseFunc, error) {
		return store.Subscribe(request.Topic, func(ctx context.Context, msg *messaging.MessageEnvelope) error {
			return request.Handler(ctx, msg)
		}, nil)
	}

	ctx := context.Background()
	server := startServer(t, ctx, publishHandler, subscribeHandler)

	tests := []struct {
		name             string
		publishRequest   *v1.PublishRequest
		subscribeRequest *v1.SubscriptionRequest
		wantData         string
		wantMetadata     map[string]string
		wantErr          bool
	}{
		{"test pubsub with one message",
			&v1.PublishRequest{
				PubsubName: "p1",
				Topic:      "t1",
				Data:       []byte("\"data1\""), //json
				Metadata:   nil,
			},
			&v1.SubscriptionRequest{
				PubsubName: "p1",
				Topic:      "t1",
			}, "data1", map[string]string{"topic": "t1"}, false,
		},
		{"test pubsub with one message and headers",
			&v1.PublishRequest{
				PubsubName: "p1",
				Topic:      "t1",
				Data:       []byte("\"data2\""), //json
				Metadata:   map[string]string{"ip": "10"},
			},
			&v1.SubscriptionRequest{
				PubsubName: "p1",
				Topic:      "t1",
			}, "data2", map[string]string{"ip": "10", "topic": "t1"}, false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, closer := newClient(ctx, t)
			stream, err := client.Subscribe(ctx)
			assert.NoError(t, err)
			err = stream.Send(createSubscribeRequest(tt.subscribeRequest))
			assert.NoError(t, err)
			//wait for subscribe
			err = waitInLoop(func() bool {
				return store.GetSubscribersCount(tt.subscribeRequest.Topic) == 1
			})
			assert.NoError(t, err, "subscribers count does not match")

			client.Publish(ctx, tt.publishRequest)
			msg1, err := stream.Recv() //blocks
			assert.NoError(t, err)
			var data string
			_ = serdes.Unmarshal(msg1.GetData(), &data)
			assert.Equal(t, tt.wantData, data)
			assert.Equal(t, tt.wantMetadata, msg1.GetMetadata())

			client.Publish(ctx, tt.publishRequest)
			msg2, err := stream.Recv() //blocks

			stream.Send(createAckRequest(msg1.Id, ""))
			stream.Send(createAckRequest(msg2.Id, ""))
			err = waitInLoop(func() bool {
				return store.IsDoneWorking()
			})
			assert.NoError(t, err, "not done processing all subscribers")
			closer()
			//wait for unsubscribe
			time.Sleep(100 * time.Millisecond)
		})
	}

	t.Run("refresh subscriber and maintain grpc stream", func(t *testing.T) {
		topic := "t4"
		client, closer := newClient(ctx, t)
		pubRequest := &v1.PublishRequest{
			PubsubName: "p1",
			Topic:      topic,
			Data:       []byte("\"data1\""),
			Metadata:   map[string]string{"ip": "10"},
		}
		stream, err := client.Subscribe(ctx)
		err = stream.Send(createSubscribeRequest(&v1.SubscriptionRequest{
			PubsubName: "p1",
			Topic:      topic,
		}))
		assert.NoError(t, err)
		//wait for subscribe
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscribers count does not match")
		client.Publish(ctx, pubRequest)
		var msg1, msg2 *v1.ReceivedMessage
		err = wait(func() error {
			msg1, err = stream.Recv() //blocks
			assert.NotNil(t, msg1)
			assert.NoError(t, err)
			return err
		}, "timeout waiting for receiving message on stream")
		assert.NoError(t, err)
		err = wait(func() error {
			return server.Refresh()
		}, "timeout waiting for server refresh")
		assert.NoError(t, err)
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err)
		client.Publish(ctx, pubRequest)
		msg2, err = stream.Recv() //blocks
		assert.NotNil(t, msg2)
		assert.NoError(t, err)
		stream.Send(createAckRequest(msg1.Id, ""))
		err = stream.Send(createAckRequest(msg2.Id, ""))
		assert.NoError(t, err)
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
		closer()
		//wait closing
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
	})

	t.Run("refresh subscriber when prev handler is not finished", func(t *testing.T) {
		topic := "t5"
		client, closer := newClient(ctx, t)
		pubRequest := &v1.PublishRequest{
			PubsubName: "p1",
			Topic:      topic,
			Data:       []byte("\"data1\""),
			Metadata:   map[string]string{"ip": "10"},
		}

		stream, err := client.Subscribe(ctx)
		err = stream.Send(createSubscribeRequest(&v1.SubscriptionRequest{
			PubsubName: "p1",
			Topic:      topic,
		}))
		assert.NoError(t, err)
		//wait for subscribe
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscribers count does not match")
		client.Publish(ctx, pubRequest)
		err = wait(func() error {
			return server.Refresh()
		}, "timeout waiting for server refresh")
		assert.NoError(t, err)
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscriber count does not match")
		err = wait(func() error {
			msg, err := stream.Recv() //blocks
			assert.NotNil(t, msg)
			assert.Nil(t, err)
			return err
		}, "timeout waiting for receiving message on stream")
		assert.NoError(t, err)
		client.Publish(ctx, pubRequest)
		assert.Nil(t, err)
		msg, err := stream.Recv() //blocks
		assert.NotNil(t, msg)
		assert.NoError(t, err)
		err = stream.Send(createAckRequest(msg.Id, ""))
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
		closer()
		//wait closing
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
	})

	t.Run("close subscription, resubscribe then refresh", func(t *testing.T) {
		topic := "t6"
		client, closer := newClient(ctx, t)
		ctx2, cancelFunc := context.WithCancel(ctx)
		subRequest := &v1.SubscriptionRequest{
			PubsubName: "p1",
			Topic:      topic,
		}
		pubRequest := &v1.PublishRequest{
			PubsubName: "p1",
			Topic:      topic,
			Data:       []byte("\"data1\""),
		}
		stream, err := client.Subscribe(ctx2)
		assert.NoError(t, err)
		err = stream.Send(createSubscribeRequest(subRequest))
		assert.NoError(t, err)
		//wait for subscribe
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscribers count does not match")
		cancelFunc()
		stream, err = client.Subscribe(ctx)
		assert.NoError(t, err)
		err = stream.Send(createSubscribeRequest(subRequest))
		assert.NoError(t, err)
		//wait for subscribe
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscribers count does not match")
		client.Publish(ctx, pubRequest)

		err = wait(func() error {
			return server.Refresh()
		}, "timeout waiting for server refresh")
		assert.NoError(t, err)
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscribers count does not match")

		err = wait(func() error {
			msg, err := stream.Recv() //blocks
			assert.NotNil(t, msg)
			assert.NoError(t, err)
			return err
		}, "timeout waiting for receiving message on stream")
		assert.NoError(t, err)
		closer()
		//wait closing
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
	})

	t.Run("subscribe and ack wrong message", func(t *testing.T) {
		topic := "t7"
		client, closer := newClient(ctx, t)
		subRequest := &v1.SubscriptionRequest{
			PubsubName: "p1",
			Topic:      topic,
		}
		stream, err := client.Subscribe(ctx)
		err = stream.Send(createSubscribeRequest(subRequest))
		assert.NoError(t, err)
		//wait for subscribe
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscribers count does not match")
		err = stream.Send(createAckRequest("fdssdfsdf", ""))
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		closer()
		//wait closing
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
	})

	t.Run("subscribe, receive two messages and ack both", func(t *testing.T) {
		topic := "t7"
		client, closer := newClient(ctx, t)
		subRequest := &v1.SubscriptionRequest{
			PubsubName: "p1",
			Topic:      topic,
		}
		pubRequest := &v1.PublishRequest{
			PubsubName: "p1",
			Topic:      topic,
			Data:       []byte("\"data1\""),
			Metadata:   map[string]string{"ip": "10"},
		}
		stream, err := client.Subscribe(ctx)
		err = stream.Send(createSubscribeRequest(subRequest))
		assert.NoError(t, err)
		//wait for subscribe
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscribers count does not match")

		client.Publish(ctx, pubRequest)
		client.Publish(ctx, pubRequest)
		var msg1, msg2 *v1.ReceivedMessage
		err = wait(func() error {
			msg1, _ = stream.Recv() //blocks
			msg2, _ = stream.Recv() //blocks
			return nil
		}, "timeout waiting for receiving message on stream")
		assert.NoError(t, err)
		err = stream.Send(createAckRequest(msg1.Id, ""))
		err = stream.Send(createAckRequest(msg2.Id, ""))
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err, "not done processing all subscribers")
		closer()
		//wait closing
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
	})

	t.Run("subscribe, receive messages ack with error", func(t *testing.T) {
		topic := "t7"
		client, closer := newClient(ctx, t)
		subRequest := &v1.SubscriptionRequest{
			PubsubName: "p1",
			Topic:      topic,
		}
		pubRequest := &v1.PublishRequest{
			PubsubName: "p1",
			Topic:      topic,
			Data:       []byte("\"data1\""),
		}
		stream, err := client.Subscribe(ctx)
		err = stream.Send(createSubscribeRequest(subRequest))
		assert.NoError(t, err)
		//wait for subscribe
		err = waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		assert.NoError(t, err, "subscribers count does not match")

		client.Publish(ctx, pubRequest)
		var msg *v1.ReceivedMessage
		err = wait(func() error {
			msg, _ = stream.Recv() //blocks
			return nil
		}, "timeout waiting for receiving message on stream")
		assert.NoError(t, err)
		err = stream.Send(createAckRequest(msg.Id, "error from processing"))
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err, "not done processing all subscribers")
		closer()
		//wait closing
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
	})

	t.Run("subscribe twice", func(t *testing.T) {
		topic := "t7"
		client, closer := newClient(ctx, t)
		subRequest := &v1.SubscriptionRequest{
			PubsubName: "p1",
			Topic:      topic,
		}
		pubRequest := &v1.PublishRequest{
			PubsubName: "p1",
			Topic:      topic,
			Data:       []byte("\"data1\""),
		}
		s1, _ := client.Subscribe(ctx)
		s1.Send(createSubscribeRequest(subRequest))
		s2, _ := client.Subscribe(ctx)
		s2.Send(createSubscribeRequest(subRequest))
		//wait for subscribe
		err := waitInLoop(func() bool {
			return store.GetSubscribersCount(topic) == 2
		})
		assert.NoError(t, err, "subscribers count does not match")

		client.Publish(ctx, pubRequest)
		err = wait(func() error {
			s1.Recv() //blocks
			s2.Recv() //blocks
			return nil
		}, "timeout waiting for receiving message on stream")
		assert.NoError(t, err)
		closer()
		//wait for unsubscribe
		err = waitInLoop(func() bool {
			return store.IsDoneWorking()
		})
		assert.NoError(t, err)
	})

	//check everything closed ok
	err := waitInLoop(func() bool {
		return store.IsDoneWorking()
	})
	if err != nil {
		t.Fatal(err)
	}
}

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func startServer(t *testing.T, ctx context.Context, publishHandler messaging.PublishRequestHandler,
	subscribeHandler messaging.SubscribeRequestHandler) *rusiServerImpl {
	server := newRusiServer(ctx, publishHandler, subscribeHandler)
	lis = bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	v1.RegisterRusiServer(grpcServer, server)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Server exited with error: %v", err)
		}
	}()
	return server
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
func waitInLoop(fun func() bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		time.Sleep(50 * time.Millisecond)
		if fun() {
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}
func wait(fun func() error, msg string) error {
	c := make(chan error)
	go func() {
		c <- fun()
	}()

	select {
	case <-time.After(10 * time.Second):
		return errors.New(msg)
	case e := <-c:
		return e
	}
}
func createSubscribeRequest(subscriptionRequest *v1.SubscriptionRequest) *v1.SubscribeRequest {
	return &v1.SubscribeRequest{RequestType: &v1.SubscribeRequest_SubscriptionRequest{
		SubscriptionRequest: subscriptionRequest}}
}
func createAckRequest(id, error string) *v1.SubscribeRequest {
	return &v1.SubscribeRequest{RequestType: &v1.SubscribeRequest_AckRequest{
		AckRequest: &v1.AckRequest{MessageId: id, Error: error}}}
}
