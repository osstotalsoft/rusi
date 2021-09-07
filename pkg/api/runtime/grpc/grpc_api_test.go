package grpc

import (
	"context"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"rusi/pkg/messaging"
	v1 "rusi/pkg/proto/runtime/v1"
	"testing"
)

func Test_RusiServer_Pubsub(t *testing.T) {

	store := make(map[string]messaging.MessageEnvelope)
	publishHandler := func(request messaging.PublishRequest) error {
		store[request.Topic] = messaging.MessageEnvelope{
			Headers: request.Metadata,
			Payload: string(request.Data),
		}
		return nil
	}
	subscribeHandler := func(request messaging.SubscribeRequest) (messaging.UnsubscribeFunc, error) {
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
		want             string
		wantErr          bool
	}{
		// TODO: Add test cases.
		{"test pubsub with one message",
			&v1.PublishRequest{
				PubsubName: "p1",
				Topic:      "t1",
				Data:       []byte("data1"),
				Metadata:   nil,
			},
			&v1.SubscribeRequest{
				PubsubName: "p1",
				Topic:      "t1",
			}, "{\"headers\":null,\"payload\":\"data1\"}", false,
		},
		{"test pubsub with one message and headers",
			&v1.PublishRequest{
				PubsubName: "p1",
				Topic:      "t1",
				Data:       []byte("data1"),
				Metadata:   map[string]string{"ip": "10"},
			},
			&v1.SubscribeRequest{
				PubsubName: "p1",
				Topic:      "t1",
			}, "{\"headers\":{\"ip\":\"10\"},\"payload\":\"data1\"}", false,
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
			}, "", true,
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
			if tt.want != string(msg.GetData()) {
				t.Errorf("Expected %s, got %s", tt.want, msg.GetData())
				return
			}
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
