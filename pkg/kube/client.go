package kube

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"path/filepath"
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
