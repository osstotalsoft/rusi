package operator

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
	"net"
	"rusi/pkg/kube"
	compv1 "rusi/pkg/operator/apis/components/v1alpha1"
	configv1 "rusi/pkg/operator/apis/configuration/v1alpha1"
	"rusi/pkg/operator/client/clientset/versioned"
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
	s := grpc.NewServer()

	cfg := kube.GetConfig()
	client, _ := versioned.NewForConfig(cfg)
	ow := newObjectWatcher()

	go ow.startWatchingForConfigurations(context.Background(), client)
	go ow.startWatchingForComponents(context.Background(), client)

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

func (ow *objectWatcher) startWatchingForComponents(ctx context.Context, client *versioned.Clientset) {
	watcher, err := client.ComponentsV1alpha1().Components("").Watch(ctx, v1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "error creating components watcher")
		return
	}
	for event := range watcher.ResultChan() {
		klog.V(4).InfoS("received component", "event", event.Type, "object", event.Object)
		if event.Type != watch.Error {
			comp := event.Object.(*compv1.Component)
			ow.mu.Lock()
			ow.componentsMap[string(comp.UID)] = *comp
			ow.mu.Unlock()
			ow.componentChange(*comp)
		}
	}
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

func (ow *objectWatcher) startWatchingForConfigurations(ctx context.Context, client *versioned.Clientset) {
	watcher, err := client.ConfigurationV1alpha1().Configurations("").Watch(ctx, v1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "error creating configurations watcher")
		return
	}
	for event := range watcher.ResultChan() {
		klog.V(4).InfoS("received configuration", "event", event.Type, "object", event.Object)
		if event.Type != watch.Error {
			comp := event.Object.(*configv1.Configuration)
			ow.mu.Lock()
			ow.configurationsMap[string(comp.UID)] = *comp
			ow.mu.Unlock()
			ow.configurationChange(*comp)
		}
	}
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
