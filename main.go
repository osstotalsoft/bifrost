package main

import (
	"github.com/osstotalsoft/bifrost/config"
	"github.com/osstotalsoft/bifrost/gateway"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/handler/nats"
	"github.com/osstotalsoft/bifrost/handler/reverseproxy"
	"github.com/osstotalsoft/bifrost/middleware/auth"
	r "github.com/osstotalsoft/bifrost/router"
	"github.com/osstotalsoft/bifrost/servicediscovery/provider/kubernetes"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
)

func main() {

	//https://github.com/golang/go/issues/16012
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	cfg := getConfig()
	setLogging(cfg.LogLevel)

	provider := kubernetes.NewKubernetesServiceDiscoveryProvider(cfg.InCluster, cfg.OverrideServiceAddress)
	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher)
	//registry := in_memory_registry.NewInMemoryStore()
	gate := gateway.NewGateway(cfg)
	registerHandlerFunc := gateway.RegisterHandler(gate)

	gateway.UseMiddleware(gate)(auth.AuthorizationFilterCode, auth.AuthorizationFilter(getIdentityServerConfig()))
	registerHandlerFunc(handler.EventPublisherHandlerType, nats.NewNatsPublisher(getNatsHandlerConfig()))
	registerHandlerFunc(handler.ReverseProxyHandlerType, reverseproxy.NewReverseProxy())

	addRouteFunc := r.AddRoute(dynRouter)
	removeRouteFunc := r.RemoveRoute(dynRouter)

	//configure and start ServiceDiscovery
	kubernetes.Compose(
		kubernetes.SubscribeOnAddService(gateway.AddService(gate)(addRouteFunc)),
		kubernetes.SubscribeOnRemoveService(gateway.RemoveService(gate)(removeRouteFunc)),
		kubernetes.SubscribeOnUpdateService(gateway.UpdateService(gate)(addRouteFunc, removeRouteFunc)),
		kubernetes.Start,
	)(provider)

	err := gateway.ListenAndServe(gate, r.GetHandler(dynRouter))
	if err != nil {
		log.Print(err)
	}
}

func setLogging(logLevel string) {
	log.SetFormatter(&log.JSONFormatter{})
	//log.SetReportCaller(true)
	level, e := log.ParseLevel(logLevel)
	if e == nil {
		log.SetLevel(level)
	}
}

func getConfig() *config.Config {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("json")
	viper.AutomaticEnv()

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Panicf("Fatal error config file: %s", err)
	}

	var cfg = new(config.Config)
	err = viper.Unmarshal(cfg)
	if err != nil {
		log.Panicf("unable to decode into struct, %v", err)
	}
	log.Infof("Using configuration : %v", viper.AllSettings())

	return cfg
}

func getNatsHandlerConfig() nats.Config {
	var cfg = new(nats.Config)
	err := viper.UnmarshalKey("handlers.event.nats", cfg)
	if err != nil {
		log.Panicf("unable to decode into NatsConfig, %v", err)
	}

	return *cfg
}

func getIdentityServerConfig() auth.AuthorizationOptions {
	var cfg = new(auth.AuthorizationOptions)
	err := viper.UnmarshalKey("filters.auth", cfg)
	if err != nil {
		log.Panicf("unable to decode into AuthorizationOptions, %v", err)
	}

	return *cfg
}
