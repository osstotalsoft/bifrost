package provider

import (
	"github.com/osstotalsoft/bifrost/servicediscovery"
	log "github.com/sirupsen/logrus"
)

type TestProvider struct {
	onRegisterHandlers   []servicediscovery.ServiceFunc
	onUnRegisterHandlers []servicediscovery.ServiceFunc
}

func NewTestProvider() *TestProvider {

	return &TestProvider{
		onRegisterHandlers:   []servicediscovery.ServiceFunc{},
		onUnRegisterHandlers: []servicediscovery.ServiceFunc{},
	}
}

func Start(provider *TestProvider) *TestProvider {

	callAllSubscribers(provider.onRegisterHandlers, servicediscovery.Service{
		Address:   "http://kube-worker1:32344",
		Resource:  "api1",
		Namespace: "gateway",
	})

	callAllSubscribers(provider.onRegisterHandlers, servicediscovery.Service{
		Address:   "http://kube-worker1:32684",
		Resource:  "api2",
		Namespace: "gateway",
	})

	callAllSubscribers(provider.onRegisterHandlers, servicediscovery.Service{
		Address:   "http://downstream-api-1.gateway/",
		Resource:  "kube-api1",
		Namespace: "gateway",
	})

	callAllSubscribers(provider.onRegisterHandlers, servicediscovery.Service{
		Address:   "http://downstream-api-2.gateway/",
		Resource:  "kube-api2",
		Namespace: "gateway",
	})

	callAllSubscribers(provider.onRegisterHandlers, servicediscovery.Service{
		Address:   "http://localhost:64307",
		Resource:  "dealers",
		Namespace: "lsng",
	})

	callAllSubscribers(provider.onRegisterHandlers, servicediscovery.Service{
		Address:   "http://localhost:64307/",
		Resource:  "countries",
		Namespace: "lsng",
	})

	callAllSubscribers(provider.onRegisterHandlers, servicediscovery.Service{
		Address:   "http://localhost:64307",
		Resource:  "lsng-api",
		Namespace: "lsng",
	})

	return provider
}

func callAllSubscribers(handlers []servicediscovery.ServiceFunc, service servicediscovery.Service) {

	log.Infof("Added new service: %v", service)

	for _, fn := range handlers {
		fn(service)
	}
}
