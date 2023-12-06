package runtime

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"reflect"
	"rusi/pkg/custom-resource/components"
	components_loader "rusi/pkg/custom-resource/components/loader"
	components_middleware "rusi/pkg/custom-resource/components/middleware"
	"rusi/pkg/custom-resource/components/pubsub"
	"rusi/pkg/custom-resource/configuration"
	"rusi/pkg/healthcheck"
	"rusi/pkg/messaging"
	"strings"
	"sync"
)

type ComponentsManager struct {
	appId           string
	pubSubInstances map[string]messaging.PubSub
	components      map[string]components.Spec
	mux             *sync.RWMutex

	componentUpdatesChan   <-chan components.Spec
	changeNotificationChan chan components.ChangeNotification

	pubSubRegistry     pubsub.Registry
	middlewareRegistry components_middleware.Registry
}

func NewComponentsManager(ctx context.Context, appId string,
	componentsLoader components_loader.ComponentsLoader,
	opts ...Option) (*ComponentsManager, error) {

	var runtimeOpts runtimeOpts
	for _, opt := range opts {
		opt(&runtimeOpts)
	}

	compChan, err := componentsLoader(ctx)
	klog.V(4).InfoS("Components channel created")

	if err != nil {
		klog.ErrorS(err, "error loading components")
		return nil, err
	}

	manager := &ComponentsManager{
		appId:                  appId,
		mux:                    &sync.RWMutex{},
		componentUpdatesChan:   compChan,
		changeNotificationChan: make(chan components.ChangeNotification),
		components:             map[string]components.Spec{},
		pubSubInstances:        map[string]messaging.PubSub{},
		pubSubRegistry:         pubsub.NewRegistry(),
		middlewareRegistry:     components_middleware.NewRegistry(),
	}

	manager.pubSubRegistry.Register(runtimeOpts.pubsubs...)
	manager.middlewareRegistry.Register(runtimeOpts.pubsubMiddleware...)
	klog.V(4).InfoS("Components added to registry")

	go manager.watchComponentsUpdates()

	return manager, nil
}

func (m *ComponentsManager) watchComponentsUpdates() {
	for update := range m.componentUpdatesChan {
		err := m.addOrUpdateComponent(update)
		if err != nil {
			klog.ErrorS(err, "Error loading ", "component", update.Name)
		}
	}
	close(m.changeNotificationChan)
}

func (m *ComponentsManager) addOrUpdateComponent(spec components.Spec) (err error) {
	operation := components.Insert
	msg := "adding component"
	oldComp, exists := m.getComponent(spec.Type, spec.Name)
	if exists && reflect.DeepEqual(oldComp.Metadata, spec.Metadata) {
		return
	}
	if exists {
		operation = components.Update
		msg = "updating component"
	}

	klog.V(4).InfoS(msg, "name", spec.Name, "type", spec.Type, "version", spec.Version)

	category := extractComponentCategory(spec)
	switch {
	case category == components.BindingsComponent:
	case category == components.MiddlewareComponent:
	case category == components.SecretStoreComponent:
		//fallthrough
	case category == components.PubsubComponent && operation == components.Update:
		err = m.updatePubSub(spec)
	case category == components.PubsubComponent && operation == components.Insert:
		err = m.initPubSub(spec)
	default:
		err = errors.Errorf("invalid category %s or operation %s for this component %s",
			category, spec.Name, operation)
		return
	}

	m.storeComponent(spec)
	if err == nil {
		m.publishChange(category, operation, spec)
	}
	return
}

func (m *ComponentsManager) updatePubSub(spec components.Spec) (err error) {
	m.mux.RLock()
	ps, ok := m.pubSubInstances[spec.Name]
	m.mux.RUnlock()
	if ok {
		err = ps.Close()
	}
	if err != nil {
		return
	}
	return m.initPubSub(spec)
}
func (m *ComponentsManager) initPubSub(spec components.Spec) error {
	pubSub, err := m.pubSubRegistry.Create(spec.Type, spec.Version)
	if err != nil {
		klog.Warningf("error creating pub sub %s (%s/%s): %s", spec.Name, spec.Type, spec.Version, err)
		return err
	}

	//clone the metadata
	metadata := make(map[string]string)
	for index, element := range spec.Metadata {
		metadata[index] = element
	}

	metadata["consumerID"] = m.appId
	err = pubSub.Init(metadata)
	if err != nil {
		klog.Warningf("error initializing pub sub %s/%s: %s", spec.Type, spec.Version, err)
		return err
	}

	m.mux.Lock()
	defer m.mux.Unlock()
	m.pubSubInstances[spec.Name] = pubSub

	return nil
}

func (m *ComponentsManager) getComponent(componentType string, name string) (components.Spec, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	for _, c := range m.components {
		if c.Type == componentType && c.Name == name {
			return c, true
		}
	}
	return components.Spec{}, false
}

func (m *ComponentsManager) GetMiddleware(middlewareSpec configuration.HandlerSpec) (messaging.Middleware, error) {
	component, exists := m.getComponent(middlewareSpec.Type, middlewareSpec.Name)
	if !exists {
		return nil,
			errors.Errorf("couldn't find middleware component with name %s and type %s/%s",
				middlewareSpec.Name,
				middlewareSpec.Type,
				middlewareSpec.Version)
	}
	midlw, err := m.middlewareRegistry.Create(middlewareSpec.Type, middlewareSpec.Version,
		component.Metadata)

	return midlw, err
}
func (m *ComponentsManager) GetPublisher(pubsubName string) messaging.Publisher {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.pubSubInstances[pubsubName]
}
func (m *ComponentsManager) GetSubscriber(pubsubName string) messaging.Subscriber {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.pubSubInstances[pubsubName]
}
func (m *ComponentsManager) Watch() <-chan components.ChangeNotification {
	return m.changeNotificationChan
}
func (m *ComponentsManager) storeComponent(spec components.Spec) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.components[getComponentKey(spec.Type, spec.Name)] = spec
}
func (m *ComponentsManager) publishChange(category components.ComponentCategory,
	operation components.Operation, spec components.Spec) {

	m.changeNotificationChan <- components.ChangeNotification{
		ComponentCategory: category,
		ComponentSpec:     spec,
		Operation:         operation,
	}
}

func extractComponentCategory(spec components.Spec) components.ComponentCategory {
	for _, category := range components.ComponentCategories {
		if strings.HasPrefix(spec.Type, fmt.Sprintf("%s.", category)) {
			return category
		}
	}
	return ""
}

func getComponentKey(ctype, name string) string {
	return fmt.Sprintf("%s-%s", ctype, name)
}

func (m *ComponentsManager) IsHealthy(ctx context.Context) healthcheck.HealthResult {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for _, ps := range m.pubSubInstances {
		if hc, ok := ps.(healthcheck.HealthChecker); ok {
			if r := hc.IsHealthy(ctx); r.Status != healthcheck.Healthy {
				return r
			}
		}
	}

	if len(m.pubSubInstances) == 0 {
		return healthcheck.HealthResult{
			Status:      healthcheck.Unhealthy,
			Description: "no pubsub instance created yet",
		}
	}

	return healthcheck.HealthyResult
}
