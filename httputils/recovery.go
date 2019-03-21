package httputils

import (
	"github.com/osstotalsoft/bifrost/log"
	"go.uber.org/zap"
	"net/http"
)

//RecoveryHandler handles pipeline panic
func RecoveryHandler(loggerFactory log.Factory) func(inner http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					loggerFactory(req.Context()).Error("internal server error", zap.Any("error", err), zap.Stack("stack_trace"))
				}
			}()

			inner.ServeHTTP(w, req)
		})
	}
}
