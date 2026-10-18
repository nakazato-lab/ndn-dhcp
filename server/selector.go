package main

import (
	"math/rand/v2"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

type nodeRouterMap struct {
	mu              sync.RWMutex
	routerIPsByNode map[string][]string
}

func newNodeRouterMap() *nodeRouterMap {
	return &nodeRouterMap{routerIPsByNode: make(map[string][]string)}
}

func (r *nodeRouterMap) setRouterPods(objects []interface{}) {
	routerIPsByNode := make(map[string][]string)
	for _, object := range objects {
		pod, ok := object.(*corev1.Pod)
		if !ok || pod.Status.Phase != corev1.PodRunning || pod.Spec.NodeName == "" || pod.Status.PodIP == "" {
			continue
		}
		if pod.DeletionTimestamp != nil {
			continue
		}
		routerIPsByNode[pod.Spec.NodeName] = append(routerIPsByNode[pod.Spec.NodeName], pod.Status.PodIP)
	}

	r.mu.Lock()
	r.routerIPsByNode = routerIPsByNode
	r.mu.Unlock()
}

func (r *nodeRouterMap) selectRouter(nodeName string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if local := r.routerIPsByNode[nodeName]; len(local) > 0 {
		return local[0], true
	}
	// Collect Router IPs from every other Node and select one at random.
	var remote []string
	for _, routers := range r.routerIPsByNode {
		remote = append(remote, routers...)
	}
	if len(remote) == 0 {
		return "", false
	}
	return remote[rand.IntN(len(remote))], true
}
