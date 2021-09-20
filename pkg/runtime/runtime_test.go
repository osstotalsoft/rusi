package runtime

import (
	"context"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	runtime_api "rusi/pkg/api/runtime"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/custom-resource/components/pubsub"
	"rusi/pkg/custom-resource/configuration"
	"rusi/pkg/messaging"
	"testing"
	"time"
)

func Test_runtime_PublishHandler(t *testing.T) {

	appId := "test-app-id"
	mainCtx := context.Background()

	optionPubsub := WithPubSubs(
		pubsub.New("natsstreaming", func() messaging.PubSub {
			return messaging.NewInMemoryBus()
		}),
	)

	subRequestWrap := func(channel chan messaging.MessageEnvelope, request messaging.SubscribeRequest) messaging.SubscribeRequest {
		h := request.Handler
		request.Handler = func(ctx context.Context, msg messaging.MessageEnvelope) error {
			channel <- msg
			if h != nil {
				return h(ctx, msg)
			}
			return nil
		}
		return request
	}

	configLoader := func(channel chan configuration.Spec, err error, streamer func(channel chan configuration.Spec)) func(ctx context.Context, name string) (<-chan configuration.Spec, error) {
		return func(ctx context.Context, name string) (<-chan configuration.Spec, error) {
			go streamer(channel)
			return channel, err
		}
	}
	compLoader := func(channel chan components.Spec, err error, streamer func(channel chan components.Spec)) func(ctx context.Context) (<-chan components.Spec, error) {
		return func(ctx context.Context) (<-chan components.Spec, error) {
			go streamer(channel)
			return channel, err
		}
	}

	type fields struct {
		configErr      error
		compErr        error
		configStreamer func(channel chan configuration.Spec)
		compStreamer   func(channel chan components.Spec)
	}

	type args struct {
		publishRequest   messaging.PublishRequest
		subscribeRequest messaging.SubscribeRequest
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		runtimeErr   string
		publishErr   string
		subscribeErr string
	}{
		{
			"runtime subscribe with no pubsub component",
			fields{
				nil, nil,
				func(channel chan configuration.Spec) {
					channel <- configuration.Spec{}
				},
				func(channel chan components.Spec) {

				},
			},
			args{
				publishRequest: messaging.PublishRequest{
					PubsubName: "p1",
					Topic:      "t1",
					Data:       "data1",
				},
				subscribeRequest: messaging.SubscribeRequest{
					PubsubName: "p1",
					Topic:      "t1",
				},
			},
			"", "", "cannot find PubsubName named p1",
		},
		{
			"runtime publish with no pubsub component",
			fields{
				nil, nil,
				func(channel chan configuration.Spec) {
					channel <- configuration.Spec{}
				},
				func(channel chan components.Spec) {
					channel <- components.Spec{
						Name:     "p1",
						Type:     "pubsub.natsstreaming",
						Version:  "",
						Metadata: map[string]string{},
					}
				},
			},
			args{
				publishRequest: messaging.PublishRequest{
					PubsubName: "p2",
					Topic:      "t1",
					Data:       "data1",
				},
				subscribeRequest: messaging.SubscribeRequest{
					PubsubName: "p1",
					Topic:      "t1",
				},
			},
			"", "pubsub p2 not found", "",
		},
		{
			"runtime with error loading config",
			fields{
				errors.New("invalid path"),
				nil,
				func(channel chan configuration.Spec) {
					channel <- configuration.Spec{}
				},
				func(channel chan components.Spec) {

				},
			},
			args{
				publishRequest: messaging.PublishRequest{
					PubsubName: "p1",
					Topic:      "t1",
					Data:       "data1",
				},
				subscribeRequest: messaging.SubscribeRequest{
					PubsubName: "p1",
					Topic:      "t1",
				},
			},
			"invalid path", "", "",
		},
		{
			"runtime works even if subscriber has an error",
			fields{
				nil, nil,
				func(channel chan configuration.Spec) {
					channel <- configuration.Spec{}
				},
				func(channel chan components.Spec) {
					channel <- components.Spec{
						Name:     "p1",
						Type:     "pubsub.natsstreaming",
						Version:  "",
						Metadata: map[string]string{},
						Scopes:   nil,
					}
				},
			},
			args{
				publishRequest: messaging.PublishRequest{
					PubsubName: "p1",
					Topic:      "t1",
					Data:       "data1",
				},
				subscribeRequest: messaging.SubscribeRequest{
					PubsubName: "p1",
					Topic:      "t1",
					Handler: func(ctx context.Context, msg messaging.MessageEnvelope) error {
						return errors.New("subscribe error")
					},
					Options: nil,
				},
			},
			"", "", "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			compChannel := make(chan components.Spec)
			configChannel := make(chan configuration.Spec)
			api := runtime_api.NewTestApi()
			manager, _ := NewComponentsManager(mainCtx, appId, compLoader(compChannel,
				tt.fields.compErr, tt.fields.compStreamer), optionPubsub)
			rt, err := NewRuntime(mainCtx, Config{AppID: appId}, api, configLoader(configChannel,
				tt.fields.configErr, tt.fields.configStreamer), manager)
			if tt.runtimeErr != "" {
				assert.EqualError(t, err, tt.runtimeErr)
				assert.Nil(t, rt)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, manager)
			assert.NotNil(t, rt)

			newCtx := context.Background()
			//wait for components
			time.Sleep(500 * time.Millisecond)

			subsChan := make(chan messaging.MessageEnvelope)
			unsub, err := rt.SubscribeHandler(newCtx, subRequestWrap(subsChan, tt.args.subscribeRequest))
			if tt.subscribeErr != "" {
				assert.EqualError(t, err, tt.subscribeErr)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, unsub)
			if unsub != nil {
				defer assert.NoError(t, unsub())
			}
			err = rt.PublishHandler(newCtx, tt.args.publishRequest)
			if tt.publishErr != "" {
				assert.EqualError(t, err, tt.publishErr)
				return
			}
			assert.NoError(t, err)
			env := <-subsChan
			assert.Equal(t, tt.args.publishRequest.Data, env.Payload)
		})
	}

	t.Run("runtime should refresh on component change", func(t *testing.T) {
		compChannel := make(chan components.Spec)
		configChannel := make(chan configuration.Spec)
		c1 := configLoader(configChannel, nil, func(channel chan configuration.Spec) {
			channel <- configuration.Spec{}
		})

		c2 := compLoader(compChannel, nil, func(channel chan components.Spec) {
			channel <- components.Spec{
				Name:     "p1",
				Type:     "pubsub.natsstreaming",
				Version:  "",
				Metadata: map[string]string{},
				Scopes:   nil,
			}
			//wait
			time.Sleep(1 * time.Second)
			channel <- components.Spec{
				Name:     "p1",
				Type:     "pubsub.natsstreaming",
				Version:  "",
				Metadata: map[string]string{"data": "data"},
				Scopes:   nil,
			}
		})

		api := runtime_api.NewTestApi()
		manager, _ := NewComponentsManager(mainCtx, appId, c2, optionPubsub)
		NewRuntime(mainCtx, Config{AppID: appId}, api, c1, manager)

		newCtx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelFunc()

		//wait for components
		time.Sleep(500 * time.Millisecond)

		select {
		case <-api.RefreshChan:
		case <-newCtx.Done():
			t.Errorf("Iohohohoo 5 seconds have passed and your test is not done")
		}
	})

}
