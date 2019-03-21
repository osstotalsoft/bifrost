package httputils

import "net/http"

//Compose Handlers
func Compose(funcs ...func(handler http.Handler) http.Handler) func(handler http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for _, f := range funcs {
			h = f(h)
		}
		return h
	}
}
