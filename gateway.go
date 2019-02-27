package main

import (
	"bifrost/servicediscovery"
	"bifrost/utils"
	"net/http"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
)

type PreFilterFunc func(request *http.Request) error
type PostFilterFunc func(request, proxyRequest *http.Request, proxyResponse *http.Response) ([]byte, error)

type Gateway struct {
	preFilters            []PreFilterFunc
	postFilters           []PostFilterFunc
	config                *Config
	endPointToRouteMapper sync.Map
}

func NewGateway(config *Config) *Gateway {
	if config == nil {
		log.Panicf("Gateway: Must provide a configuration file")
	}
	return &Gateway{config: config}
}

type AddEndpointFunc func(addRouteFunc AddRouteFunc) func(service servicediscovery.Service)
type AddRouteFunc func(path string, pathPrefix string, methods []string, targetUrl, targetUrlPath, targetUrlPrefix string) string
type UpdateEndpointFunc func(addRouteFunc AddRouteFunc, removeRouteFunc func(routeId string)) func(oldService servicediscovery.Service, newService servicediscovery.Service)

func AddEndpoint(gate *Gateway) AddEndpointFunc {
	return func(addRouteFunc AddRouteFunc) func(service servicediscovery.Service) {
		return func(service servicediscovery.Service) {
			internalAddRoute(gate, service, addRouteFunc)
		}
	}
}

func AddPreFilter(gate *Gateway) func(f PreFilterFunc) {
	return func(f PreFilterFunc) {
		router.preFilters = append(router.preFilters, f)
	}
}

func AddPostFilter(gate *Gateway) func(f PostFilterFunc) {
	return func(f PostFilterFunc) {
		router.postFilters = append(router.postFilters, f)
	}
}

func UpdateEndpoint(gate *Gateway) UpdateEndpointFunc {
	return func(addRouteFunc AddRouteFunc, removeRouteFunc func(routeId string)) func(oldService servicediscovery.Service, newService servicediscovery.Service) {
		return func(oldService servicediscovery.Service, newService servicediscovery.Service) {

			//removing routes
			internalRemoveRoute(gate, oldService, removeRouteFunc)

			//adding routes
			internalAddRoute(gate, newService, addRouteFunc)
		}
	}
}

func internalRemoveRoute(gate *Gateway, oldService servicediscovery.Service, removeRouteFunc func(routeId string)) {
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

func runPreFilters(preFilters []PreFilterFunc, request *http.Request) error {
	for _, filter := range preFilters {
		err := filter(request)
		if err != nil {
			return err
		}
	}
	return nil
}

func internalAddRoute(gate *Gateway, service servicediscovery.Service, addRouteFunc AddRouteFunc) {
	endPoints := findEndpoints(gate.config.Endpoints, service.Resource)
	var routes []string
	for _, endp := range endPoints {
		pathPrefix := endp.DownstreamPathPrefix
		if pathPrefix == "" {
			pathPrefix = utils.SingleJoiningSlash(gate.config.DownstreamPathPrefix, service.Resource)
		}

		upstreamPathPrefix := endp.UpstreamPathPrefix
		if upstreamPathPrefix == "" {
			upstreamPathPrefix = gate.config.UpstreamPathPrefix
		}

		targetUrl := utils.SingleJoiningSlash(service.Address, utils.SingleJoiningSlash(upstreamPathPrefix, endp.UpstreamPath))
		routeId := addRouteFunc(endp.DownstreamPath, pathPrefix, endp.Methods, targetUrl, endp.UpstreamPath, upstreamPathPrefix)
		routes = append(routes, routeId)
	}

	//add default route if no config found
	if len(endPoints) == 0 {
		pathPrefix := utils.SingleJoiningSlash(gate.config.DownstreamPathPrefix, service.Resource)
		targetUrl := utils.SingleJoiningSlash(service.Address, gate.config.UpstreamPathPrefix)
		routeId := addRouteFunc("", pathPrefix, nil, targetUrl, "", gate.config.UpstreamPathPrefix)
		routes = append(routes, routeId)
		log.Infof("Gateway: Applied default configuration for service %v", service)
	}

	gate.endPointToRouteMapper.Store(service.UID, routes)
}

func RemoveEndpoint(gate *Gateway) func(removeRouteFunc func(routeId string)) func(service servicediscovery.Service) {
	return func(removeRouteFunc func(routeId string)) func(service servicediscovery.Service) {
		return func(service servicediscovery.Service) {
			internalRemoveRoute(gate, service, removeRouteFunc)
		}
	}
}

func findEndpoints(endpoints []Endpoint, serviceName string) []Endpoint {

	var result []Endpoint //endpoints[:0]

	for _, endp := range endpoints {
		if endp.ServiceName == serviceName {
			result = append(result, endp)
		}
	}

	return result
}

func handleFilterError(responseWriter http.ResponseWriter, request *http.Request, err error) {
	responseWriter.Header().Set("Content/Type", "text/html")
	responseWriter.WriteHeader(500)
	_, err = responseWriter.Write([]byte(err.Error()))
	if err != nil {
		log.Errorln(err)
	}
}

func GatewayListenAndServe(gate *Gateway, handler http.Handler) error {
	return http.ListenAndServe(":"+strconv.Itoa(gate.config.Port), func(w http.ResponseWriter, r *http.Request) {
		err := runPreFilters(router.preFilters, r)
		if err != nil {
			handleFilterError(writer, newReq, err)
			return
		}
		handler()
	})
}
