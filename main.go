package main

import (
	"api-gateway/filters"
	"api-gateway/handlers"
	r "api-gateway/router"
	"api-gateway/servicediscovery/provider/kubernetes"
	log "github.com/sirupsen/logrus"
)

func main() {
	config := getConfiguration()
	setLogging(config)

	provider := kubernetes.NewKubernetesServiceDiscoveryProvider(config.InCluster)
	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher)
	//registry := in_memory_registry.NewInMemoryStore()
	gateway := NewGateway(config)

	r.AddPreFilter(dynRouter)(filters.AuthorizationFilter())
	//r.AddPostFilter(dynRouter)(filters.AuthorizationFilter())

	addRouteFunc := r.AddRoute(dynRouter)
	removeRouteFunc := r.RemoveRoute(dynRouter)
	addEndpointFunc := func(path, pathPrefix string, methods []string, targetUrl, targetUrlPath, targetUrlPrefix string) string {
		revProxy := handlers.NewReverseProxy(r.GetDirector(targetUrl, targetUrlPath, targetUrlPrefix))
		rt := addRouteFunc(path, pathPrefix, methods, revProxy)
		return rt.UID
	}

	kubernetes.Compose(
		kubernetes.SubscribeOnAddService(AddEndpoint(gateway)(addEndpointFunc)),
		kubernetes.SubscribeOnRemoveService(RemoveEndpoint(gateway)(removeRouteFunc)),
		kubernetes.SubscribeOnUpdateService(UpdateEndpoint(gateway)(addEndpointFunc, removeRouteFunc)),
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
