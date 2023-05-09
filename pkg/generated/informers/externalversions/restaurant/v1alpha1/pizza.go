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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	restaurantv1alpha1 "github.com/zeroisme/pizza-crd/pkg/apis/restaurant/v1alpha1"
	versioned "github.com/zeroisme/pizza-crd/pkg/generated/clientset/versioned"
	internalinterfaces "github.com/zeroisme/pizza-crd/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/zeroisme/pizza-crd/pkg/generated/listers/restaurant/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// PizzaInformer provides access to a shared informer and lister for
// Pizzas.
type PizzaInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.PizzaLister
}

type pizzaInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewPizzaInformer constructs a new informer for Pizza type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPizzaInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredPizzaInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredPizzaInformer constructs a new informer for Pizza type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPizzaInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.RestaurantV1alpha1().Pizzas(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.RestaurantV1alpha1().Pizzas(namespace).Watch(context.TODO(), options)
			},
		},
		&restaurantv1alpha1.Pizza{},
		resyncPeriod,
		indexers,
	)
}

func (f *pizzaInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredPizzaInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *pizzaInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&restaurantv1alpha1.Pizza{}, f.defaultInformer)
}

func (f *pizzaInformer) Lister() v1alpha1.PizzaLister {
	return v1alpha1.NewPizzaLister(f.Informer().GetIndexer())
}
