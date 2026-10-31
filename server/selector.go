package main

import (
	"math/rand/v2"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

const nfdPort uint32 = 6363

type nodeRouterMap struct {
	mu                   sync.RWMutex
	routerEndpointByNode map[string]routerEndpoint
}

type routerEndpoint struct {
	ip   string
	port uint32
	name string
}

func newNodeRouterMap() *nodeRouterMap {
	return &nodeRouterMap{routerEndpointByNode: make(map[string]routerEndpoint)}
}

func (r *nodeRouterMap) setRouterPods(objects []interface{}) {
	routerEndpointByNode := make(map[string]routerEndpoint)
	for _, object := range objects {
		pod, ok := object.(*corev1.Pod)
		if !ok || pod.Status.Phase != corev1.PodRunning || pod.Spec.NodeName == "" || pod.Status.PodIP == "" {
			continue
		}
		if pod.DeletionTimestamp != nil {
			continue
		}
		routerEndpointByNode[pod.Spec.NodeName] = routerEndpoint{
			ip:   pod.Status.PodIP,
			port: nfdPort,
			name: pod.Name,
		}
	}

	r.mu.Lock()
	r.routerEndpointByNode = routerEndpointByNode
	r.mu.Unlock()
}

func (r *nodeRouterMap) selectRouter(nodeName string) (routerEndpoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if local, found := r.routerEndpointByNode[nodeName]; found {
		return local, true
	}
	// Collect Router endpoints from every other Node and select one at random.
	var remote []routerEndpoint
	for _, router := range r.routerEndpointByNode {
		remote = append(remote, router)
	}
	if len(remote) == 0 {
		return routerEndpoint{}, false
	}
	return remote[rand.IntN(len(remote))], true
}
