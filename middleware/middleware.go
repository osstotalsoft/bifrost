package middleware

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"net/http"
)

//Func is a signature that each middleware must implement
type Func func(endpoint abstraction.Endpoint) func(http.Handler) http.Handler
