package filters

import (
	"github.com/auth0-community/go-auth0"
	"github.com/osstotalsoft/bifrost/gateway"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	"net/http"
)

func AuthorizationFilter() func(endpoint gateway.Endpoint) func(next http.Handler) http.Handler {

	client := auth0.NewJWKClient(auth0.JWKClientOptions{URI: "https://tech0.eu.auth0.com/.well-known/jwks.json"}, nil)
	audience := []string{"http://localhost:8000/api/"}
	configuration := auth0.NewConfiguration(client, audience, "https://tech0.eu.auth0.com/", jose.RS256)
	validator := auth0.NewValidator(configuration, nil)

	return func(endpoint gateway.Endpoint) func(next http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			if !endpoint.Secured {
				return next
			}
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				//log.Info("AuthorizationFilter")
				token, err := validator.ValidateRequest(request)
				if err != nil {
					log.Errorln("Token is not valid:", token, err)
					writer.WriteHeader(http.StatusUnauthorized)
					_, _ = writer.Write([]byte("Unauthorized"))
				}

				next.ServeHTTP(writer, request)
			})
		}
	}

}
