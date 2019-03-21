package main

import (
	"github.com/osstotalsoft/bifrost/config"
	"github.com/osstotalsoft/bifrost/gateway"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/handler/reverseproxy"
	"github.com/osstotalsoft/bifrost/httputils"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/osstotalsoft/bifrost/middleware"
	"github.com/osstotalsoft/bifrost/middleware/cors"
	r "github.com/osstotalsoft/bifrost/router"
	"github.com/osstotalsoft/bifrost/servicediscovery"
	"github.com/osstotalsoft/bifrost/tracing"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type mainTest struct {
	title               string
	responseFromGateway string
	requestUrl          string
	backendUrl          string
}

var (
	testConfig2 = config.Config{
		DownstreamPathPrefix: "",
		UpstreamPathPrefix:   "/api",
		Endpoints: []config.EndpointConfig{
			{
				UpstreamPathPrefix:   "/api/v1/users",
				UpstreamPath:         "",
				DownstreamPathPrefix: "/users",
				DownstreamPath:       "",
				ServiceName:          "users",
				Methods:              nil,
			},
			{
				UpstreamPathPrefix:   "/api/v1/partners",
				UpstreamPath:         "",
				DownstreamPathPrefix: "/partners",
				DownstreamPath:       "",
				ServiceName:          "partners",
				Methods:              nil,
			},
			{
				UpstreamPathPrefix:   "/api/v2",
				UpstreamPath:         "",
				DownstreamPathPrefix: "/dealers2",
				DownstreamPath:       "",
				ServiceName:          "dealers",
				Methods:              nil,
			},
			{
				UpstreamPathPrefix:   "/api/offers1",
				UpstreamPath:         "",
				DownstreamPathPrefix: "/offers1",
				DownstreamPath:       "",
				ServiceName:          "offers1",
				Methods:              nil,
			},
			{
				UpstreamPathPrefix:   "/api/offers2",
				UpstreamPath:         "/add/{v1}",
				DownstreamPathPrefix: "/offers2",
				DownstreamPath:       `/add_offer/{v1}`,
				ServiceName:          "offers2",
				Methods:              []string{"GET"},
			},
			{
				UpstreamPathPrefix:   "/offers3",
				UpstreamPath:         "",
				DownstreamPathPrefix: "/offers3",
				DownstreamPath:       "",
				ServiceName:          "offers3",
			},
			{
				UpstreamPathPrefix:   "/api/v1",
				UpstreamPath:         "/offers4?id={id}",
				DownstreamPathPrefix: "/offers4",
				DownstreamPath:       "/{id}",
				ServiceName:          "offers4",
				Methods:              nil,
			},
		},
	}

	serviceList = []*servicediscovery.Service{
		{Resource: "users", Secured: false},
		{Resource: "partners", Secured: false},
		{Resource: "dealers", Secured: false},
		{Resource: "offers1", Secured: false},
		{Resource: "offers2", Secured: false},
		{Resource: "offers3", Secured: false},
		{Resource: "offers4", Secured: false},
	}

	testCases2 = []*mainTest{
		{
			title:               "serviceWithPrefixNoPath",
			responseFromGateway: "responseFromGateway",
			requestUrl:          "/users",
			backendUrl:          "/api/v1/users",
		},
		{
			title:               "serviceWithDefaults",
			responseFromGateway: "responseFromGateway1",
			requestUrl:          "/partners/details/4545",
			backendUrl:          "/api/v1/partners/details/4545",
		},
		{
			title:               "serviceWithPrefix2",
			responseFromGateway: "responseFromGateway2",
			requestUrl:          "/dealers2",
			backendUrl:          "/api/v2",
		},
		{
			title:               "serviceWithPrefix3",
			responseFromGateway: "responseFromGateway3",
			requestUrl:          "/offers1/4435",
			backendUrl:          "/api/offers1/4435",
		},
		{
			title:               "serviceWithPrefix4",
			responseFromGateway: "responseFromGateway4",
			requestUrl:          "/offers2/add_offer/555",
			backendUrl:          "/api/offers2/add/555",
		},
		{
			title:               "serviceWithPrefix5",
			responseFromGateway: "responseFromGateway5",
			requestUrl:          "/offers3",
			backendUrl:          "/offers3",
		},
		{
			title:               "serviceWithPrefix6",
			responseFromGateway: "responseFromGateway6",
			requestUrl:          "/offers4/555",
			backendUrl:          "/api/v1/offers4?id=555",
		},
		{
			title:               "testEncodingUrl",
			responseFromGateway: "testEncodingUrlResponse",
			requestUrl:          "/dealers2/singWebApp%2F2137%2F6026a931-7c35",
			backendUrl:          "/api/v2/singWebApp%2F2137%2F6026a931-7c35",
		},
		{
			title:               "testEncodingUrl2",
			responseFromGateway: "testEncodingUrl2Response",
			requestUrl:          "/dealers2/singWe?search=&partnerId=",
			backendUrl:          "/api/v2/singWe?search=&partnerId=",
		},
	}
)

func TestGatewayReverseProxy(t *testing.T) {
	backendServer := startBackend()
	defer backendServer.Close()

	logger, _ := zap.NewDevelopment()
	factory := log.ZapLoggerFactory(logger)

	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher, factory)
	gate := gateway.NewGateway(&testConfig2, factory)
	gateway.RegisterHandler(gate)(handler.ReverseProxyHandlerType, reverseproxy.NewReverseProxy(http.DefaultTransport, factory))
	frontendProxy := httptest.NewServer(r.GetHandler(dynRouter))
	defer frontendProxy.Close()

	for _, service := range serviceList {
		gateway.AddService(gate)(r.AddRoute(dynRouter))(*service)
	}

	t.Run("group", func(t *testing.T) {
		for _, tc := range testCases2 {
			tc := tc
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				resp, err := http.Get(frontendProxy.URL + tc.requestUrl)
				if err != nil {
					t.Fatal(err)
				}

				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}

				if string(body) != tc.responseFromGateway {
					t.Errorf("test %s failed : expected %v, but got %v", tc.title, tc.responseFromGateway, string(body))
				}
			})
		}
	})
}

