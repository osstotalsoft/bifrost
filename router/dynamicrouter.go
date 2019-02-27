package router

import (
	"context"
	"net/http"
	"sync"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type dynamicRouter struct {
	routes       *sync.Map
	routeMatcher RouteMatcherFunc
}

func NewDynamicRouter(routeMatcher RouteMatcherFunc) *dynamicRouter {
	return &dynamicRouter{
		new(sync.Map),
		routeMatcher,
	}
}

func GetHandler(router *dynamicRouter) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Gateway", "GoGateway")

		route, routeMatch := matchRoute(router.routes, request)
		if !routeMatch.Matched {
			http.NotFound(writer, request)
			return
		}

		ctx := context.WithValue(request.Context(), ContextRouteKey, RouteContext{
			route.Path,
			route.PathPrefix,
			route.Timeout,
			routeMatch.Vars,
		})

		newReq := request.WithContext(ctx)

		route.handler.ServeHTTP(writer, newReq)
	})
}

func AddRoute(router *dynamicRouter) func(path, pathPrefix string, methods []string, handler http.Handler) Route {
	return func(path, pathPrefix string, methods []string, handler http.Handler) Route {
		route := Route{
			Path:       path,
			PathPrefix: pathPrefix,
			Methods:    methods,
			handler:    handler,
			UID:        uuid.NewV4().String(),
		}

		route.matcher = router.routeMatcher(route)

		//check for multiple registrations
		router.routes.Range(func(key, value interface{}) bool {
			r := value.(Route)
			if r.String() == route.String() {
				log.Error("DynamicRouter: multiple registrations for : " + route.String())
				return false
			}
			return true
		})

		router.routes.Store(route.UID, route)
		log.Infof("DynamicRouter: Added new route: id: %s; pathPrefix: %s; path %s", route.UID, route.PathPrefix, route.Path)

		return route
	}
}

func RemoveRoute(router *dynamicRouter) func(routeId string) {
	return func(routeId string) {
		route, ok := router.routes.Load(routeId)
		if !ok {
			log.Error("DynamicRouter: Route does not exist " + routeId)
		}

		router.routes.Delete(routeId)
		log.Debugf("DynamicRouter: Deleted route id: %s; pathPrefix: %s; path %s", route.(Route).UID, route.(Route).PathPrefix, route.(Route).Path)
	}
}
