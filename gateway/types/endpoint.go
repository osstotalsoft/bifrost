package types

type Endpoint struct {
	UpstreamPath         string
	Secured              bool
	UpstreamPathPrefix   string
	UpstreamURL          string
	DownstreamPath       string
	DownstreamPathPrefix string
	Methods              []string
	HandlerType          string
	Topic                string
}
