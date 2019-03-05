package gateway

import (
	"github.com/osstotalsoft/bifrost/config"
	"github.com/osstotalsoft/bifrost/servicediscovery"
	"github.com/osstotalsoft/bifrost/strutils"
	"net/http"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
)

const DefaultHandlerType = "reverseproxy"

type MiddlewareFunc func(endpoint Endpoint) func(http.Handler) http.Handler
type HandlerFunc func(endpoint Endpoint) http.Handler

type Gateway struct {
	config                *config.Config
	endPointToRouteMapper sync.Map
	middlewares           []middlewareTuple
	handlers              map[string]HandlerFunc
}
type Endpoint struct {
	UpstreamPath         string
	Secured              bool
	UpstreamPathPrefix   string
	UpstreamURL          string
	DownstreamPath       string
	DownstreamPathPrefix string
	Methods              []string
	HandlerType          string
	HandlerConfig        map[string]interface{}
	Filters              map[string]interface{}
}

type middlewareTuple struct {
	key        string
	middleware MiddlewareFunc
}

func NewGateway(config *config.Config) *Gateway {
	if config == nil {
		log.Panicf("Gateway: Must provide a configuration file")
	}
	return &Gateway{
		config:   config,
		handlers: map[string]HandlerFunc{},
	}
}

type AddServiceFunc func(addRouteFunc AddRouteFunc) func(service servicediscovery.Service)
type AddRouteFunc func(path string, pathPrefix string, methods []string, handler http.Handler) (string, error)
type UpdateEndpointFunc func(addRouteFunc AddRouteFunc, removeRouteFunc func(routeId string)) func(oldService servicediscovery.Service, newService servicediscovery.Service)

func AddService(gate *Gateway) AddServiceFunc {
	return func(addRouteFunc AddRouteFunc) func(service servicediscovery.Service) {
		return func(service servicediscovery.Service) {
			internalAddService(gate, service, addRouteFunc)
		}
	}
}

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

func UseMiddleware(gate *Gateway) func(key string, mwf MiddlewareFunc) {
	return func(key string, mwf MiddlewareFunc) {
		gate.middlewares = append(gate.middlewares, middlewareTuple{key, mwf})
	}
}

func RegisterHandler(gate *Gateway) func(key string, handlerFunc HandlerFunc) {
	return func(key string, handlerFunc HandlerFunc) {
		gate.handlers[key] = handlerFunc
	}
}

func RemoveService(gate *Gateway) func(removeRouteFunc func(routeId string)) func(service servicediscovery.Service) {
	return func(removeRouteFunc func(routeId string)) func(service servicediscovery.Service) {
		return func(service servicediscovery.Service) {
			removeRoutes(gate, service, removeRouteFunc)
		}
	}
}

func internalAddService(gate *Gateway, service servicediscovery.Service, addRouteFunc AddRouteFunc) []Endpoint {
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

func createEndpoints(config *config.Config, service servicediscovery.Service) []Endpoint {
	configEndpoints := findConfigEndpoints(config.Endpoints, service.Resource)
	var endPoints []Endpoint

	for _, endp := range configEndpoints {
		var endPoint Endpoint

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
		var endPoint Endpoint

		endPoint.Secured = service.Secured
		endPoint.HandlerType = DefaultHandlerType
		endPoint.DownstreamPathPrefix = strutils.SingleJoiningSlash(config.DownstreamPathPrefix, service.Resource)
		endPoint.UpstreamURL = strutils.SingleJoiningSlash(service.Address, config.UpstreamPathPrefix)
		endPoint.UpstreamPathPrefix = config.UpstreamPathPrefix
		endPoints = append(endPoints, endPoint)
	}

	return endPoints
}

func findConfigEndpoints(endpoints []config.Endpoint, serviceName string) []config.Endpoint {
	var result []config.Endpoint //endpoints[:0]
	for _, endp := range endpoints {
		if endp.ServiceName == serviceName {
			result = append(result, endp)
		}
	}
	return result
}

func ListenAndServe(gate *Gateway, handler http.Handler) error {
	return http.ListenAndServe(":"+strconv.Itoa(gate.config.Port), handler)
}

func getEndpointHandler(gate *Gateway, endPoint Endpoint) http.Handler {

	//TODO: validation
	handlerFunc := gate.handlers[endPoint.HandlerType]
	handler := handlerFunc(endPoint)
	name := gate.config.Name

	var h http.Handler
	h = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Gateway", name)
		handler.ServeHTTP(writer, request)
	})

	for i := len(gate.middlewares) - 1; i >= 0; i-- {
		h = gate.middlewares[i].middleware(endPoint)(h)
	}

	return h
}