package auth

import (
	"context"
	"github.com/golang-jwt/jwt/v4"
	jwtRequest "github.com/golang-jwt/jwt/v4/request"
	"github.com/mitchellh/mapstructure"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/osstotalsoft/bifrost/middleware"
	"github.com/osstotalsoft/oidc-jwt-go"
	"github.com/osstotalsoft/oidc-jwt-go/discovery"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

//AuthorizationFilterCode is the code used to register this middleware
const AuthorizationFilterCode = "auth"

//AuthorizationOptions are the options configured for all endpoints
type AuthorizationOptions struct {
	Authority      string `mapstructure:"authority"`
	SecretProvider oidc.SecretProvider
}

//AuthorizationEndpointOptions are the options configured for each endpoint
type AuthorizationEndpointOptions struct {
	Audience          string            `mapstructure:"audience"`
	Disabled          bool              `mapstructure:"disabled"`
	ClaimsRequirement map[string]string `mapstructure:"claims_requirement"`
	AllowedScopes     []string          `mapstructure:"allowed_scopes"`
}

//AuthorizationFilter is a middleware that handles authorization using
//an OpendID Connect server
func AuthorizationFilter(opts AuthorizationOptions) middleware.Func {
	if opts.SecretProvider == nil {
		opts.SecretProvider = oidc.NewOidcSecretProvider(discovery.NewClient(discovery.Options{opts.Authority}))
	}

	return func(endpoint abstraction.Endpoint, loggerFactory log.Factory) func(http.Handler) http.Handler {
		cfg := AuthorizationEndpointOptions{}
		if fl, ok := endpoint.Filters[AuthorizationFilterCode]; ok {
			err := mapstructure.Decode(fl, &cfg)
			if err != nil {
				loggerFactory(nil).Error("AuthorizationFilter: Cannot find or decode AuthorizationEndpointOptions for authorization filter", zap.Error(err))
			}
		}
		audience := endpoint.OidcAudience
		if cfg.Audience != "" {
			audience = cfg.Audience
		}
		validator := oidc.NewJWTValidator(jwtRequest.OAuth2Extractor, opts.SecretProvider, audience, opts.Authority)

		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				logger := loggerFactory(request.Context())
				if !endpoint.Secured || cfg.Disabled {
					logger.Debug("AuthorizationFilter skipped")
					next.ServeHTTP(writer, request)
					return
				}

				token, err := validator(request)
				if err != nil {
					logger.Error("AuthorizationFilter: Token is not valid", zap.Error(err))
					UnauthorizedWithHeader(writer, err.Error())
					return
				}

				if len(cfg.AllowedScopes) > 0 || len(cfg.ClaimsRequirement) > 0 {
					if len(cfg.AllowedScopes) > 0 {
						hasScope := checkScopes(cfg.AllowedScopes, token.Claims.(jwt.MapClaims)["scope"].([]interface{}))
						if !hasScope {
							logger.Error("AuthorizationFilter: insufficient scope", zap.String("error", "insufficient scope"))
							InsufficientScope(writer, "insufficient scope", cfg.AllowedScopes)
							return
						}
					}

					if len(cfg.ClaimsRequirement) > 0 {
						hasScope := checkClaimsRequirements(cfg.ClaimsRequirement, token.Claims.(jwt.MapClaims))
						if !hasScope {
							logger.Error("AuthorizationFilter: invalid claim", zap.String("error", "invalid claim"))
							Forbidden(writer, "invalid claim")
							return
						}
					}
				}

				ctx := context.WithValue(request.Context(), abstraction.ContextClaimsKey, token.Claims)
				request = request.WithContext(ctx)
				next.ServeHTTP(writer, request)

			})
		}
	}
}

//UnauthorizedWithHeader adds to the response a WWW-Authenticate header and returns a StatusUnauthorized error
func UnauthorizedWithHeader(writer http.ResponseWriter, err string) {
	writer.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\", error_description=\""+err+"\"")
	http.Error(writer, "", http.StatusUnauthorized)
}

//InsufficientScope adds to the response a WWW-Authenticate header and returns a StatusForbidden error
func InsufficientScope(writer http.ResponseWriter, err string, scopes []string) {
	val := "Bearer error=\"insufficient_scope\", error_description=\"" + err + "\" scope=\"" + strings.Join(scopes, ",") + "\""
	writer.Header().Set("WWW-Authenticate", val)
	Forbidden(writer, "")
}

//Forbidden returns a StatusForbidden error
func Forbidden(writer http.ResponseWriter, err string) {
	http.Error(writer, err, http.StatusForbidden)
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

func checkClaimsRequirements(requiredClaims map[string]string, claims jwt.MapClaims) bool {
	for key, val := range requiredClaims {
		v, ok := claims[key]
		if ok {
			if v != val {
				return false
			}
		} else {
			return false
		}
	}

	return true
}
