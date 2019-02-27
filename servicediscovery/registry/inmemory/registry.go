package in_memory_registry

import (
	"bifrost/servicediscovery"
)

func Store(store *inMemoryRegistryData) servicediscovery.ServiceFunc {
	return func(service servicediscovery.Service) {
		store.Store(service.Address, service)
	}
}
