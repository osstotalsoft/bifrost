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

//AuthorizationOptions are the options configured for all endpoints
type Options struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// CORSFilter provides Cross-Origin Resource Sharing middleware.
// using gorilla cors handlers
func CORSFilter(options Options) middleware.Func {
	return func(endpoint abstraction.Endpoint, loggerFactory log.Factory) func(http.Handler) http.Handler {
		originis := handlers.AllowedOrigins(options.AllowedOrigins)
		methods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
		headers := handlers.AllowedHeaders([]string{"Accept", "Accept-Language", "Content-Language", "Origin", "X-Requested-With", "Content-Type", "Authorization"})

		return handlers.CORS(originis, methods, headers, handlers.AllowCredentials())
	}
}
