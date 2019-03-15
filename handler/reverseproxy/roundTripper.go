package reverseproxy

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"net/http"
)

type roundTripper struct {
	http.RoundTripper
}

func newRoundTripper() *roundTripper {
	return &roundTripper{RoundTripper: http.DefaultTransport}
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	tracer := opentracing.GlobalTracer()
	parent := opentracing.SpanFromContext(req.Context())
	var spanctx opentracing.SpanContext
	if parent != nil {
		spanctx = parent.Context()
	}
	root := tracer.StartSpan("Reverse Proxy", opentracing.ChildOf(spanctx))

	ctx := root.Context()
	sp := tracer.StartSpan(req.URL.Scheme+" "+req.Method, opentracing.ChildOf(ctx))
	ext.SpanKindRPCClient.Set(sp)
	ext.Component.Set(sp, "net/http")

	ext.HTTPMethod.Set(sp, req.Method)
	ext.HTTPUrl.Set(sp, req.URL.String())

	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	sp.Tracer().Inject(sp.Context(), opentracing.HTTPHeaders, carrier)

	resp, err := rt.RoundTripper.RoundTrip(req)
	ext.HTTPStatusCode.Set(sp, uint16(resp.StatusCode))

	defer sp.Finish()
	defer root.Finish()

	return resp, err
}
