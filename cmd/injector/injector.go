package main

import (
	"context"
	"flag"
	"k8s.io/klog/v2"
	"log"
	"net/http"
	"rusi/internal/kube"
	"rusi/pkg/injector"
)

func main() {
	//https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md
	klog.InitFlags(nil)
	kube.InitFlags(nil)
	injector.BindConfigFlags(nil)

	flag.Parse()
	defer klog.Flush()

	cfg, err := injector.GetConfig()
	if err != nil {
		klog.Fatalf("error getting config: %s", err)
	}
	ctx := context.Background()

	kubeClient := kube.GetKubeClient()
	var authUIDs []string

	if cfg.ValidateServiceAccount {
		authUIDs, err = injector.AllowedControllersServiceAccountUID(ctx, kubeClient)
		if err != nil {
			log.Fatalf("failed to get authentication uids from services accounts: %s", err)
		}
	}
	err = injector.NewInjector(authUIDs, cfg, kubeClient).Run(ctx)
	if err != http.ErrServerClosed {
		klog.Fatal(err)
	}
}
