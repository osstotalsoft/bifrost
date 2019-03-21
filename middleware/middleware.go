package middleware

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"net/http"
)

//Func is a signature that each middleware must implement
type Func func(endpoint abstraction.Endpoint) func(http.Handler) http.Handler

//Compose Funcs
func Compose(funcs ...func(f Func) Func) func(f Func) Func {
	return func(m Func) Func {
		for _, f := range funcs {
			m = f(m)
		}
		return m
	}
}
