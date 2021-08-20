/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "rusi/pkg/operator/apis/subscriptions/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// SubscriptionLister helps list Subscriptions.
// All objects returned here must be treated as read-only.
type SubscriptionLister interface {
	// List lists all Subscriptions in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Subscription, err error)
	// Subscriptions returns an object that can list and get Subscriptions.
	Subscriptions(namespace string) SubscriptionNamespaceLister
	SubscriptionListerExpansion
}

// subscriptionLister implements the SubscriptionLister interface.
type subscriptionLister struct {
	indexer cache.Indexer
}

// NewSubscriptionLister returns a new SubscriptionLister.
func NewSubscriptionLister(indexer cache.Indexer) SubscriptionLister {
	return &subscriptionLister{indexer: indexer}
}

// List lists all Subscriptions in the indexer.
func (s *subscriptionLister) List(selector labels.Selector) (ret []*v1alpha1.Subscription, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Subscription))
	})
	return ret, err
}

// Subscriptions returns an object that can list and get Subscriptions.
func (s *subscriptionLister) Subscriptions(namespace string) SubscriptionNamespaceLister {
	return subscriptionNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// SubscriptionNamespaceLister helps list and get Subscriptions.
// All objects returned here must be treated as read-only.
type SubscriptionNamespaceLister interface {
	// List lists all Subscriptions in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Subscription, err error)
	// Get retrieves the Subscription from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Subscription, error)
	SubscriptionNamespaceListerExpansion
}

// subscriptionNamespaceLister implements the SubscriptionNamespaceLister
// interface.
type subscriptionNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Subscriptions in the indexer for a given namespace.
func (s subscriptionNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Subscription, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Subscription))
	})
	return ret, err
}

// Get retrieves the Subscription from the indexer for a given namespace and name.
func (s subscriptionNamespaceLister) Get(name string) (*v1alpha1.Subscription, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("subscription"), name)
	}
	return obj.(*v1alpha1.Subscription), nil
}
