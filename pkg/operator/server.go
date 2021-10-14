package operator

import (
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"k8s.io/klog/v2"
	"rusi/pkg/custom-resource/components"
	compv1 "rusi/pkg/operator/apis/components/v1alpha1"
	configv1 "rusi/pkg/operator/apis/configuration/v1alpha1"
	operatorv1 "rusi/pkg/proto/operator/v1"
	"strings"
)

type operatorServer struct {
	ow *objectWatcher
}

func (opsrv *operatorServer) WatchConfiguration(request *operatorv1.WatchConfigurationRequest, stream operatorv1.RusiOperator_WatchConfigurationServer) error {
	c := make(chan configv1.Configuration)
	opsrv.ow.addConfigurationListener(c)
	defer opsrv.ow.removeConfigurationListener(c)

	for {
		select {
		case obj := <-c:
			if obj.Namespace == request.Namespace && obj.Name == request.ConfigName {
				b, _ := jsoniter.Marshal(obj.Spec)
				stream.Send(&operatorv1.GenericItem{
					Data: b,
				})
			}
		case <-stream.Context().Done():
			klog.V(4).ErrorS(stream.Context().Err(), "grpc WatchConfiguration stream closed")
			return nil
		}
	}
}

func (opsrv *operatorServer) WatchComponents(request *operatorv1.WatchComponentsRequest, stream operatorv1.RusiOperator_WatchComponentsServer) error {
	c := make(chan compv1.Component)
	opsrv.ow.addComponentListener(c)
	defer opsrv.ow.removeComponentListener(c)

	for {
		select {
		case obj := <-c:
			if obj.Namespace == request.Namespace {
				b, _ := jsoniter.Marshal(convertToComponent(obj))
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

func convertToComponent(item compv1.Component) components.Spec {
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
