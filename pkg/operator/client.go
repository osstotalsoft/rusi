package operator

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"k8s.io/klog/v2"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/custom-resource/configuration"
	"rusi/pkg/kube"
	operatorv1 "rusi/pkg/proto/operator/v1"
	"sync"
)

var conn *grpc.ClientConn

func newClient(ctx context.Context, address string) (cl operatorv1.RusiOperatorClient, err error) {
	var retryPolicy = `{
		"methodConfig": [{
		  "name": [{"service": "rusi.proto.operator.v1.RusiOperator"}],
		  "waitForReady": true,
		  "retryPolicy": {
			  "MaxAttempts": 4,
			  "InitialBackoff": ".01s",
			  "MaxBackoff": ".01s",
			  "BackoffMultiplier": 1.0,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }
		}]}`

	if conn == nil {
		conn, err = grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithDefaultServiceConfig(retryPolicy))
		if err != nil {
			return nil, err
		}
	}
	return operatorv1.NewRusiOperatorClient(conn), nil
}

func GetComponentsWatcher(ctx context.Context, address string, wg *sync.WaitGroup) func(context.Context) (<-chan components.Spec, error) {
	client, _ := newClient(ctx, address)
	return func(ctx context.Context) (<-chan components.Spec, error) {
		c := make(chan components.Spec)
		namespace := kube.GetCurrentNamespace()
		req := &operatorv1.WatchComponentsRequest{Namespace: namespace}
		stream, err := client.WatchComponents(ctx, req)
		if err != nil {
			return nil, err
		}
		go func() {
			wg.Add(1)
			defer wg.Done()
			defer close(c)
			for {
				select {
				case <-ctx.Done():
					klog.ErrorS(ctx.Err(), "watch components shutting down")
					return
				default:
					for {
						msg, err := stream.Recv()
						if err != nil {
							klog.ErrorS(err, "watch components grpc stream error")
							break
						}
						spec := components.Spec{}
						err = jsoniter.Unmarshal(msg.Data, &spec)
						if err != nil {
							klog.ErrorS(err, "unable to Unmarshal operator data ")
						}
						c <- spec
					}
					klog.Warning("watch components grpc stream closed, reconnecting...")
					stream, _ = client.WatchComponents(ctx, req)
				}
			}
		}()
		return c, nil
	}
}

func GetConfigurationWatcher(ctx context.Context, address, configName string, wg *sync.WaitGroup) func(context.Context) (<-chan configuration.Spec, error) {
	client, _ := newClient(ctx, address)
	return func(ctx context.Context) (<-chan configuration.Spec, error) {
		c := make(chan configuration.Spec)
		namespace := kube.GetCurrentNamespace()
		req := &operatorv1.WatchConfigurationRequest{ConfigName: configName, Namespace: namespace}
		stream, err := client.WatchConfiguration(ctx, req)
		if err != nil {
			return nil, err
		}
		go func() {
			wg.Add(1)
			defer wg.Done()
			defer close(c)
			for {
				select {
				case <-ctx.Done():
					klog.ErrorS(ctx.Err(), "watch configuration shutting down")
					return
				default:
					for {
						msg, err := stream.Recv()
						if err != nil {
							klog.ErrorS(err, "watch configuration grpc stream error")
							break
						}
						spec := configuration.Spec{}
						err = jsoniter.Unmarshal(msg.Data, &spec)
						if err != nil {
							klog.ErrorS(err, "unable to Unmarshal operator data ")
						}
						c <- spec
					}
					klog.Warning("watch configuration grpc stream closed, reconnecting ...")
					stream, _ = client.WatchConfiguration(ctx, req)
				}
			}
		}()
		return c, nil
	}
}

func IsOperatorClientAlive() bool {
	return conn != nil && conn.GetState() == connectivity.Ready
}
