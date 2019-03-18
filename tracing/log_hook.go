package tracing

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/rs/zerolog"
)

func LogHookForTracing(ctx context.Context) zerolog.HookFunc {
	return func(e *zerolog.Event, level zerolog.Level, message string) {
		span := opentracing.SpanFromContext(ctx)
		if span == nil {
			return
		}
		span.LogFields(log.String("error", message))
	}
}
