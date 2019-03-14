package config

//Config is an object loaded from config.json
type Config struct {
	Endpoints              []EndpointConfig `mapstructure:"endpoints"`
	Port                   int              `mapstructure:"port"`
	Version                string           `mapstructure:"version"`
	Name                   string           `mapstructure:"name"`
	UpstreamPathPrefix     string           `mapstructure:"upstream_path_prefix"`
	DownstreamPathPrefix   string           `mapstructure:"downstream_path_prefix"`
	LogLevel               string           `mapstructure:"log_level"`
	InCluster              bool             `mapstructure:"in_cluster"`
	OverrideServiceAddress string           `mapstructure:"override_service_address"`
}

//EndpointConfig is a configuration detail from config.json
type EndpointConfig struct {
	UpstreamPath         string                 `mapstructure:"upstream_path"`
	UpstreamPathPrefix   string                 `mapstructure:"upstream_path_prefix"`
	DownstreamPath       string                 `mapstructure:"downstream_path"`
	DownstreamPathPrefix string                 `mapstructure:"downstream_path_prefix"`
	ServiceName          string                 `mapstructure:"service_name"`
	Methods              []string               `mapstructure:"methods"`
	HandlerType          string                 `mapstructure:"handler_type"`
	HandlerConfig        map[string]interface{} `mapstructure:"handler_config"`
	Filters              map[string]interface{} `mapstructure:"filters"`
}
