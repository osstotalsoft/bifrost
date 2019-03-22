package middleware

import (
	"net/http"
)

type options struct {
	startMiddlewareObserver func(r *http.Request)
	endMiddlewareObserver   func(r *http.Request)
}

type Option func(*options)

func StartMiddlewareObserver(f func(r *http.Request)) Option {
	return func(options *options) {
		options.startMiddlewareObserver = f
	}
}

func EndMiddlewareObserver(f func(r *http.Request)) Option {
	return func(options *options) {
		options.endMiddlewareObserver = f
	}
}
