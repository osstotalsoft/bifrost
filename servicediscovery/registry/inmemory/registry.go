package inmemory

import (
	"github.com/osstotalsoft/bifrost/servicediscovery"
)

func Store(store *inMemoryRegistryData) servicediscovery.ServiceFunc {
	return func(service servicediscovery.Service) {
		store.Store(service.Address, service)
	}
}
