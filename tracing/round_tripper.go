package tracing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"net/http"
)

type roundTripper struct {
	http.RoundTripper
}

func NewRoundTripperWithOpenTrancing() *roundTripper {
	return &roundTripper{RoundTripper: http.DefaultTransport}
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	sp, _ := opentracing.StartSpanFromContext(req.Context(), "Reverse Proxy to "+req.URL.String())
	defer sp.Finish()

	ext.SpanKindRPCClient.Set(sp)
	ext.Component.Set(sp, "net/http")

	ext.HTTPMethod.Set(sp, req.Method)
	ext.HTTPUrl.Set(sp, req.URL.String())

	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	_ = sp.Tracer().Inject(sp.Context(), opentracing.HTTPHeaders, carrier)

	resp, err := rt.RoundTripper.RoundTrip(req)
	ext.HTTPStatusCode.Set(sp, uint16(resp.StatusCode))

	return resp, err
}
