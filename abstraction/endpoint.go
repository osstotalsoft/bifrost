package abstraction

//Endpoint stores the gateway configuration for each routing and is passed around to all handlers and middleware

type Endpoint struct {
	UpstreamPath         string
	Secured              bool
	UpstreamPathPrefix   string
	UpstreamURL          string
	DownstreamPath       string
	DownstreamPathPrefix string
	Methods              []string
	HandlerType          string
	HandlerConfig        map[string]interface{}
	Filters              map[string]interface{}
}
