package router

import (
	"context"
	"errors"
	"fmt"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/satori/go.uuid"
	"go.uber.org/zap"
	"net/http"
	"sync"
)

type dynamicRouter struct {
	routes       *sync.Map
	routeMatcher RouteMatcherFunc
	logger       log.Logger
}

//NewDynamicRouter creates a new dynamic router
//Its dynamic because it can add/remove routes at runtime
//this router does not do any route matching, it relies on third parties for that
func NewDynamicRouter(routeMatcher RouteMatcherFunc, loggerFactory log.Factory) *dynamicRouter {
	return &dynamicRouter{
		new(sync.Map),
		routeMatcher,
		loggerFactory(nil),
	}
}

//GetHandler returns the router http.Handler
func GetHandler(router *dynamicRouter) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		route, routeMatch := MatchRoute(router.routes, request)
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

		route.handler.ServeHTTP(writer, request.WithContext(ctx))
	})
}

//AddRoute adds a new route
func AddRoute(router *dynamicRouter) func(path, pathPrefix string, methods []string, handler http.Handler) (string, error) {
	return func(path, pathPrefix string, methods []string, handler http.Handler) (string, error) {
		route := Route{
			Path:       path,
			PathPrefix: pathPrefix,
			Methods:    methods,
			handler:    handler,
			UID:        uuid.NewV4().String(),
		}

		route.matcher = router.routeMatcher(route)
		err := validateRoute(router, route)
		if err != nil {
			router.logger.Error("invalid route", zap.Error(err))
			return "", err
		}

		router.routes.Store(route.UID, route)
		router.logger.Info(fmt.Sprintf("DynamicRouter: Added new route: id: %s; pathPrefix: %s; path %s", route.UID, route.PathPrefix, route.Path))
		return route.UID, nil
	}
}

func validateRoute(router *dynamicRouter, route Route) error {
	err := error(nil)

	//check for multiple registrations
	router.routes.Range(func(key, value interface{}) bool {
		r := value.(Route)
		if r.String() == route.String() {
			err = errors.New("DynamicRouter: multiple registrations for : " + route.String())
			return false
		}
		return true
	})

	return err
}

//RemoveRoute removes a route
func RemoveRoute(router *dynamicRouter) func(routeId string) {
	return func(routeId string) {
		route, ok := router.routes.Load(routeId)
		if !ok {
			router.logger.Error("DynamicRouter: Route does not exist " + routeId)
		}

		router.routes.Delete(routeId)
		router.logger.Info(fmt.Sprintf("DynamicRouter: Deleted route id: %s; pathPrefix: %s; path %s", route.(Route).UID, route.(Route).PathPrefix, route.(Route).Path))
	}
}
