package main

import (
	"github.com/opentracing/opentracing-go"
	"github.com/osstotalsoft/bifrost/config"
	"github.com/osstotalsoft/bifrost/gateway"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/handler/nats"
	"github.com/osstotalsoft/bifrost/handler/reverseproxy"
	"github.com/osstotalsoft/bifrost/middleware/auth"
	"github.com/osstotalsoft/bifrost/middleware/cors"
	r "github.com/osstotalsoft/bifrost/router"
	"github.com/osstotalsoft/bifrost/servicediscovery/provider/kubernetes"
	"github.com/osstotalsoft/bifrost/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
	"io"
	"net/http"
)

func main() {
	//var signalsChannel = make(chan os.Signal, 1)
	//signal.Notify(signalsChannel, os.Interrupt, syscall.SIGTERM)

	//https://github.com/golang/go/issues/16012
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	cfg := getConfig()
	setLogging(cfg.LogLevel)

	_, closer := setupJaeger()
	defer closer.Close()

	provider := kubernetes.NewKubernetesServiceDiscoveryProvider(cfg.InCluster, cfg.OverrideServiceAddress)
	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher)
	//registry := in_memory_registry.NewInMemoryStore()
	gate := gateway.NewGateway(cfg)
	registerHandlerFunc := gateway.RegisterHandler(gate)

	//gateway.AddPreFilter(gate)(filters.AuthorizationFilter())
	natsHandler, closeNatsConnection := nats.NewNatsPublisher(getNatsHandlerConfig(), nats.TransformMessage, nats.BuildResponse)
	defer closeNatsConnection()

	gateMiddlewareFunc := gateway.UseMiddleware(gate)
	gateMiddlewareFunc(cors.CORSFilterCode, tracing.WrapMiddleware(cors.CORSFilter("*"), "CORSFilter"))
	gateMiddlewareFunc(auth.AuthorizationFilterCode, tracing.WrapMiddleware(auth.AuthorizationFilter(getIdentityServerConfig()), "AuthorizationFilter"))

	registerHandlerFunc(handler.EventPublisherHandlerType, natsHandler)
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

	err := gateway.ListenAndServe(gate, tracing.Wrap(r.GetHandler(dynRouter)))
	if err != nil {
		log.Print(err)
	}

	//log.Info("Shutting down")
	//kubernetes.Stop(provider)
	//closeNatsConnection()
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

func setupJaeger() (opentracing.Tracer, io.Closer) {

	cfg := jaegercfg.Configuration{
		ServiceName: "Bifrost API Gateway",
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "kube-worker1:31457",
		},
	}

	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, _ := cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)

	return tracer, closer
}
