package reverseproxy

import (
	"context"
	"errors"
	"github.com/osstotalsoft/bifrost/abstraction"
	"net/http"
)

//ClearCorsHeaders deletes cors headers from upstream service
func ClearCorsHeaders(response *http.Response) error {
	//hack when upstream service has cors enabled; cors will be handled by the gateway
	response.Header.Del("Access-Control-Allow-Origin")
	response.Header.Del("Access-Control-Allow-Credentials")
	response.Header.Del("Access-Control-Allow-Methods")
	response.Header.Del("Access-Control-Allow-Headers")
	return nil
}

//AddUserIdToHeader puts userId claim to request header
func AddUserIdToHeader(req *http.Request) error {
	claims, err := getClaims(req.Context())
	if err == nil {
		if sub, ok := claims["sub"]; ok {
			req.Header.Add(abstraction.HttpUserIdHeader, sub.(string))
		}
	}
	return nil
}

//getClaims get the claims map stored in the context
func getClaims(context context.Context) (map[string]interface{}, error) {
	claims, ok := context.Value(abstraction.ContextClaimsKey).(map[string]interface{})
	if !ok {
		return nil, errors.New("claims not present or not authenticated")
	}

	return claims, nil
}
