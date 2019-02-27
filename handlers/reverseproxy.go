package handlers

import (
	"net/http"
	"net/http/httputil"
)

func NewReverseProxy(director func(*http.Request)) http.Handler {

	//https://github.com/golang/go/issues/16012
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	return &httputil.ReverseProxy{Director: director}
}
