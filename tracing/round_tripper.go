package tracing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"net/http"
)

type roundTripper struct {
	http.RoundTripper
}

//NewRoundTripperWithOpenTrancing creates a new roundTripper with OpenTracing
func NewRoundTripperWithOpenTrancing() *roundTripper {
	return &roundTripper{RoundTripper: http.DefaultTransport}
}

//RoundTrip starts a opentracing span and then delegates the request to the actual roundtripper
func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	sp, _ := opentracing.StartSpanFromContext(req.Context(), "RoundTrip to "+req.URL.String())
	defer sp.Finish()

	ext.SpanKindRPCClient.Set(sp)
	ext.Component.Set(sp, "RoundTripper")

	ext.HTTPMethod.Set(sp, req.Method)
	ext.HTTPUrl.Set(sp, req.URL.String())

	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	_ = sp.Tracer().Inject(sp.Context(), opentracing.HTTPHeaders, carrier)

	resp, err := rt.RoundTripper.RoundTrip(req)
	if resp != nil {
		ext.HTTPStatusCode.Set(sp, uint16(resp.StatusCode))
	} else {
		sp.SetTag("error", true)
		sp.LogFields(log.Error(err))
	}

	return resp, err
}
