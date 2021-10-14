package operator

import (
	"encoding/json"
	"errors"
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

func (opsrv *operatorServer) WatchConfiguration(request *operatorv1.WatchConfigurationRequest, stream operatorv1.RusiOperator_WatchConfigurationServer) error {
	watcher, err := opsrv.client.ConfigurationV1alpha1().Configurations(request.Namespace).Watch(stream.Context(), v1.ListOptions{})
	if err != nil {
		return err
	}
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return errors.New("kubernetes configuration watcher closed")
			}
			klog.V(4).InfoS("received configuration", "event", event.Type, "object", event.Object)
			if event.Type != watch.Error {
				obj := event.Object.(*configv1.Configuration)
				if obj.Name == request.ConfigName {
					b, _ := json.Marshal(obj.Spec)
					stream.Send(&operatorv1.GenericItem{
						Data: b,
					})
				}
			}
		case <-stream.Context().Done():
			klog.V(4).ErrorS(stream.Context().Err(), "grpc WatchConfiguration stream closed")
			return nil
		}
	}
}

func (opsrv *operatorServer) WatchComponents(request *operatorv1.WatchComponentsRequest, stream operatorv1.RusiOperator_WatchComponentsServer) error {
	watcher, err := opsrv.client.ComponentsV1alpha1().Components(request.Namespace).Watch(stream.Context(), v1.ListOptions{})
	if err != nil {
		return err
	}
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return errors.New("kubernetes components watcher closed")
			}
			klog.V(4).InfoS("received component", "event", event.Type, "object", event.Object)
			if event.Type != watch.Error {
				b, _ := json.Marshal(convertToComponent(event))
				stream.Send(&operatorv1.GenericItem{
					Data: b,
				})
			}
		case <-stream.Context().Done():
			klog.V(4).ErrorS(stream.Context().Err(), "grpc WatchComponents stream closed")
			return nil
		}
	}
}
func convertToComponent(event watch.Event) components.Spec {
	item := event.Object.(*compv1.Component)
	return components.Spec{
		Name:     item.Name,
		Type:     item.Spec.Type,
		Version:  item.Spec.Version,
		Metadata: convertMetadataItemsToProperties(item.Spec.Metadata),
		Scopes:   item.Scopes,
	}
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
