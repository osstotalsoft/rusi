package kube

import (
	"flag"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
)

var (
	clientSet      *kubernetes.Clientset
	kubeConfig     *rest.Config
	kubeConfigPath string
)

func initKubeConfig() {
	kubeConfig = GetConfig()
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.Fatal(err)
	}

	clientSet = clientset
}

// GetConfig gets a kubernetes rest config.
func GetConfig() *rest.Config {
	if kubeConfig != nil {
		return kubeConfig
	}

	conf, err := rest.InClusterConfig()
	if err != nil {
		conf, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			panic(err)
		}
	}

	return conf
}

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeConfigPath, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeConfigPath, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
}

// GetKubeClient gets a kubernetes client.
func GetKubeClient() *kubernetes.Clientset {
	if clientSet == nil {
		initKubeConfig()
	}

	return clientSet
}

func GetCurrentNamespace() string {
	// This way assumes you've set the POD_NAMESPACE environment variable using the downward API.
	// This check has to be done first for backwards compatibility with the way InClusterConfig was originally set up
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	if ns := os.Getenv("NAMESPACE"); ns != "" {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return "default"
}
