package middleware

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"net/http"
)

type Func func(endpoint abstraction.Endpoint) func(http.Handler) http.Handler
