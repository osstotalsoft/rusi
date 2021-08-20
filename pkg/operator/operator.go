package operator

import (
	"context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"rusi/pkg/kube"
	"rusi/pkg/operator/apis/components/v1alpha1"
	"rusi/pkg/operator/client/clientset/versioned"
)

func ListComponents() *v1alpha1.ComponentList {
	cfg := kube.GetConfig()
	compClient, _ := versioned.NewForConfig(cfg)

	ctx := context.Background()
	dd, _ := compClient.ComponentsV1alpha1().Components("").
		List(ctx, v1.ListOptions{
			TypeMeta: v1.TypeMeta{
				Kind:       "",
				APIVersion: "",
			},
			LabelSelector:        "",
			FieldSelector:        "",
			Watch:                false,
			AllowWatchBookmarks:  false,
			ResourceVersion:      "",
			ResourceVersionMatch: "",
			TimeoutSeconds:       nil,
			Limit:                0,
			Continue:             "",
		})

	return dd
}