func startBackend() *httptest.Server {
	mux := http.NewServeMux()
	backendServer := httptest.NewServer(mux)

	for _, tc := range serviceList {
		tc.Address = backendServer.URL
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
		url1, _ := url.PathUnescape(request.RequestURI)

		for _, tc := range testCases2 {
			if tc.backendUrl == url1 {
				_, _ = w.Write([]byte(tc.responseFromGateway))
				return
			}
		}
		http.NotFound(w, request)
	})

	return backendServer
}

func BenchmarkGatewayReverseProxy(b *testing.B) {
	backendServer := startBackend()
	defer backendServer.Close()

	factory := log.ZapLoggerFactory(zap.NewNop())
	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher, factory)
	gate := gateway.NewGateway(&testConfig2, factory)
	gateway.RegisterHandler(gate)(handler.ReverseProxyHandlerType, reverseproxy.NewReverseProxy(http.DefaultTransport, factory))

	gateHandler := r.GetHandler(dynRouter)

	for _, service := range serviceList {
		gateway.AddService(gate)(r.AddRoute(dynRouter))(*service)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/offers2/add_offer/555", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gateHandler.ServeHTTP(w, req)
		w.Result()
	}
}

func BenchmarkGateway(b *testing.B) {
	backendServer := startBackend()
	defer backendServer.Close()

	factory := log.ZapLoggerFactory(zap.NewNop())
	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher, factory)
	gate := gateway.NewGateway(&testConfig2, factory)

	gateway.UseMiddleware(gate)(cors.CORSFilterCode, middleware.Compose(
		tracing.MiddlewareStartSpan("CORS Filter"),
	)(cors.CORSFilter("*")))

	gateway.RegisterHandler(gate)(handler.ReverseProxyHandlerType, reverseproxy.NewReverseProxy(http.DefaultTransport, factory))
	gateHandler := httputils.RecoveryHandler(factory)(r.GetHandler(dynRouter))

	for _, service := range serviceList {
		gateway.AddService(gate)(r.AddRoute(dynRouter))(*service)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/offers2/add_offer/555", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gateHandler.ServeHTTP(w, req)
		w.Result()
	}
}
