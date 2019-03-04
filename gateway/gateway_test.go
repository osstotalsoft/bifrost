package gateway

import (
	"github.com/osstotalsoft/bifrost/config"
	"github.com/osstotalsoft/bifrost/servicediscovery"
	"net/http"
	"testing"
)

type gateTest struct {
	title               string
	service             servicediscovery.Service
	expectedPath        string
	expectedPathPrefix  string
	expectedDestination string
}

var (
	testConfig1 = config.Config{
		DownstreamPathPrefix: "",
		UpstreamPathPrefix:   "/api",
		Endpoints: []config.Endpoint{
			{
				UpstreamPathPrefix:   "/api/v1",
				DownstreamPathPrefix: "/users",
				DownstreamPath:       "",
				UpstreamPath:         "",
				ServiceName:          "users",
				Methods:              nil,
			},
			{
				UpstreamPathPrefix:   "/api/v2",
				DownstreamPathPrefix: "/dealers2",
				DownstreamPath:       "",
				UpstreamPath:         "",
				ServiceName:          "dealers",
				Methods:              nil,
			},
			{
				UpstreamPathPrefix:   "/api/offers",
				DownstreamPathPrefix: "",
				DownstreamPath:       "",
				UpstreamPath:         "",
				ServiceName:          "offers",
				Methods:              nil,
			},
			{
				UpstreamPathPrefix:   "/api/offers",
				DownstreamPathPrefix: "/offers2",
				DownstreamPath:       "/add_offer/{id}",
				UpstreamPath:         "/add/{id}",
				ServiceName:          "offers3",
				Methods:              []string{"POST"},
			},
			{
				UpstreamPathPrefix:   "/",
				DownstreamPathPrefix: "/offers4",
				DownstreamPath:       "",
				UpstreamPath:         "",
				ServiceName:          "offers4",
			},
		},
	}

	testCases1 = []gateTest{
		{
			title: "serviceWithPrefixNoPath",
			service: servicediscovery.Service{
				Resource:  "users",
				Namespace: "app",
				Address:   "http://users.app:80/",
				Secured:   false},
			expectedPath:        "",
			expectedPathPrefix:  "/users",
			expectedDestination: "http://users.app:80/api/v1",
		},
		{
			title: "serviceWithDefaults",
			service: servicediscovery.Service{
				Resource:  "partners",
				Namespace: "app",
				Address:   "http://partners.app:80/",
				Secured:   false},
			expectedPath:        "",
			expectedPathPrefix:  "/partners",
			expectedDestination: "http://partners.app:80/api",
		},
		{
			title: "serviceWithPrefix2",
			service: servicediscovery.Service{
				Resource:  "dealers",
				Namespace: "app",
				Address:   "http://dealers.app:80/",
				Secured:   false},
			expectedPath:        "",
			expectedPathPrefix:  "/dealers2",
			expectedDestination: "http://dealers.app:80/api/v2",
		},
		{
			title: "serviceWithPrefix3",
			service: servicediscovery.Service{
				Resource:  "offers",
				Namespace: "app",
				Address:   "http://offers.app:80/",
				Secured:   false},
			expectedPath:        "",
			expectedPathPrefix:  "/offers",
			expectedDestination: "http://offers.app:80/api/offers",
		},
		{
			title: "serviceWithPrefix4",
			service: servicediscovery.Service{
				Resource:  "offers3",
				Namespace: "app",
				Address:   "http://offers3.app:80/",
				Secured:   false},
			expectedPath:        "/add_offer/{id}",
			expectedPathPrefix:  "/offers2",
			expectedDestination: "http://offers3.app:80/api/offers/add/{id}",
		},
		{
			title: "serviceWithPrefix5",
			service: servicediscovery.Service{
				Resource:  "offers4",
				Namespace: "app",
				Address:   "http://offers4.app:80/",
				Secured:   false},
			expectedPath:        "",
			expectedPathPrefix:  "/offers4",
			expectedDestination: "http://offers4.app:80/",
		},
	}
)

func TestAddService(t *testing.T) {
	gate := NewGateway(&testConfig1)
	RegisterHandler(gate)("http", func(endpoint Endpoint) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		})
	})

	t.Run("group", func(t *testing.T) {
		for _, tc := range testCases1 {
			tc := tc
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				endp := internalAddService(gate, tc.service, func(path string, pathPrefix string, methods []string, handler http.Handler) (id string, e error) {
					return "1", nil
				})

				if endp[0].DownstreamPath != tc.expectedPath {
					t.Fatalf("expectedPath %v, but got %v", tc.expectedPath, endp[0].DownstreamPath)
				}
				if endp[0].DownstreamPathPrefix != tc.expectedPathPrefix {
					t.Fatalf("expectedPathPrefix %v, but got %v", tc.expectedPathPrefix, endp[0].DownstreamPathPrefix)
				}
				if endp[0].UpstreamURL != tc.expectedDestination {
					t.Fatalf("expectedDestination %v, but got %v", tc.expectedDestination, endp[0].UpstreamURL)
				}
			})
		}
	})
}
