package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/test"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/osstotalsoft/oidc-jwt-go"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testEndPoint = abstraction.Endpoint{
	Secured: true,
	Filters: map[string]interface{}{
		"auth": AuthorizationEndpointOptions{
			ClaimsRequirement: map[string]string{
				"client_id": "CharismaFinancialServices",
			},
			Audience:      "LSNG.Api",
			AllowedScopes: []string{"LSNG.Api.read_only", "Notifier.Api.write"},
		},
	},
}

var intentityConfig = AuthorizationOptions{
	Authority: "http://kube-worker1:30692",
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

func TestAuthorizationFilter(t *testing.T) {
	privateKey := test.LoadRSAPrivateKeyFromDisk("sample_key")
	publicKey := test.LoadRSAPublicKeyFromDisk("sample_key.pub")
	intentityConfig.SecretProvider = oidc.NewKeyProvider(publicKey)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, _ := token.SignedString(privateKey)

	logger, _ := zap.NewDevelopment()
	filter := AuthorizationFilter(intentityConfig)(testEndPoint, log.ZapLoggerFactory(logger))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "OK")
	})
	req := httptest.NewRequest("GET", "/whatever", nil)
	req.Header.Add("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	filter(handler).ServeHTTP(w, req)
	result := w.Result()

	if result.StatusCode != http.StatusOK {
		t.Error("request failed status: ", result.StatusCode)
	}
}

func BenchmarkAuthorizationFilter(b *testing.B) {
	privateKey := test.LoadRSAPrivateKeyFromDisk("sample_key")
	publicKey := test.LoadRSAPublicKeyFromDisk("sample_key.pub")
	intentityConfig.SecretProvider = oidc.NewKeyProvider(publicKey)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, _ := token.SignedString(privateKey)

	logger, _ := zap.NewDevelopment()
	filter := AuthorizationFilter(intentityConfig)(testEndPoint, log.ZapLoggerFactory(logger))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "OK")
	})
	req := httptest.NewRequest("GET", "/whatever", nil)
	req.Header.Add("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter(handler).ServeHTTP(w, req)
		w.Result()
	}
}
