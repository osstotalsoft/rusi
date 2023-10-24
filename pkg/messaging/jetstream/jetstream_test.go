package jetstream

import (
	"reflect"
	"rusi/pkg/messaging"
	"testing"
)

func Test_parseNATSStreamingMetadata(t *testing.T) {
	type args struct {
		properties map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    options
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNATSStreamingMetadata(tt.args.properties)
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

func Test_mergeGlobalAndSubscriptionOptions(t *testing.T) {
	type args struct {
		globalOptions       options
		subscriptionOptions *messaging.SubscriptionOptions
	}
	tests := []struct {
		name    string
		args    args
		want    options
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeGlobalAndSubscriptionOptions(tt.args.globalOptions, tt.args.subscriptionOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeGlobalAndSubscriptionOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeGlobalAndSubscriptionOptions() got = %v, want %v", got, tt.want)
			}
		})
	}
}
