package auth

import (
	rsa2 "crypto/rsa"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
	"github.com/gbrlsnchs/jwt/v3"
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

	PublicKeyGetter func(tokenKeyId string) (*rsa2.PublicKey, error)
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

				claims := &testPayload{}
				res, err := validateRequest(request, opts)
				_ = mapstructure.Decode(res, claims)

				if err != nil {
					log.Errorln("AuthorizationFilter: Token is not valid:", err)
					Unauthorized(writer)
					return
				}

				log.Debugf("AuthorizationFilter: ValidateRequest took %s", time.Since(start))

				if len(cfg.AllowedScopes) > 0 || len(cfg.ClaimsRequirement) > 0 {

					log.Debugf("AuthorizationFilter: Claims decoder took %s", time.Since(start))

					if len(cfg.AllowedScopes) > 0 {
						hasScope := checkScopes(cfg.AllowedScopes, claims.Scope)
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

type testPayload struct {
	jwt.Payload
	UserId   string                 `json:"sub,omitempty"`
	Scope    []string               `json:"scope,omitempty"`
	AllOther map[string]interface{} `json:"-"`
}

func validateRequest(request *http.Request, opts AuthorizationOptions) (claims map[string]interface{}, err error) {

	//token, err := jwtRequest.ParseFromRequest(request, opts.Extractor, opts.PublicKeyGetter)
	claims = &testPayload{}

	// perform extract
	tokenString, err := opts.Extractor.ExtractToken(request)
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse([]byte(tokenString))
	if err != nil {
		return nil, err
	}
	header, err := token.Decode(claims)
	if err != nil {
		return nil, err
	}

	key, err := opts.PublicKeyGetter(header.KeyID)
	if err != nil {
		return nil, err
	}

	rsa := jwt.NewRSA(jwt.SHA256, nil, key)

	err = token.Verify(rsa)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	iatValidator := jwt.IssuedAtValidator(now)
	expValidator := jwt.ExpirationTimeValidator(now, true)
	notBeforeValidator := jwt.NotBeforeValidator(now)
	audValidator := jwt.AudienceValidator(jwt.Audience{opts.Audience})
	issuerValidator := jwt.IssuerValidator(opts.Authority)

	err = claims.Validate(iatValidator, expValidator, audValidator, notBeforeValidator, issuerValidator)
	if err != nil {
		return nil, err
	}

	return claims, err
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
