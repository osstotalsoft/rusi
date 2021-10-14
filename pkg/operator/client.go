package operator

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/grpc"
	"io"
	"k8s.io/klog/v2"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/custom-resource/configuration"
	"rusi/pkg/kube"
	operatorv1 "rusi/pkg/proto/operator/v1"
)

func newClient(ctx context.Context, address string) (operatorv1.RusiOperatorClient, *grpc.ClientConn, error) {
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

	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithDefaultServiceConfig(retryPolicy))
	if err != nil {
		return nil, nil, err
	}
	return operatorv1.NewRusiOperatorClient(conn), conn, nil
}

func GetComponentsWatcher(address string) func(context.Context) (<-chan components.Spec, error) {
	return func(ctx context.Context) (<-chan components.Spec, error) {
		c := make(chan components.Spec)
		client, conn, err := newClient(ctx, address)
		if err != nil {
			return nil, err
		}
		namespace := kube.GetCurrentNamespace()
		stream, err := client.WatchComponents(ctx, &operatorv1.WatchComponentsRequest{Namespace: namespace})
		if err != nil {
			return nil, err
		}
		go func() {
			for {
				msg, err := stream.Recv()
				if err == io.EOF {
					//done <- true
					klog.Warning("watch components grpc stream EOF")
					break
				}
				if err != nil {
					//reconnect <- true
					klog.ErrorS(err, "watch components grpc stream lost")
					break
				}
				spec := components.Spec{}
				err = jsoniter.Unmarshal(msg.Data, &spec)
				if err != nil {
					klog.ErrorS(err, "unable to Unmarshal operator data ")
				}
				c <- spec
			}
			klog.Warning("watch components grpc stream closed")
			close(c)
			conn.Close()
		}()
		return c, nil
	}
}

func GetConfigurationWatcher(address string) func(context.Context, string) (<-chan configuration.Spec, error) {
	return func(ctx context.Context, name string) (<-chan configuration.Spec, error) {
		c := make(chan configuration.Spec)
		client, conn, err := newClient(ctx, address)
		if err != nil {
			return nil, err
		}
		namespace := kube.GetCurrentNamespace()
		stream, err := client.WatchConfiguration(ctx, &operatorv1.WatchConfigurationRequest{
			ConfigName: name, Namespace: namespace})

		if err != nil {
			return nil, err
		}
		go func() {
			for {
				msg, err := stream.Recv()
				if err == io.EOF {
					//done <- true
					klog.Warning("watch configuration grpc stream EOF")
					break
				}
				if err != nil {
					klog.ErrorS(err, "watch configuration grpc stream lost")
					break
				}
				spec := configuration.Spec{}
				err = jsoniter.Unmarshal(msg.Data, &spec)
				if err != nil {
					klog.ErrorS(err, "unable to Unmarshal operator data ")
				}
				c <- spec
			}
			klog.Warning("watch configuration grpc stream closed")
			close(c)
			conn.Close()
		}()
		return c, nil
	}
}
