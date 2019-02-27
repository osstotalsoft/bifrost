package in_memory_registry

import (
	"api-gateway/servicediscovery"
	"sync"
)

type inMemoryRegistryData struct {
	m        sync.RWMutex
	internal map[string]servicediscovery.Service
}

func NewInMemoryStore() *inMemoryRegistryData {
	return &inMemoryRegistryData{
		internal: make(map[string]servicediscovery.Service),
	}
}

func (rm *inMemoryRegistryData) Load(key string) (value servicediscovery.Service, ok bool) {
	rm.m.RLock()
	result, ok := rm.internal[key]
	rm.m.RUnlock()
	return result, ok
}

func (rm *inMemoryRegistryData) GetAll() []servicediscovery.Service {
	rm.m.RLock()
	var result []servicediscovery.Service
	for _, value := range rm.internal {
		result = append(result, value)
	}
	rm.m.RUnlock()
	return result
}

func (rm *inMemoryRegistryData) Delete(key string) {
	rm.m.Lock()
	delete(rm.internal, key)
	rm.m.Unlock()
}

func (rm *inMemoryRegistryData) Store(key string, value servicediscovery.Service) {
	rm.m.Lock()
	rm.internal[key] = value
	rm.m.Unlock()
}
