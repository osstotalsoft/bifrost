package router

import "time"

const ContextRouteKey = "ContextRouteKey"

type RouteContext struct {
	Path       string
	PathPrefix string
	Timeout    time.Duration
	Vars       map[string]string
}
