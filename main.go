package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/osstotalsoft/bifrost/config"
	"github.com/osstotalsoft/bifrost/gateway"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/handler/nats"
	"github.com/osstotalsoft/bifrost/handler/reverseproxy"
	"github.com/osstotalsoft/bifrost/httputils"
	"github.com/osstotalsoft/bifrost/middleware"
	"github.com/osstotalsoft/bifrost/middleware/auth"
	"github.com/osstotalsoft/bifrost/middleware/cors"
	r "github.com/osstotalsoft/bifrost/router"
	"github.com/osstotalsoft/bifrost/servicediscovery/provider/kubernetes"
	"github.com/osstotalsoft/bifrost/tracing"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	jaegerprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"net/http"
)

func main() {
	//var signalsChannel = make(chan os.Signal, 1)
	//signal.Notify(signalsChannel, os.Interrupt, syscall.SIGTERM)

	//https://github.com/golang/go/issues/16012
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	level, logger, _ := getZapLogger()
	defer logger.Sync()

	cfg := getConfig(logger)
	changeLogLevel(level, cfg.LogLevel)

	loggerFactory := tracing.SpanLoggerFactory(logger.With(zap.String("service", "api gateway")))

	_, closer := setupJaeger(logger)
	defer closer.Close()

	provider := kubernetes.NewKubernetesServiceDiscoveryProvider(cfg.InCluster, cfg.OverrideServiceAddress, loggerFactory)
	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher, loggerFactory)
	//registry := in_memory_registry.NewInMemoryStore()

	natsHandler, closeNatsConnection := nats.NewNatsPublisher(getNatsHandlerConfig(logger), nats.TransformMessage, nats.BuildResponse, loggerFactory)
	defer closeNatsConnection()

	gate := gateway.NewGateway(cfg, loggerFactory)
	registerHandlerFunc := gateway.RegisterHandler(gate)
	gateMiddlewareFunc := gateway.UseMiddleware(gate)

	gateMiddlewareFunc(cors.CORSFilterCode, middleware.Compose(
		tracing.MiddlewareStartSpan("CORS Filter"),
	)(cors.CORSFilter("*")))

	gateMiddlewareFunc(auth.AuthorizationFilterCode, middleware.Compose(
		tracing.MiddlewareStartSpan("Authorization Filter"),
	)(auth.AuthorizationFilter(getIdentityServerConfig(logger), loggerFactory)))

	registerHandlerFunc(handler.EventPublisherHandlerType, handler.Compose(tracing.HandlerStartSpan("Nats Handler"))(natsHandler))
	registerHandlerFunc(handler.ReverseProxyHandlerType, handler.Compose(
		tracing.HandlerStartSpan("Reverse Proxy Handler"),
	)(reverseproxy.NewReverseProxy(tracing.NewRoundTripperWithOpenTrancing(), loggerFactory)))

	addRouteFunc := r.AddRoute(dynRouter)
	removeRouteFunc := r.RemoveRoute(dynRouter)

	//configure and start ServiceDiscovery
	kubernetes.Compose(
		kubernetes.SubscribeOnAddService(gateway.AddService(gate)(addRouteFunc)),
		kubernetes.SubscribeOnRemoveService(gateway.RemoveService(gate)(removeRouteFunc)),
		kubernetes.SubscribeOnUpdateService(gateway.UpdateService(gate)(addRouteFunc, removeRouteFunc)),
		kubernetes.Start,
	)(provider)

	err := gateway.ListenAndServe(gate, httputils.Compose(
		httputils.RecoveryHandler(loggerFactory),
		tracing.StartSpan,
	)(r.GetHandler(dynRouter)))

	if err != nil {
		logger.Error("gateway cannot start", zap.Error(err))
	}

	//log.Info("Shutting down")
	//kubernetes.Stop(provider)
	//closeNatsConnection()
}

func getZapLogger() (zap.AtomicLevel, *zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "json"
	cfg.DisableCaller = true
	l, e := cfg.Build()
	return cfg.Level, l, e
}

func changeLogLevel(oldLevel zap.AtomicLevel, newLevel string) {
	level := zapcore.InfoLevel
	e := level.Set(newLevel)
	if e == nil {
		oldLevel.SetLevel(level)
	}
	return
}

func getConfig(logger *zap.Logger) *config.Config {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("json")
	viper.AutomaticEnv()

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		logger.Panic("unable to read configuration file", zap.Error(err))
	}

	var cfg = new(config.Config)
	err = viper.Unmarshal(cfg)
	if err != nil {
		logger.Panic("unable to decode into struct", zap.Error(err))
	}
	logger.Info(fmt.Sprintf("using configuration: %v", viper.AllSettings()))

	return cfg
}

func getNatsHandlerConfig(logger *zap.Logger) nats.Config {
	var cfg = new(nats.Config)
	err := viper.UnmarshalKey("handlers.event.nats", cfg)
	if err != nil {
		logger.Panic("unable to decode into NatsConfig", zap.Error(err))
	}

	return *cfg
}

func getIdentityServerConfig(logger *zap.Logger) auth.AuthorizationOptions {
	var cfg = new(auth.AuthorizationOptions)
	err := viper.UnmarshalKey("filters.auth", cfg)
	if err != nil {
		logger.Panic("unable to decode into AuthorizationOptions", zap.Error(err))
	}

	return *cfg
}

func setupJaeger(logger *zap.Logger) (opentracing.Tracer, io.Closer) {
	var cfg = &struct {
		Enabled bool   `json:"enabled"`
		Agent   string `json:"agent"`
	}{}

	err := viper.UnmarshalKey("opentracing", cfg)
	if err != nil {
		logger.Panic("unable to decode into Jaeger Config", zap.Error(err))
	}

	jconfig := jaegercfg.Configuration{
		Disabled:    !cfg.Enabled,
		ServiceName: "Bifrost API Gateway",
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           false,
			LocalAgentHostPort: cfg.Agent,
		},
	}

	jLogger := jaegerlog.StdLogger
	jMetricsFactory := jaegerprom.New() //metrics.NullFactory
	//jaeger.NewMetrics(factory, map[string]string{"lib": "jaeger"})

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, _ := jconfig.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)

	return tracer, closer
}
