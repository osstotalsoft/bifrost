package httputils

import (
	"net/http"
	"net/url"
)

func CloneRequest(r *http.Request) *http.Request {
	if r == nil {
		panic("nil Request")
	}

	r2 := new(http.Request)
	*r2 = *r

	// Deep copy the URL because it isn't
	// a map and the URL is mutable by users
	// of WithContext.
	if r.URL != nil {
		r2URL := new(url.URL)
		*r2URL = *r.URL
		r2.URL = r2URL
	}

	r2.Header = CloneHeader(r.Header)

	return r2
}

func CloneHeader(h http.Header) http.Header {
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}
