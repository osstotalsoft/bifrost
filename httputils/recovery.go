package httputils

import (
	"fmt"
	"github.com/osstotalsoft/bifrost/log"
	"go.uber.org/zap"
	"net/http"
	"runtime"
)

//RecoveryHandler handles pipeline panic
func RecoveryHandler(loggerFactory log.Factory) func(inner http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				if err := recover(); err != nil {

					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]

					loggerFactory(req.Context()).Error("internal server error",
						zap.Any("address", req.RemoteAddr),
						zap.Any("url", req.URL),
						zap.Any("error", err),
						zap.String("runtime_stack", fmt.Sprintf("%s", buf)),
						zap.Stack("stack_trace"))
					//w.WriteHeader(http.StatusInternalServerError)

				}
			}()

			inner.ServeHTTP(w, req)
		})
	}
}
