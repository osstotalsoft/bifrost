package handler

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/log"
	"net/http"
)

//ReverseProxyHandlerType is a handler type, used when registering a reverse proxy handler
const ReverseProxyHandlerType = "reverseproxy"

//EventPublisherHandlerType is a handler type, used when registering an event handler
//ex: nats, kafka
const EventPublisherHandlerType = "event"

//Func is a signature that each handler must implement
type Func func(endpoint abstraction.Endpoint, loggerFactory log.Factory) http.Handler

//Compose Funcs
func Compose(funcs ...func(f Func) Func) func(f Func) Func {
	return func(m Func) Func {
		for _, f := range funcs {
			m = f(m)
		}
		return m
	}
}
