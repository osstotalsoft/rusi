package operator

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
	"rusi/pkg/custom-resource/components"
	compv1 "rusi/pkg/operator/apis/components/v1alpha1"
	configv1 "rusi/pkg/operator/apis/configuration/v1alpha1"
	"rusi/pkg/operator/client/clientset/versioned"
	operatorv1 "rusi/pkg/proto/operator/v1"
	"strings"
)

type operatorServer struct {
	client *versioned.Clientset
}

func (opsrv *operatorServer) WatchConfiguration(request *operatorv1.WatchConfigurationRequest, server operatorv1.RusiOperator_WatchConfigurationServer) error {
	c, err := listConfiguration(server.Context(), opsrv.client, request.ConfigName, request.Namespace)
	if err != nil {
		return err
	}
	for {
		select {
		case data := <-c:
			b, _ := json.Marshal(data)
			server.Send(&operatorv1.GenericItem{
				Data: b,
			})
		case <-server.Context().Done():
			return nil
		}
	}
}

func (opsrv *operatorServer) WatchComponents(request *operatorv1.WatchComponentsRequest, server operatorv1.RusiOperator_WatchComponentsServer) error {
	c, err := listComponents(server.Context(), opsrv.client, request.Namespace)
	if err != nil {
		return err
	}
	for {
		select {
		case data := <-c:
			b, _ := json.Marshal(data)
			server.Send(&operatorv1.GenericItem{
				Data: b,
			})
		case <-server.Context().Done():
			return nil
		}
	}
}

func listComponents(ctx context.Context, client *versioned.Clientset, namespace string) (<-chan components.Spec, error) {
	c := make(chan components.Spec)
	watcher, err := client.ComponentsV1alpha1().Components(namespace).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return c, err
	}
	go func() {
		for conf := range watcher.ResultChan() {
			klog.V(4).InfoS("received component", "event", conf.Type, "object", conf.Object)
			if conf.Type == watch.Error {
				continue
			}
			item := conf.Object.(*compv1.Component)
			c <- components.Spec{
				Name:     item.Name,
				Type:     item.Spec.Type,
				Version:  item.Spec.Version,
				Metadata: convertMetadataItemsToProperties(item.Spec.Metadata),
				Scopes:   item.Scopes,
			}
		}
	}()
	return c, nil
}

func listConfiguration(ctx context.Context, client *versioned.Clientset, name string, namespace string) (<-chan configv1.ConfigurationSpec, error) {
	c := make(chan configv1.ConfigurationSpec)
	watcher, err := client.ConfigurationV1alpha1().Configurations(namespace).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return c, err
	}

	go func() {
		for conf := range watcher.ResultChan() {
			klog.V(4).InfoS("received configuration", "event", conf.Type, "object", conf.Object)
			if conf.Type == watch.Error {
				continue
			}
			obj := conf.Object.(*configv1.Configuration)
			if obj.Name != name {
				continue
			}
			c <- obj.Spec
		}
	}()

	return c, err
}

func convertMetadataItemsToProperties(items []compv1.MetadataItem) map[string]string {
	properties := map[string]string{}
	for _, c := range items {
		val := c.Value.String()
		for strings.Contains(val, "{uuid}") {
			val = strings.Replace(val, "{uuid}", uuid.New().String(), 1)
		}
		properties[c.Name] = val
	}
	return properties
}
