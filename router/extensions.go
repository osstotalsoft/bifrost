package router

import (
	"github.com/gorilla/mux"
	"net/http"
)

//GorillaMuxRouteMatcher is used for route matching
func GorillaMuxRouteMatcher(route Route) func(request *http.Request) RouteMatch {
	rr := new(mux.Route)

	if route.PathPrefix != "" {
		rr = rr.PathPrefix(route.PathPrefix)
	}
	if route.Path != "" {
		rr = rr.Path(route.Path)
	}
	if route.Methods != nil && len(route.Methods) > 0 {
		rr = rr.Methods(route.Methods...)
	}

	return func(request *http.Request) RouteMatch {
		var match mux.RouteMatch
		b := rr.Match(request, &match)
		return RouteMatch{b, match.Vars}
	}
}
