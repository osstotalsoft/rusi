package operator

import (
	"context"
	"github.com/google/uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"rusi/pkg/components"
	"rusi/pkg/kube"
	compv1 "rusi/pkg/operator/apis/components/v1alpha1"
	configv1 "rusi/pkg/operator/apis/configuration/v1alpha1"
	"rusi/pkg/operator/client/clientset/versioned"
	"strings"
)

func ListComponents() ([]components.Spec, error) {
	cfg := kube.GetConfig()
	compClient, _ := versioned.NewForConfig(cfg)

	ctx := context.Background()
	list, err := compClient.ComponentsV1alpha1().Components("").List(ctx, v1.ListOptions{})
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

func ListConfigurations(namespace string) (*configv1.ConfigurationList, error) {
	cfg := kube.GetConfig()
	compClient, _ := versioned.NewForConfig(cfg)

	ctx := context.Background()
	return compClient.ConfigurationV1alpha1().Configurations(namespace).List(ctx, v1.ListOptions{})
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
