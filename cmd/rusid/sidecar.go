package main

import (
	"flag"
	"k8s.io/klog/v2"
)

func main() {

	klog.InitFlags(nil)
	flag.Parse()
	klog.InfoS("Rusid is starting")
	defer klog.Flush()
}
