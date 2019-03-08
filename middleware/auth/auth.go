package auth

import (
	"context"
	"errors"
	"github.com/dgrijalva/jwt-go"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
	"github.com/mitchellh/mapstructure"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/middleware"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const AuthorizationFilterCode = "auth"

type AuthorizationOptions struct {
	Authority        string `mapstructure:"authority"`
	Audience         string `mapstructure:"audience"`
	WellKnownJwksUrl string `mapstructure:"well_known_jwks"`

	PublicKeyGetter jwt.Keyfunc
	Extractor       jwtRequest.Extractor
}

type AuthorizationEndpointOptions struct {
	ClaimsRequirement map[string]string `mapstructure:"claims_requirement"`
	AllowedScopes     []string          `mapstructure:"allowed_scopes"`
}

func AuthorizationFilter(opts AuthorizationOptions) middleware.Func {
	if opts.Extractor == nil {
		opts.Extractor = jwtRequest.OAuth2Extractor
	}
	if opts.PublicKeyGetter == nil {
		opts.PublicKeyGetter = PublicKeyGetter(opts.WellKnownJwksUrl)
	}

	return func(endpoint abstraction.Endpoint) func(http.Handler) http.Handler {
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

				token, err := validateRequest(request, opts)
				if err != nil {
					log.Errorln("AuthorizationFilter: Token is not valid:", err)
					Unauthorized(writer)
					return
				}

				log.Debugf("AuthorizationFilter: ValidateRequest took %s", time.Since(start))

				if len(cfg.AllowedScopes) > 0 || len(cfg.ClaimsRequirement) > 0 {

					log.Debugf("AuthorizationFilter: Claims decoder took %s", time.Since(start))

					if len(cfg.AllowedScopes) > 0 {
						hasScope := checkScopes(cfg.AllowedScopes, token.Claims.(jwt.MapClaims)["scope"].([]interface{}))
						if !hasScope {
							Forbidden(writer)
							return
						}
						log.Debugf("AuthorizationFilter: CheckScopes took %s", time.Since(start))
					}

					if len(cfg.ClaimsRequirement) > 0 {

					}
				}

				claimsMap := map[string]interface{}(token.Claims.(jwt.MapClaims))
				ctx := context.WithValue(request.Context(), abstraction.ContextClaimsKey, claimsMap)
				request = request.WithContext(ctx)

				next.ServeHTTP(writer, request)
			})
		}
	}
}

func validateRequest(request *http.Request, opts AuthorizationOptions) (*jwt.Token, error) {
	token, err := jwtRequest.ParseFromRequest(request, opts.Extractor, opts.PublicKeyGetter)

	if err != nil {
		return nil, err
	}

	checkAud := verifyAudience(opts.Audience, token.Claims.(jwt.MapClaims)["aud"])
	if !checkAud {
		return token, errors.New("invalid audience")
	}
	// Verify 'iss' claim
	checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(opts.Authority, true)
	if !checkIss {
		return token, errors.New("invalid issuer")
	}
	return token, err
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

func verifyAudience(audience string, tokenAudience interface{}) bool {
	switch tokenAudience.(type) {
	case string:
		return tokenAudience == audience
	case []interface{}:
		{
			for _, aud := range tokenAudience.([]interface{}) {
				if aud == audience {
					return true
				}
			}
		}
	}

	return false
}
