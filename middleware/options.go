package middleware

import (
	"net/http"
)

type Options struct {
	startMiddlewareObserver func(r *http.Request)
	endMiddlewareObserver   func(r *http.Request)
}

type Option func(*Options)

func StartMiddlewareObserver(f func(r *http.Request)) Option {
	return func(options *Options) {
		options.startMiddlewareObserver = f
	}
}

func EndMiddlewareObserver(f func(r *http.Request)) Option {
	return func(options *Options) {
		options.endMiddlewareObserver = f
	}
}
