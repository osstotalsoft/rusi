package main

import (
	"context"
	"flag"
	"k8s.io/klog/v2"
	"log"
	"net/http"
	"rusi/pkg/injector"
	"rusi/pkg/kube"
)

func main() {
	klog.InitFlags(nil)
	kube.InitFlags(nil)

	flag.Parse()
	defer klog.Flush()

	cfg, err := injector.GetConfig()
	if err != nil {
		klog.Fatalf("error getting config: %s", err)
	}
	ctx := context.Background()

	kubeClient := kube.GetKubeClient()
	uids, err := injector.AllowedControllersServiceAccountUID(ctx, kubeClient)
	if err != nil {
		log.Fatalf("failed to get authentication uids from services accounts: %s", err)
	}

	err = injector.NewInjector(uids, cfg, kubeClient).Run(ctx)
	if err != http.ErrServerClosed {
		klog.Fatal(err)
	}
}