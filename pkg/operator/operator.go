package operator

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/custom-resource/configuration"
	"rusi/pkg/kube"
	compv1 "rusi/pkg/operator/apis/components/v1alpha1"
	"rusi/pkg/operator/client/clientset/versioned"
	"strings"
)

func ListComponents() ([]components.Spec, error) {
	cfg := kube.GetConfig()
	client, _ := versioned.NewForConfig(cfg)

	ctx := context.Background()
	namespace := kube.GetCurrentNamespace()

	list, err := client.ComponentsV1alpha1().Components(namespace).List(ctx, v1.ListOptions{})
	var res []components.Spec
	if err != nil {
		return nil, err
	}
	for _, item := range list.Items {
		res = append(res, components.Spec{
			Name:     item.Name,
			Type:     item.Spec.Type,
			Version:  item.Spec.Version,
			Metadata: convertMetadataItemsToProperties(item.Spec.Metadata),
			Scopes:   item.Scopes,
		})
	}

	return res, nil
}

func GetConfiguration(name string) (configuration.Spec, error) {
	cfg := kube.GetConfig()
	client, _ := versioned.NewForConfig(cfg)
	spec := configuration.Spec{}

	ctx := context.Background()
	namespace := kube.GetCurrentNamespace()

	conf, err := client.ConfigurationV1alpha1().Configurations(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return spec, err
	}
	//deep copy
	temporaryVariable, _ := json.Marshal(conf.Spec)
	err = json.Unmarshal(temporaryVariable, &spec)

	return spec, err
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
