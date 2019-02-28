package main

import (
	"github.com/osstotalsoft/bifrost/config"
	"github.com/osstotalsoft/bifrost/filters"
	"github.com/osstotalsoft/bifrost/gateway"
	r "github.com/osstotalsoft/bifrost/router"
	"github.com/osstotalsoft/bifrost/servicediscovery/providers/kubernetes"
	log "github.com/sirupsen/logrus"
)

func main() {
	config := getConfiguration()
	setLogging(config)

	provider := kubernetes.NewKubernetesServiceDiscoveryProvider(config.InCluster)
	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher)
	//registry := in_memory_registry.NewInMemoryStore()
	gate := gateway.NewGateway(config)

	gateway.AddPreFilter(gate)(filters.AuthorizationFilter())
	//r.AddPostFilter(dynRouter)(filters.AuthorizationFilter())

	addRouteFunc := r.AddRoute(dynRouter)
	removeRouteFunc := r.RemoveRoute(dynRouter)

	kubernetes.Compose(
		kubernetes.SubscribeOnAddService(gateway.AddService(gate)(addRouteFunc)),
		kubernetes.SubscribeOnRemoveService(gateway.RemoveService(gate)(removeRouteFunc)),
		kubernetes.SubscribeOnUpdateService(gateway.UpdateService(gate)(addRouteFunc, removeRouteFunc)),
		kubernetes.Start,
	)(provider)

	err := gateway.GatewayListenAndServe(gate, r.GetHandler(dynRouter))
	if err != nil {
		log.Print(err)
	}
}

func setLogging(config *config.Config) {
	log.SetFormatter(&log.JSONFormatter{})
	//log.SetReportCaller(true)
	level, e := log.ParseLevel(config.LogLevel)
	if e == nil {
		log.SetLevel(level)
	}
}

func getConfiguration() *config.Config {
	return config.LoadConfig()
}
