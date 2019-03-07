package handler

import (
	"github.com/osstotalsoft/bifrost/abstraction"
	"net/http"
)

const ReverseProxyHandlerType = "reverseproxy"
const EventPublisherHandlerType = "event"

type Func func(endpoint abstraction.Endpoint) http.Handler
