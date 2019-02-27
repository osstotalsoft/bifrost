package router

import (
	"sync"
)

type routeMap struct {
	m        sync.RWMutex
	internal map[string]Route
}

func NewRouteMap() *routeMap {
	return &routeMap{
		internal: make(map[string]Route),
	}
}

func (rm *routeMap) Load(key string) (value Route, ok bool) {
	rm.m.RLock()
	defer rm.m.RUnlock()
	result, ok := rm.internal[key]
	return result, ok
}

func (rm *routeMap) GetAll() []Route {
	rm.m.RLock()
	defer rm.m.RUnlock()

	var result []Route
	for _, value := range rm.internal {
		result = append(result, value)
	}
	return result
}

func (rm *routeMap) Delete(key string) {
	rm.m.Lock()
	defer rm.m.Unlock()

	delete(rm.internal, key)
}

func (rm *routeMap) Store(key string, value Route) {
	rm.m.Lock()
	defer rm.m.Unlock()

	rm.internal[key] = value
}
