package reverseproxy

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/router"
	"github.com/osstotalsoft/bifrost/strutils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type HttpHandlerConfig struct {
	UpstreamPathPrefix string `mapstructure:"upstream_path_prefix"`
}

type HttpHandlerEndpointConfig struct {
	UpstreamPath       string `mapstructure:"upstream_path"`
	UpstreamPathPrefix string `mapstructure:"upstream_path_prefix"`
}

func NewReverseProxy() handler.Func {

	return func(endPoint abstraction.Endpoint) http.Handler {
		//https://github.com/golang/go/issues/16012
		//http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

		return &httputil.ReverseProxy{
			Director:       GetDirector(endPoint.UpstreamURL, endPoint.UpstreamPath, endPoint.UpstreamPathPrefix),
			ModifyResponse: ModifyResponse,
		}
	}
}

func ModifyResponse(response *http.Response) error {
	//hack when upstream service has cors enabled; cors will be handled by the gateway
	response.Header.Del("Access-Control-Allow-Origin")
	return nil
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
		req.Host = target.Host
		if targetUrlPath == "" {
			a := req.URL.EscapedPath()
			req.URL.Path = strutils.SingleJoiningSlash(target.Path, strings.TrimPrefix(a, routeContext.PathPrefix))

			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
		} else {
			req.URL.Path = target.Path
			req.URL.RawQuery = targetQuery
		}

		req.URL.Path = replaceVarsInTarget(req.URL.Path, routeContext.Vars)
		req.URL.RawQuery = replaceVarsInTarget(req.URL.RawQuery, routeContext.Vars)

		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}

		log.Tracef("Forwarding request from %v to %v", initial, req.URL.String())
	}
}

func replaceVarsInTarget(targetUrl string, vars map[string]string) string {
	for key, val := range vars {
		targetUrl = strings.Replace(targetUrl, "{"+key+"}", val, 1)
	}

	return targetUrl
}
