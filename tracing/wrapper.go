package tracing

import (
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/middleware"
	"net/http"
)

//Wrap http.Handler with opentracing
func Wrap(inner http.Handler) http.Handler {

	tracer := opentracing.GlobalTracer()

	//setup opentracing for main handler
	return nethttp.Middleware(tracer, inner, nethttp.OperationNameFunc(func(r *http.Request) string {
		return "HTTP " + r.Method + " " + r.URL.String()
	}))
}

func WrapMiddleware(inner middleware.Func, operation string) middleware.Func {
	return func(endpoint abstraction.Endpoint) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				span, ctx := opentracing.StartSpanFromContext(request.Context(), operation)
				defer span.Finish()
				inner(endpoint)(next).ServeHTTP(writer, request.WithContext(ctx))
			})
		}
	}
}
