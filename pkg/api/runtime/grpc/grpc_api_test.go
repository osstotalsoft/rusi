package grpc

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"
	"net"
	"reflect"
	"rusi/pkg/messaging"
	"rusi/pkg/messaging/serdes"
	v1 "rusi/pkg/proto/runtime/v1"
	"strconv"
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

	ctx := context.Background()
	store := messaging.NewInMemoryBus()
	server, newClient := prepareNewInMemoryServer(t, ctx, store)

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
			return store.GetSubscribersCount(topic) == 0
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
			return store.GetSubscribersCount(topic) == 0
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
			return store.GetSubscribersCount(topic) == 0
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
	require.NoError(t, err)
}

func BenchmarkRusiServer_Subscribe(b *testing.B) {
	b.ReportAllocs()
	klog.ClearLogger()
	klog.SetLogger(logr.Discard())

	ctx := context.Background()
	store := messaging.NewInMemoryBus()
	_, newClient := prepareNewInMemoryServer(b, ctx, store)
	client, closer := newClient(ctx, b)
	defer closer()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		stream, err := client.Subscribe(ctx)
		topic := "topic_" + strconv.Itoa(i)
		b.StartTimer()
		err = stream.Send(createSubscribeRequest(&v1.SubscriptionRequest{
			PubsubName: "p1",
			Topic:      topic,
		}))
		require.NoError(b, err)
		err = waitInLoopInf(func() bool {
			return store.GetSubscribersCount(topic) == 1
		})
		require.NoError(b, err)
		_, err = client.Publish(ctx, &v1.PublishRequest{PubsubName: "p1", Topic: topic, Data: []byte("\"data1\"")})
		require.NoError(b, err)
		msg1, err := stream.Recv() //blocks
		require.NoError(b, err)
		err = stream.Send(createAckRequest(msg1.Id, ""))
		require.NoError(b, err)
		//wait for all handlers to complete
		err = waitInLoopInf(func() bool {
			return store.IsDoneWorking()
		})
		require.NoError(b, err)
	}
}

func prepareNewInMemoryServer(t testing.TB, ctx context.Context, pubsub messaging.PubSub) (server *rusiServerImpl,
	newClientFunc func(ctx context.Context, tt testing.TB) (v1.RusiClient, func())) {

	publishHandler := func(ctx context.Context, request messaging.PublishRequest) error {
		return pubsub.Publish(request.Topic, &messaging.MessageEnvelope{
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
		return pubsub.Subscribe(request.Topic, func(ctx context.Context, msg *messaging.MessageEnvelope) error {
			return request.Handler(ctx, msg)
		}, nil)
	}

	bufSize := 1024 * 1024
	lis := bufconn.Listen(bufSize)
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	newClientFunc = func(ctx context.Context, tt testing.TB) (v1.RusiClient, func()) {
		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
		if err != nil {
			tt.Fatalf("Failed to dial bufnet: %v", err)
		}
		return v1.NewRusiClient(conn), func() {
			conn.Close()
		}
	}
	server = startServer(t, ctx, publishHandler, subscribeHandler, lis)
	return server, newClientFunc
}

func startServer(t testing.TB, ctx context.Context,
	publishHandler messaging.PublishRequestHandler,
	subscribeHandler messaging.SubscribeRequestHandler, listener net.Listener) *rusiServerImpl {

	server := newRusiServer(ctx, publishHandler, subscribeHandler)
	grpcServer := grpc.NewServer()
	v1.RegisterRusiServer(grpcServer, server)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Fatalf("Server exited with error: %v", err)
		}
	}()
	return server
}
func waitInLoopInf(fun func() bool) error {
	for {
		if fun() {
			break
		}
	}
	return nil
}

func waitInLoop(fun func() bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		//time.Sleep(time.Millisecond)
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
