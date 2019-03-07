package abstraction

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
