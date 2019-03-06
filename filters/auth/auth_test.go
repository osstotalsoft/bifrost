package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/test"
	"github.com/osstotalsoft/bifrost/gateway"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testEndPoint = gateway.Endpoint{
	Secured: true,
	Filters: map[string]interface{}{
		"auth": AuthorizationEndpointOptions{
			ClaimsRequirement: map[string]string{
				"client_id": "CharismaFinancialServices",
			},
			AllowedScopes: []string{"LSNG.Api.read_only", "Notifier.Api.write"},
		},
	},
}

var intentityConfig = AuthorizationOptions{
	Authority:        "http://kube-worker1:30692",
	Audience:         "LSNG.Api",
	WellKnownJwksUrl: "http://kube-worker1:30692/.well-known/openid-configuration/jwks",
}

var claims = jwt.MapClaims{
	"iss": "http://kube-worker1:30692",
	"aud": []string{
		"http://kube-worker1:30692/resources",
		"LSNG.Api",
		"Notifier.Api",
	},
	"client_id":        "CharismaFinancialServices",
	"sub":              "c8124881-ad67-443e-9473-08d5777d1ba8",
	"idp":              "local",
	"partner_id":       "-100",
	"charisma_user_id": "1",
	"scope": []string{
		"openid",
		"profile",
		"roles",
		"LSNG.Api.read_only",
		"charisma_data",
		"Notifier.Api.write",
	},
	"amr": []string{
		"pwd",
	},
}

func init() {

}

func TestAuthorizationFilter(t *testing.T) {
	// load keys from disk
	privateKey := test.LoadRSAPrivateKeyFromDisk("sample_key")
	publicKey := test.LoadRSAPublicKeyFromDisk("sample_key.pub")
	intentityConfig.PublicKeyGetter = func(*jwt.Token) (interface{}, error) {
		return publicKey, nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, _ := token.SignedString(privateKey)
	log.Info(tokenString)

	filter := AuthorizationFilter(intentityConfig)(testEndPoint)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "OK")
	})

	req := httptest.NewRequest("GET", "/whaterver", nil)
	req.Header.Add("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	filter(handler).ServeHTTP(w, req)

	result := w.Result()

	log.Info(result)
}

func BenchmarkAuthorizationFilter(b *testing.B) {

}
