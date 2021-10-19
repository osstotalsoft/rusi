package main

import (
	"flag"
	"k8s.io/klog/v2"
	"rusi/internal/kube"
	"rusi/pkg/operator"
)

func main() {
	//https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md
	klog.InitFlags(nil)
	kube.InitFlags(nil)
	defer klog.Flush()
	flag.Parse()

	operator.Run()
}
