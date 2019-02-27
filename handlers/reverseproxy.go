package handlers

import (
	"bifrost/router"
	"bifrost/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func NewReverseProxy(targetUrl, targetUrlPath, targetUrlPrefix string) http.Handler {

	//https://github.com/golang/go/issues/16012
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	return &httputil.ReverseProxy{Director: GetDirector(targetUrl, targetUrlPath, targetUrlPrefix)}
}

func GetDirector(targetUrl, targetUrlPath, targetUrlPrefix string) func(req *http.Request) {
	return func(req *http.Request) {
		routeContext := req.Context().Value(router.ContextRouteKey).(router.RouteContext)
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
