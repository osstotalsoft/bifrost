package nats

import (
	"context"
	"github.com/osstotalsoft/bifrost/log"
	"go.uber.org/zap"
)

type Option func(Config) Config

//NoTransformation is a no op function
func NoTransformation(messageContext messageContext, requestContext context.Context, payloadBytes []byte) (bytes []byte, e error) {
	return payloadBytes, nil
}

//EmptyResponse returns a empty byte[]
func EmptyResponse(messageContext messageContext, requestContext context.Context) (bytes []byte, e error) {
	return nil, nil
}

//TransformMessage adds a TransformMessageFunc to config
func TransformMessage(f TransformMessageFunc) Option {
	return func(config Config) Config {
		config.transformMessageFunc = f
		return config
	}
}

//BuildResponse adds a BuildResponseFunc to config
func BuildResponse(f BuildResponseFunc) Option {
	return func(config Config) Config {
		config.buildResponseFunc = f
		return config
	}
}

//Logger adds a logger to config
func Logger(logger log.Logger) Option {
	return func(config Config) Config {
		config.logger = logger.With(zap.String("handler", "nats"))
		return config
	}
}

func applyOptions(config Config, opts []Option) Config {
	for _, opt := range opts {
		config = opt(config)
	}

	return config
}
