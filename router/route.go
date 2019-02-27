package router

import (
	"net/http"
	"sync"
	"time"
)

type RouteMatcherFunc func(route Route) func(request *http.Request) RouteMatch

type Route struct {
	UID        string
	Path       string
	PathPrefix string
	Methods    []string
	Timeout    time.Duration
	matcher    func(request *http.Request) RouteMatch
	handler    http.Handler
}

type RouteMatch struct {
	Matched bool
	Vars    map[string]string
}

func (r Route) String() string {
	return r.PathPrefix + r.Path
}

func matchRoute(routes *sync.Map, request *http.Request) (Route, RouteMatch) {

	var resRM RouteMatch
	var resR Route

	routes.Range(func(key, value interface{}) bool {
		r := value.(Route)
		rm := r.matcher(request)
		if rm.Matched {
			resRM = RouteMatch{rm.Matched, rm.Vars}
			resR = r
			return false
		}
		return true
	})

	return resR, resRM
}
