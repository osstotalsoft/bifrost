package router

import (
	"net/http"
	"sync"
	"time"
)

//RouteMatcherFunc type signature that any route matcher have to implement
type RouteMatcherFunc func(route Route) func(request *http.Request) RouteMatch

//Route stores the information about a certain route
type Route struct {
	UID        string
	Path       string
	PathPrefix string
	Methods    []string
	Timeout    time.Duration
	matcher    func(request *http.Request) RouteMatch
	handler    http.Handler
}

//RouteMatch is the result of a route matching
type RouteMatch struct {
	Matched bool
	Vars    map[string]string
}

func (r Route) String() string {
	return r.PathPrefix + r.Path
}

//MatchRoute checks all the route if they match the incoming request
func MatchRoute(routes *sync.Map, request *http.Request) (Route, RouteMatch) {

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
