package main

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func startRouterWatcher(ctx context.Context, namespace string) (*nodeRouterMap, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	kube, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	routerMap := newNodeRouterMap()
	factory := informers.NewSharedInformerFactoryWithOptions(kube, 0,
		informers.WithNamespace(namespace),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = "app=nfd"
		}),
	)
	pods := factory.Core().V1().Pods().Informer()
	update := func() { routerMap.setRouterPods(pods.GetStore().List()) }
	pods.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(interface{}) { update() },
		UpdateFunc: func(interface{}, interface{}) { update() },
		DeleteFunc: func(interface{}) { update() },
	})

	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), pods.HasSynced) {
		return nil, context.Canceled
	}
	update()
	return routerMap, nil
}
