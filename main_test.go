package main

import (
	r "bifrost/router"
	"bifrost/servicediscovery"
	"bifrost/utils"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
)

type mainTest struct {
	title               string
	service             servicediscovery.Service
	responseFromGateway string
	requestUrl          string
	backendUrl          string
}

var (
	testConfig2 = Config{
		DownstreamPathPrefix: "",
		UpstreamPathPrefix:   "/api",
		Endpoints: []Endpoint{
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
		},
	}

	testCases2 = []mainTest{
		{
			title: "serviceWithPrefixNoPath",
			service: servicediscovery.Service{
				Resource:  "users",
				Namespace: "app",
				Address:   "http://users.app:80/",
				Secured:   false},
			responseFromGateway: "responseFromGateway",
			requestUrl:          "/users",
			backendUrl:          "/api/v1/users",
		},
		{
			title: "serviceWithDefaults",
			service: servicediscovery.Service{
				Resource:  "partners",
				Namespace: "app",
				Address:   "http://partners.app:80/",
				Secured:   false},
			responseFromGateway: "responseFromGateway1",
			requestUrl:          "/partners/details/4545",
			backendUrl:          "/api/v1/partners/details/4545",
		},
		{
			title: "serviceWithPrefix2",
			service: servicediscovery.Service{
				Resource:  "dealers",
				Namespace: "app",
				Address:   "http://dealers.app:80/",
				Secured:   false},
			responseFromGateway: "responseFromGateway2",
			requestUrl:          "/dealers2",
			backendUrl:          "/api/v2",
		},
		{
			title: "serviceWithPrefix3",
			service: servicediscovery.Service{
				Resource:  "offers1",
				Namespace: "app",
				Address:   "http://offers1.app:80/",
				Secured:   false},
			responseFromGateway: "responseFromGateway3",
			requestUrl:          "/offers1/4435",
			backendUrl:          "/api/offers1/4435",
		},
		{
			title: "serviceWithPrefix4",
			service: servicediscovery.Service{
				Resource:  "offers2",
				Namespace: "app",
				Address:   "http://offers2.app:80/",
				Secured:   false},
			responseFromGateway: "responseFromGateway4",
			requestUrl:          "/offers2/add_offer/555",
			backendUrl:          "/api/offers2/add/555",
		},
		{
			title: "serviceWithPrefix5",
			service: servicediscovery.Service{
				Resource:  "offers3",
				Namespace: "app",
				Address:   "http://offers3.app:80/",
				Secured:   false},
			responseFromGateway: "responseFromGateway5",
			requestUrl:          "/offers3",
			backendUrl:          "/offers3",
		},
	}
)

func TestGateway(t *testing.T) {

	mux := http.NewServeMux()
	backendServer := httptest.NewServer(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
		for _, tc := range testCases2 {
			if tc.backendUrl == request.RequestURI {
				_, _ = w.Write([]byte(tc.responseFromGateway))
				return
			}
		}
		http.NotFound(w, request)
	})
	defer backendServer.Close()

	dynRouter := r.NewDynamicRouter(r.GorillaMuxRouteMatcher)
	gateway := NewGateway(&testConfig2)

	frontendProxy := httptest.NewServer(r.GetHandler(dynRouter))
	defer frontendProxy.Close()

	for _, tc := range testCases2 {
		tc := tc
		AddEndpoint(gateway)(func(path, pathPrefix string, methods []string, targetUrl, targetUrlPath, targetUrlPrefix string) string {
			destinationUrl, _ := url.Parse(targetUrl)
			targetUrl = utils.SingleJoiningSlash(backendServer.URL, destinationUrl.Path)
			revProxy := &httputil.ReverseProxy{Director: r.GetDirector(targetUrl, targetUrlPath, targetUrlPrefix)}
			r := r.AddRoute(dynRouter)(path, pathPrefix, methods, revProxy)
			return r.UID
		})(tc.service)
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
					t.Fatalf("expected %v, but got %v", tc.responseFromGateway, string(body))
				}
			})
		}
	})
}

func RandomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(65 + rand.Intn(25)) //A=65 and Z = 65+25
	}
	return string(bytes)
}
