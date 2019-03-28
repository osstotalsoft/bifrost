package tracing

import (
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/osstotalsoft/bifrost/middleware"
	"net/http"
)

//Wrap http.Handler with opentracing
func SpanWrapper(inner http.Handler) http.Handler {

	tracer := opentracing.GlobalTracer()

	//setup opentracing for main handler
	return nethttp.Middleware(tracer, inner, nethttp.OperationNameFunc(func(r *http.Request) string {
		return "HTTP " + r.Method + ":" + r.URL.Path
	}), nethttp.MWSpanObserver(func(span opentracing.Span, r *http.Request) {
		span.SetTag("http.uri", r.URL.EscapedPath())
	}))
}

func MiddlewareSpanWrapper(operation string) func(inner middleware.Func) middleware.Func {
	return func(inner middleware.Func) middleware.Func {
		return func(endpoint abstraction.Endpoint, loggerFactory log.Factory) func(http.Handler) http.Handler {
			return func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					span, ctx := opentracing.StartSpanFromContext(request.Context(), operation)
					defer span.Finish()
					inner(endpoint, loggerFactory)(next).ServeHTTP(writer, request.WithContext(ctx))
				})
			}
		}
	}
}

func HandlerSpanWrapper(operation string) func(inner handler.Func) handler.Func {
	return func(inner handler.Func) handler.Func {
		return func(endpoint abstraction.Endpoint, loggerFactory log.Factory) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				span, ctx := opentracing.StartSpanFromContext(request.Context(), operation)
				defer span.Finish()
				inner(endpoint, loggerFactory).ServeHTTP(writer, request.WithContext(ctx))
			})
		}
	}
}
