package auth

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
	"github.com/mitchellh/mapstructure"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/middleware"
	"github.com/osstotalsoft/oidc-jwt-go"
	"github.com/osstotalsoft/oidc-jwt-go/discovery"
	log "github.com/sirupsen/logrus"
	"net/http"
)

const AuthorizationFilterCode = "auth"

type AuthorizationOptions struct {
	Authority      string `mapstructure:"authority"`
	Audience       string `mapstructure:"audience"`
	SecretProvider oidc.SecretProvider
}

type AuthorizationEndpointOptions struct {
	ClaimsRequirement map[string]string `mapstructure:"claims_requirement"`
	AllowedScopes     []string          `mapstructure:"allowed_scopes"`
}

func AuthorizationFilter(opts AuthorizationOptions) middleware.Func {
	return func(endpoint abstraction.Endpoint) func(http.Handler) http.Handler {
		cfg := AuthorizationEndpointOptions{}
		if fl, ok := endpoint.Filters[AuthorizationFilterCode]; ok {
			err := mapstructure.Decode(fl, &cfg)
			if err != nil {
				log.Errorf("AuthorizationFilter: Cannot find or decode AuthorizationEndpointOptions for authorization filter: %v", err)
			}
		}

		if opts.SecretProvider == nil {
			opts.SecretProvider = oidc.NewOidcSecretProvider(discovery.NewClient(discovery.Options{opts.Authority}))
		}
		validator := oidc.NewJWTValidator(jwtRequest.OAuth2Extractor, opts.SecretProvider, opts.Audience, opts.Authority)

		return func(next http.Handler) http.Handler {
			if !endpoint.Secured {
				return next
			}
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				token, err := validator(request)
				if err != nil {
					log.Errorln("AuthorizationFilter: Token is not valid:", err)
					Unauthorized(writer)
					return
				}

				if len(cfg.AllowedScopes) > 0 || len(cfg.ClaimsRequirement) > 0 {
					if len(cfg.AllowedScopes) > 0 {
						hasScope := checkScopes(cfg.AllowedScopes, token.Claims.(jwt.MapClaims)["scope"].([]interface{}))
						if !hasScope {
							Forbidden(writer)
							return
						}
					}

					if len(cfg.ClaimsRequirement) > 0 {

					}
				}

				ctx := context.WithValue(request.Context(), abstraction.ContextClaimsKey, token.Claims)
				request = request.WithContext(ctx)

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

func checkScopes(requiredScopes []string, userScopes []interface{}) bool {
	for _, el := range userScopes {
		for _, el1 := range requiredScopes {
			if el == el1 {
				return true
			}
		}
	}
	return false
}
