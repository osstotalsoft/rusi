package operator

import (
	"fmt"
	"google.golang.org/grpc"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"net"
	"rusi/pkg/kube"
	compv1 "rusi/pkg/operator/apis/components/v1alpha1"
	configv1 "rusi/pkg/operator/apis/configuration/v1alpha1"
	"rusi/pkg/operator/client/clientset/versioned"
	"rusi/pkg/operator/client/informers/externalversions"
	operatorv1 "rusi/pkg/proto/operator/v1"
	"sync"
)

const serverPort = 6500

type objectWatcher struct {
	componentsMap     map[string]compv1.Component
	configurationsMap map[string]configv1.Configuration
	mu                sync.RWMutex
	compChans         []chan compv1.Component
	configChans       []chan configv1.Configuration
}

func Run() {
	cfg := kube.GetConfig()
	client, _ := versioned.NewForConfig(cfg)
	factory := externalversions.NewSharedInformerFactory(client, 0)

	ow := newObjectWatcher()
	ow.startWatchingForConfigurations(factory)
	ow.startWatchingForComponents(factory)

	stopper := make(chan struct{})
	defer close(stopper)
	factory.Start(stopper)
	s := grpc.NewServer()
	operatorv1.RegisterRusiOperatorServer(s, &operatorServer{ow})

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", serverPort))
	if err != nil {
		klog.Fatalf("error starting tcp listener: %v", err)
	}
	klog.Info("starting Operator gRPC server")
	if err := s.Serve(lis); err != nil {
		klog.Fatalf("gRPC server error: %v", err)
	}
}

func newObjectWatcher() *objectWatcher {
	return &objectWatcher{
		componentsMap:     map[string]compv1.Component{},
		configurationsMap: map[string]configv1.Configuration{},
		mu:                sync.RWMutex{},
		compChans:         []chan compv1.Component{},
		configChans:       []chan configv1.Configuration{},
	}
}

func (ow *objectWatcher) startWatchingForComponents(factory externalversions.SharedInformerFactory) {
	informer := factory.Components().V1alpha1().Components().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			comp := obj.(*compv1.Component)
			klog.V(4).InfoS("component added", "name", comp.Name, "namespace", comp.Namespace)
			ow.updateComponent(*comp)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			comp := newObj.(*compv1.Component)
			klog.V(4).InfoS("component updated", "name", comp.Name, "namespace", comp.Namespace)
			ow.updateComponent(*comp)
		},
		DeleteFunc: func(obj interface{}) {
			comp := obj.(*compv1.Component)
			klog.V(4).InfoS("component deleted", "name", comp.Name, "namespace", comp.Namespace)
		},
	})
}

func (ow *objectWatcher) updateComponent(comp compv1.Component) {
	ow.mu.Lock()
	ow.componentsMap[string(comp.UID)] = comp
	ow.mu.Unlock()
	ow.componentChange(comp)
}

func (ow *objectWatcher) componentChange(comp compv1.Component) {
	ow.mu.RLock()
	defer ow.mu.RUnlock()
	klog.V(4).InfoS("publishing component change", "subscribers", len(ow.compChans))

	for _, c := range ow.compChans {
		c := c
		go func() {
			c <- comp
		}()
	}
}

func (ow *objectWatcher) startWatchingForConfigurations(factory externalversions.SharedInformerFactory) {
	informer := factory.Configuration().V1alpha1().Configurations().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			comp := obj.(*configv1.Configuration)
			klog.V(4).InfoS("configuration added", "name", comp.Name, "namespace", comp.Namespace)
			ow.updateConfiguration(*comp)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			comp := newObj.(*configv1.Configuration)
			klog.V(4).InfoS("configuration updated", "name", comp.Name, "namespace", comp.Namespace)
			ow.updateConfiguration(*comp)
		},
		DeleteFunc: func(obj interface{}) {
			comp := obj.(*configv1.Configuration)
			klog.V(4).InfoS("configuration deleted", "name", comp.Name, "namespace", comp.Namespace)
		},
	})
}

func (ow *objectWatcher) updateConfiguration(comp configv1.Configuration) {
	ow.mu.Lock()
	ow.configurationsMap[string(comp.UID)] = comp
	ow.mu.Unlock()
	ow.configurationChange(comp)
}

func (ow *objectWatcher) configurationChange(comp configv1.Configuration) {
	ow.mu.RLock()
	defer ow.mu.RUnlock()
	klog.V(4).InfoS("publishing configuration change", "subscribers", len(ow.configChans))

	for _, c := range ow.configChans {
		c := c
		go func() {
			c <- comp
		}()
	}
}

func (ow *objectWatcher) addComponentListener(c chan compv1.Component) {
	ow.mu.Lock()
	ow.compChans = append(ow.compChans, c)
	ow.mu.Unlock()
	go ow.replayComponents(c)
}
func (ow *objectWatcher) addConfigurationListener(c chan configv1.Configuration) {
	ow.mu.Lock()
	ow.configChans = append(ow.configChans, c)
	ow.mu.Unlock()
	go ow.replayConfigs(c)
}

func (ow *objectWatcher) replayConfigs(c chan configv1.Configuration) {
	ow.mu.RLock()
	defer ow.mu.RUnlock()
	for _, item := range ow.configurationsMap {
		c <- item
	}
}

func (ow *objectWatcher) replayComponents(c chan compv1.Component) {
	ow.mu.RLock()
	defer ow.mu.RUnlock()
	for _, item := range ow.componentsMap {
		c <- item
	}
}

func (ow *objectWatcher) removeComponentListener(c chan compv1.Component) {
	ow.mu.Lock()
	defer ow.mu.Unlock()

	var list []chan compv1.Component
	for _, compChan := range ow.compChans {
		if compChan != c {
			list = append(list, compChan)
		}
	}
	ow.compChans = list
}

func (ow *objectWatcher) removeConfigurationListener(c chan configv1.Configuration) {
	ow.mu.Lock()
	defer ow.mu.Unlock()

	var list []chan configv1.Configuration
	for _, compChan := range ow.configChans {
		if compChan != c {
			list = append(list, compChan)
		}
	}
	ow.configChans = list
}
