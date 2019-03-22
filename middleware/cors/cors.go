package cors

import (
	"github.com/gorilla/handlers"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/osstotalsoft/bifrost/middleware"
	"net/http"
)

//CORSFilterCode is the code used to register this middleware
const CORSFilterCode = "cors"

// CORSFilter provides Cross-Origin Resource Sharing middleware.
// using gorilla cors handlers
func CORSFilter(allowedOrigins ...string) middleware.Func {
	return func(endpoint abstraction.Endpoint, loggerFactory log.Factory) func(http.Handler) http.Handler {
		originis := handlers.AllowedOrigins(allowedOrigins)
		methods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
		headers := handlers.AllowedHeaders([]string{"Accept", "Accept-Language", "Content-Language", "Origin", "X-Requested-With", "Content-Type", "Authorization"})

		return handlers.CORS(originis, methods, headers, handlers.AllowCredentials())
	}
}