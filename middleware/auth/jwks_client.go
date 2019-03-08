package auth

import (
	rsa2 "crypto/rsa"
	"encoding/json"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"net/http"
)

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

var rsa *rsa2.PublicKey

func PublicKeyGetter(wellKnownJwksUrl string) func(tokenKeyId string) (*rsa2.PublicKey, error) {
	return func(tokenKeyId string) (*rsa2.PublicKey, error) {

		if tokenKeyId == "" {
			return nil, errors.New("KeyId header not found in token")
		}

		if rsa != nil {
			return rsa, nil
		}

		cert := ""
		resp, err := http.Get(wellKnownJwksUrl)

		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var jwks = Jwks{}
		err = json.NewDecoder(resp.Body).Decode(&jwks)

		if err != nil {
			return nil, err
		}

		for k := range jwks.Keys {
			if tokenKeyId == jwks.Keys[k].Kid {
				cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
			}
		}

		if cert == "" {
			err := errors.New("unable to find appropriate key")
			return nil, err
		}

		rsa, err = jwt.ParseRSAPublicKeyFromPEM([]byte(cert))

		return rsa, err
	}
}
