package router

import (
	"context"
	"time"
)

//ContextRouteKey is a key for storing route information into the request context
const ContextRouteKey = "ContextRouteKey"

//RouteContext is the object that gets stored into the request context
type RouteContext struct {
	Path       string
	PathPrefix string
	Timeout    time.Duration
	Vars       map[string]string
}

func GetRouteContextFromRequestContext(ctx context.Context) (RouteContext, bool) {
	a, b := ctx.Value(ContextRouteKey).(RouteContext)
	return a, b
}
