package tracing

import (
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/middleware"
	"net/http"
)

//Wrap http.Handler with opentracing
func StartSpan(inner http.Handler) http.Handler {

	tracer := opentracing.GlobalTracer()

	//setup opentracing for main handler
	return nethttp.Middleware(tracer, inner, nethttp.OperationNameFunc(func(r *http.Request) string {
		return "HTTP " + r.Method + " " + r.URL.String()
	}))
}

func MiddlewareStartSpan(operation string) func(inner middleware.Func) middleware.Func {
	return func(inner middleware.Func) middleware.Func {
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
}

func HandlerStartSpan(operation string) func(inner handler.Func) handler.Func {
	return func(inner handler.Func) handler.Func {
		return func(endpoint abstraction.Endpoint) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				span, ctx := opentracing.StartSpanFromContext(request.Context(), operation)
				defer span.Finish()
				inner(endpoint).ServeHTTP(writer, request.WithContext(ctx))
			})
		}
	}
}
