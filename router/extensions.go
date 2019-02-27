package router

import (
	"api-gateway/utils"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strings"
)

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

func GetDirector(targetUrl, targetUrlPath, targetUrlPrefix string) func(req *http.Request) {
	return func(req *http.Request) {
		routeContext := req.Context().Value(ContextRouteKey).(RouteContext)
		initial := req.URL.String()
		target, err := url.Parse(targetUrl)
		if err != nil {
			log.Panicf("Error when converting to url %v ", targetUrl)
			return
		}
		targetQuery := target.RawQuery
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		if targetUrlPath == "" {
			req.URL.Path = utils.SingleJoiningSlash(target.Path, strings.TrimPrefix(req.URL.Path, routeContext.PathPrefix))

			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
		} else {
			req.URL.Path = target.Path
		}

		req.URL.Path = replaceVarsInTarget(req.URL.Path, routeContext.Vars)
		req.URL.RawQuery = replaceVarsInTarget(req.URL.RawQuery, routeContext.Vars)

		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}

		log.Debugf("Forwarding request from %v to %v", initial, req.URL.String())
	}
}

func replaceVarsInTarget(targetUrl string, vars map[string]string) string {
	for key, val := range vars {
		targetUrl = strings.Replace(targetUrl, "{"+key+"}", val, 1)
	}

	return targetUrl
}
