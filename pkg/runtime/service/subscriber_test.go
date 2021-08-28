package service

import (
	"reflect"
	"rusi/pkg/messaging"
	"testing"
)

func Test_subscriberService_StartSubscribing(t *testing.T) {
	type fields struct {
		subscriber messaging.Subscriber
	}
	type args struct {
		topic   string
		handler messaging.Handler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    messaging.UnsubscribeFunc
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &subscriberService{
				subscriber: tt.fields.subscriber,
			}
			got, err := srv.StartSubscribing(tt.args.topic, tt.args.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartSubscribing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StartSubscribing() got = %v, want %v", got, tt.want)
			}
		})
	}
}
