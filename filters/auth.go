package filters

import (
	"github.com/auth0-community/go-auth0"
	"github.com/mitchellh/mapstructure"
	"github.com/osstotalsoft/bifrost/gateway"
	"github.com/osstotalsoft/bifrost/strutils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"net/http"
	"strings"
	"time"
)

const AuthorizationFilterCode = "auth"

type AuthorizationOptions struct {
	Authority        string `mapstructure:"authority"`
	Audience         string `mapstructure:"audience"`
	WellKnownJwksUrl string `mapstructure:"well_known_jwks"`
}

type AuthorizationEndpointOptions struct {
	ClaimsRequirement map[string]string `mapstructure:"claims_requirement"`
	AllowedScopes     []string          `mapstructure:"allowed_scopes"`
}

type customClaims struct {
	jwt.Claims
	Scopes      string `json:"scope,omitempty"`
	OtherClaims map[string]interface{}
}

func AuthorizationFilter(opts AuthorizationOptions) func(endpoint gateway.Endpoint) func(next http.Handler) http.Handler {
	client := auth0.NewJWKClient(auth0.JWKClientOptions{URI: opts.WellKnownJwksUrl}, nil)
	configuration := auth0.NewConfiguration(client, []string{opts.Audience}, opts.Authority, jose.RS256)
	validator := auth0.NewValidator(configuration, nil)

	return func(endpoint gateway.Endpoint) func(next http.Handler) http.Handler {
		cfg := AuthorizationEndpointOptions{}
		if fl, ok := endpoint.Filters[AuthorizationFilterCode]; ok {
			err := mapstructure.Decode(fl, &cfg)
			if err != nil {
				log.Errorf("AuthorizationFilter: Cannot find or decode AuthorizationEndpointOptions for authorization filter: %v", err)
			}
		}

		return func(next http.Handler) http.Handler {
			if !endpoint.Secured {
				return next
			}
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				//log.Info("AuthorizationFilter")
				start := time.Now()

				token, err := validator.ValidateRequest(request)
				if err != nil {
					log.Errorln("AuthorizationFilter: Token is not valid:", token, err)
					Unauthorized(writer)
					return
				}

				log.Debugf("AuthorizationFilter: ValidateRequest took %s", time.Since(start))

				if len(cfg.AllowedScopes) > 0 || len(cfg.ClaimsRequirement) > 0 {
					claims := customClaims{}
					err = validator.Claims(request, token, &claims)
					if err != nil {
						log.Debug(err)
						log.Debug("AuthorizationFilter: Invalid claims", token)
						Unauthorized(writer)
						return
					}

					log.Debugf("AuthorizationFilter: Claims decoder took %s", time.Since(start))

					if len(cfg.AllowedScopes) > 0 {
						hasScope := checkScopes(cfg.AllowedScopes, strings.Split(claims.Scopes, ""))
						if !hasScope {
							Forbidden(writer)
							return
						}
						log.Debugf("AuthorizationFilter: CheckScopes took %s", time.Since(start))
					}

					if len(cfg.ClaimsRequirement) > 0 {

					}
				}
				next.ServeHTTP(writer, request)
			})
		}
	}
}

func Unauthorized(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusUnauthorized)
	_, _ = writer.Write([]byte("Unauthorized"))
}

func Forbidden(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusForbidden)
	_, _ = writer.Write([]byte("Insufficient scopes."))
}

func checkScopes(requiredScopes []string, userScopes []string) bool {
	inter := strutils.Intersection(requiredScopes, userScopes)
	return len(inter) > 0
}
