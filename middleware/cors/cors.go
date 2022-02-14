package cors

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/osstotalsoft/bifrost/middleware"
	"github.com/rs/cors"
	"net/http"
)

//CORSFilterCode is the code used to register this middleware
const CORSFilterCode = "cors"

//AuthorizationOptions are the options configured for all endpoints
type Options struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// CORSFilter provides Cross-Origin Resource Sharing middleware.
// using RS cors handlers
func CORSFilter(options Options) middleware.Func {
	return func(endpoint abstraction.Endpoint, loggerFactory log.Factory) func(http.Handler) http.Handler {

		c := cors.New(cors.Options{
			AllowedOrigins: options.AllowedOrigins,
			AllowedMethods: []string{
				http.MethodHead,
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
			},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		})

		return c.Handler
	}
}
