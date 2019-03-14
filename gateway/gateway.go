package gateway

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/config"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/middleware"
	"github.com/osstotalsoft/bifrost/servicediscovery"
	"github.com/osstotalsoft/bifrost/strutils"
	"net/http"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
)

//DefaultHandlerType is the default handler used when a request matches a route
const DefaultHandlerType = handler.ReverseProxyHandlerType

//Gateway is a http.Handler able to route request to different handlers
type Gateway struct {
	config                *config.Config
	endPointToRouteMapper sync.Map
	middlewares           []middlewareTuple
	handlers              map[string]handler.Func
}

type middlewareTuple struct {
	key        string
	middleware middleware.Func
}

//NewGateway is the Gateway constructor
func NewGateway(config *config.Config) *Gateway {
	if config == nil {
		log.Panicf("Gateway: Must provide a configuration file")
	}
	return &Gateway{
		config:   config,
		handlers: map[string]handler.Func{},
	}
}

//AddServiceFunc is a type for adding services using the same signature
type AddServiceFunc func(addRouteFunc AddRouteFunc) func(service servicediscovery.Service)

//AddRouteFunc is a type for adding routes using the same signature
type AddRouteFunc func(path string, pathPrefix string, methods []string, handler http.Handler) (string, error)

//UpdateEndpointFunc is a type for updating endpoints using the same signature
type UpdateEndpointFunc func(addRouteFunc AddRouteFunc, removeRouteFunc func(routeId string)) func(oldService servicediscovery.Service, newService servicediscovery.Service)

//AddService adds a service to the gateway
func AddService(gate *Gateway) AddServiceFunc {
	return func(addRouteFunc AddRouteFunc) func(service servicediscovery.Service) {
		return func(service servicediscovery.Service) {
			internalAddService(gate, service, addRouteFunc)
		}
	}
}

//UpdateService updates a service of the gateway
func UpdateService(gate *Gateway) UpdateEndpointFunc {
	return func(addRouteFunc AddRouteFunc, removeRouteFunc func(routeId string)) func(oldService servicediscovery.Service, newService servicediscovery.Service) {
		return func(oldService servicediscovery.Service, newService servicediscovery.Service) {
			//removing routes
			removeRoutes(gate, oldService, removeRouteFunc)
			//adding routes
			internalAddService(gate, newService, addRouteFunc)
		}
	}
}

//RemoveService removes a service from the gateway
func RemoveService(gate *Gateway) func(removeRouteFunc func(routeId string)) func(service servicediscovery.Service) {
	return func(removeRouteFunc func(routeId string)) func(service servicediscovery.Service) {
		return func(service servicediscovery.Service) {
			removeRoutes(gate, service, removeRouteFunc)
		}
	}
}

//UseMiddleware registers a new middleware
func UseMiddleware(gate *Gateway) func(key string, mwf middleware.Func) {
	return func(key string, mwf middleware.Func) {
		gate.middlewares = append(gate.middlewares, middlewareTuple{key, mwf})
	}
}

//RegisterHandler registers a new handler
func RegisterHandler(gate *Gateway) func(key string, handlerFunc handler.Func) {
	return func(key string, handlerFunc handler.Func) {
		gate.handlers[key] = handlerFunc
	}
}

func internalAddService(gate *Gateway, service servicediscovery.Service, addRouteFunc AddRouteFunc) []abstraction.Endpoint {
	var routes []string

	endpoints := createEndpoints(gate.config, service)
	for _, endp := range endpoints {
		routeId, _ := addRouteFunc(endp.DownstreamPath, endp.DownstreamPathPrefix, endp.Methods, getEndpointHandler(gate, endp))
		routes = append(routes, routeId)
	}
	gate.endPointToRouteMapper.Store(service.UID, routes)

	return endpoints
}

func removeRoutes(gate *Gateway, oldService servicediscovery.Service, removeRouteFunc func(routeId string)) {
	gate.endPointToRouteMapper.Range(func(key, value interface{}) bool {
		if key == oldService.UID {
			for _, rId := range value.([]string) {
				removeRouteFunc(rId)
			}
			return false
		}
		return true
	})
}

func createEndpoints(config *config.Config, service servicediscovery.Service) []abstraction.Endpoint {
	configEndpoints := findConfigEndpoints(config.Endpoints, service.Resource)
	var endPoints []abstraction.Endpoint

	for _, endp := range configEndpoints {
		var endPoint abstraction.Endpoint

		endPoint.HandlerType = endp.HandlerType
		endPoint.HandlerConfig = endp.HandlerConfig
		endPoint.Filters = endp.Filters
		if endPoint.HandlerType == "" {
			endPoint.HandlerType = DefaultHandlerType
		}

		endPoint.Secured = service.Secured
		endPoint.DownstreamPathPrefix = endp.DownstreamPathPrefix
		if endPoint.DownstreamPathPrefix == "" {
			endPoint.DownstreamPathPrefix = strutils.SingleJoiningSlash(config.DownstreamPathPrefix, service.Resource)
		}
		endPoint.UpstreamPathPrefix = endp.UpstreamPathPrefix
		if endPoint.UpstreamPathPrefix == "" {
			endPoint.UpstreamPathPrefix = config.UpstreamPathPrefix
		}

		endPoint.UpstreamURL = strutils.SingleJoiningSlash(service.Address, strutils.SingleJoiningSlash(endPoint.UpstreamPathPrefix, endp.UpstreamPath))
		endPoint.UpstreamPath = endp.UpstreamPath
		endPoint.DownstreamPath = endp.DownstreamPath
		endPoint.Methods = endp.Methods
		endPoints = append(endPoints, endPoint)
	}

	//add default route if no config found
	if len(endPoints) == 0 {
		var endPoint abstraction.Endpoint

		endPoint.Secured = service.Secured
		endPoint.HandlerType = DefaultHandlerType
		endPoint.DownstreamPathPrefix = strutils.SingleJoiningSlash(config.DownstreamPathPrefix, service.Resource)
		endPoint.UpstreamURL = strutils.SingleJoiningSlash(service.Address, config.UpstreamPathPrefix)
		endPoint.UpstreamPathPrefix = config.UpstreamPathPrefix
		endPoints = append(endPoints, endPoint)
	}

	return endPoints
}

func findConfigEndpoints(endpoints []config.EndpointConfig, serviceName string) []config.EndpointConfig {
	var result []config.EndpointConfig //endpoints[:0]
	for _, endp := range endpoints {
		if endp.ServiceName == serviceName {
			result = append(result, endp)
		}
	}
	return result
}

//ListenAndServe start the gateway server
func ListenAndServe(gate *Gateway, handler http.Handler) error {
	name := gate.config.Name

	return http.ListenAndServe(":"+strconv.Itoa(gate.config.Port), http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Gateway", name)
		handler.ServeHTTP(writer, request)
	}))
}

func getEndpointHandler(gate *Gateway, endPoint abstraction.Endpoint) http.Handler {

	handlerFunc, ok := gate.handlers[endPoint.HandlerType]
	if !ok {
		log.Fatalf("handler %s is not registered", endPoint.HandlerType)
		return nil
	}

	endpointHandler := handlerFunc(endPoint)

	for i := len(gate.middlewares) - 1; i >= 0; i-- {
		endpointHandler = gate.middlewares[i].middleware(endPoint)(endpointHandler)
	}

	return endpointHandler
}
