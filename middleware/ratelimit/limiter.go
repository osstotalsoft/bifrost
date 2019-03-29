package ratelimit

import (
	"fmt"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/osstotalsoft/bifrost/middleware"
	"golang.org/x/time/rate"
	"net/http"
)

const RateLimitingFilterCode = "ratelimit"

//DefaultGlobalRequestLimit defines max nr of request / route / second
const DefaultGlobalRequestLimit = 5000
const MaxRequestLimit = 10000

//RateLimiting is a middleware which can limit the number of request / route / second
//and then return StatusTooManyRequests response if the limit is reached
func RateLimiting(limit int) middleware.Func {
	return func(endpoint abstraction.Endpoint, loggerFactory log.Factory) func(http.Handler) http.Handler {
		limiter := rate.NewLimiter(rate.Limit(limit), limit)

		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setResponseHeaders(limiter.Limit(), w, r)

				if limiter.Allow() == false {
					http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
					return
				}

				next.ServeHTTP(w, r)
			})
		}
	}
}

func setResponseHeaders(lmt rate.Limit, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Rate-Limit-Limit", fmt.Sprintf("%.2f", lmt))
	w.Header().Add("X-Rate-Limit-Duration", "1")
	w.Header().Add("X-Rate-Limit-Request-Forwarded-For", r.Header.Get("X-Forwarded-For"))
	w.Header().Add("X-Rate-Limit-Request-Remote-Addr", r.RemoteAddr)
}
