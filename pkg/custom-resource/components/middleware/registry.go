package middleware

import (
	"fmt"
	"rusi/pkg/custom-resource/components"
	"rusi/pkg/messaging"
	"strings"
)

type (
	// Middleware is a Pubsub middleware component definition.
	Middleware struct {
		Name          string
		FactoryMethod func(properties map[string]string) messaging.Middleware
	}

	// Registry is the interface for callers to get registered Pubsub middleware.
	Registry interface {
		Register(components ...Middleware)
		Create(name, version string, properties map[string]string) (messaging.Middleware, error)
	}

	pubsubMiddlewareRegistry struct {
		middleware map[string]func(properties map[string]string) messaging.Middleware
	}
)

// New creates a Middleware.
func New(name string, factoryMethod func(properties map[string]string) messaging.Middleware) Middleware {
	return Middleware{
		Name:          name,
		FactoryMethod: factoryMethod,
	}
}

// NewRegistry returns a new Pubsub middleware registry.
func NewRegistry() Registry {
	return &pubsubMiddlewareRegistry{
		middleware: map[string]func(properties map[string]string) messaging.Middleware{},
	}
}

// Register registers one or more new Pubsub middlewares.
func (p *pubsubMiddlewareRegistry) Register(components ...Middleware) {
	for _, component := range components {
		p.middleware[createFullName(component.Name)] = component.FactoryMethod
	}
}

// Create instantiates a Pubsub middleware based on `name`.
func (p *pubsubMiddlewareRegistry) Create(name, version string, properties map[string]string) (messaging.Middleware, error) {
	if method, ok := p.getMiddleware(name, version); ok {
		return method(properties), nil
	}
	return nil, fmt.Errorf("Pubsub middleware %s/%s has not been registered", name, version)
}

func (p *pubsubMiddlewareRegistry) getMiddleware(name, version string) (func(properties map[string]string) messaging.Middleware, bool) {
	nameLower := strings.ToLower(name)
	versionLower := strings.ToLower(version)
	middlewareFn, ok := p.middleware[nameLower+"/"+versionLower]
	if ok {
		return middlewareFn, true
	}
	if components.IsInitialVersion(versionLower) {
		middlewareFn, ok = p.middleware[nameLower]
	}
	return middlewareFn, ok
}

func createFullName(name string) string {
	return strings.ToLower("middleware.pubsub." + name)
}
