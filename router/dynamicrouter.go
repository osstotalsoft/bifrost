package router

import (
	"context"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

type PreFilterFunc func(request *http.Request) error
type PostFilterFunc func(request, proxyRequest *http.Request, proxyResponse *http.Response) ([]byte, error)

type dynamicRouter struct {
	preFilters   []PreFilterFunc
	postFilters  []PostFilterFunc
	routes       *sync.Map
	routeMatcher RouteMatcherFunc
}

func NewDynamicRouter(routeMatcher RouteMatcherFunc) *dynamicRouter {
	return &dynamicRouter{
		[]PreFilterFunc{},
		[]PostFilterFunc{},
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
		err := runPreFilters(router.preFilters, newReq)
		if err != nil {
			handleFilterError(writer, newReq, err)
			return
		}

		route.handler.ServeHTTP(writer, newReq)
	})
}

func AddPreFilter(router *dynamicRouter) func(f PreFilterFunc) {
	return func(f PreFilterFunc) {
		router.preFilters = append(router.preFilters, f)
	}
}

func AddPostFilter(router *dynamicRouter) func(f PostFilterFunc) {
	return func(f PostFilterFunc) {
		router.postFilters = append(router.postFilters, f)
	}
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

func runPreFilters(preFilters []PreFilterFunc, request *http.Request) error {
	for _, filter := range preFilters {
		err := filter(request)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleFilterError(responseWriter http.ResponseWriter, request *http.Request, err error) {
	responseWriter.Header().Set("Content/Type", "text/html")
	responseWriter.WriteHeader(500)
	_, err = responseWriter.Write([]byte(err.Error()))
	if err != nil {
		log.Errorln(err)
	}
}
