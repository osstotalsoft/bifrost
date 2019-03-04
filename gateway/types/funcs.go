package types

import "net/http"

type PreFilterFunc func(request *http.Request) error
type PostFilterFunc func(request, proxyRequest *http.Request, proxyResponse *http.Response) ([]byte, error)
type MiddlewareFunc func(endpoint Endpoint) func(http.Handler) http.Handler
type HandlerFunc func(endpoint Endpoint) http.Handler
