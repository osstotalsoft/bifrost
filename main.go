package main

import (
	"bifrost/filters"
	r "bifrost/router"
	"bifrost/servicediscovery/provider/kubernetes"
	log "github.com/sirupsen/logrus"
)

func main() {
	config := getConfiguration()
	setLogging(config)

	provider := kubernetes.NewKubernetesServiceDiscoveryProvider(config.InCluster)
	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher)
	//registry := in_memory_registry.NewInMemoryStore()
	gateway := NewGateway(config)

	AddPreFilter(gateway)(filters.AuthorizationFilter())
	//r.AddPostFilter(dynRouter)(filters.AuthorizationFilter())

	addRouteFunc := r.AddRoute(dynRouter)
	removeRouteFunc := r.RemoveRoute(dynRouter)

	kubernetes.Compose(
		kubernetes.SubscribeOnAddService(AddEndpoint(gateway)(addRouteFunc)),
		kubernetes.SubscribeOnRemoveService(RemoveEndpoint(gateway)(removeRouteFunc)),
		kubernetes.SubscribeOnUpdateService(UpdateEndpoint(gateway)(addRouteFunc, removeRouteFunc)),
		kubernetes.Start,
	)(provider)

	err := GatewayListenAndServe(gateway, r.GetHandler(dynRouter))
	if err != nil {
		log.Print(err)
	}
}

func setLogging(config *Config) {
	log.SetFormatter(&log.JSONFormatter{})
	//log.SetReportCaller(true)
	level, e := log.ParseLevel(config.LogLevel)
	if e == nil {
		log.SetLevel(level)
	}
}

func getConfiguration() *Config {
	return LoadConfig()
}
