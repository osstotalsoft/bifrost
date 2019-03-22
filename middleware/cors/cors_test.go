package cors

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/log"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	corsOptionMethod           string = "OPTIONS"
	corsAllowOriginHeader      string = "Access-Control-Allow-Origin"
	corsExposeHeadersHeader    string = "Access-Control-Expose-Headers"
	corsMaxAgeHeader           string = "Access-Control-Max-Age"
	corsAllowMethodsHeader     string = "Access-Control-Allow-Methods"
	corsAllowHeadersHeader     string = "Access-Control-Allow-Headers"
	corsAllowCredentialsHeader string = "Access-Control-Allow-Credentials"
	corsRequestMethodHeader    string = "Access-Control-Request-Method"
	corsRequestHeadersHeader   string = "Access-Control-Request-Headers"
	corsOriginHeader           string = "Origin"
	corsVaryHeader             string = "Vary"
	corsOriginMatchAll         string = "*"
)

var endpoint = abstraction.Endpoint{}

func TestCORSFilter(t *testing.T) {
	r := httptest.NewRequest("OPTIONS", "http://www.example.com/", nil)
	r.Header.Set(corsOriginHeader, r.URL.String())
	r.Header.Set(corsRequestMethodHeader, "GET")
	r.Header.Set(corsRequestHeadersHeader, "Authorization")

	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	logger, _ := zap.NewDevelopment()
	CORSFilter("http://www.example.com/")(endpoint, log.ZapLoggerFactory(logger))(testHandler).ServeHTTP(rr, r)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("bad status: got %v want %v", status, http.StatusOK)
	}

	header := rr.Header().Get(corsAllowHeadersHeader)
	if header != "Authorization" {
		t.Fatalf("bad header: expected Authorization header, got empty header.")
	}

	header = rr.Header().Get(corsAllowOriginHeader)
	if header != "http://www.example.com/" {
		t.Fatalf("bad header: expected Access-Control-Allow-Origin:http://www.example.com/  header, got %s", header)
	}
}

func BenchmarkCORSPreflight(b *testing.B) {
	r := httptest.NewRequest("OPTIONS", "http://www.example.com/", nil)
	r.Header.Set("Origin", r.URL.String())
	r.Header.Set(corsRequestMethodHeader, "GET")
	r.Header.Set(corsRequestHeadersHeader, "Authorization")

	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	logger, _ := zap.NewDevelopment()
	h := CORSFilter("*")(endpoint, log.ZapLoggerFactory(logger))(testHandler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(rr, r)
		if status := rr.Code; status != http.StatusOK {
			b.Errorf("bad status: got %v want %v", status, http.StatusOK)
		}
	}
}

func BenchmarkCORSActualRequest(b *testing.B) {
	r := httptest.NewRequest("GET", "http://www.example.com/", nil)
	r.Header.Set("Origin", r.URL.String())

	rr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	logger, _ := zap.NewDevelopment()
	h := CORSFilter("http://www.example.com/")(endpoint, log.ZapLoggerFactory(logger))(testHandler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(rr, r)
		if status := rr.Code; status != http.StatusOK {
			b.Errorf("bad status: got %v want %v", status, http.StatusOK)
		}
	}
}
