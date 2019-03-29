package tracing

import (
	"context"
	"github.com/opentracing/opentracing-go"
	tag "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/osstotalsoft/bifrost/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

//SpanLoggerFactory is a wrapper over zap.Logger with OpenTracing
func SpanLoggerFactory(logger *zap.Logger) log.Factory {
	return func(ctx context.Context) log.Logger {
		if ctx != nil {
			if span := opentracing.SpanFromContext(ctx); span != nil {
				// TODO for Jaeger span extract trace/span IDs as fields
				return spanLogger{logger, span}
			}
		}
		return log.ZapLoggerFactory(logger)(nil)
	}
}

type spanLogger struct {
	logger *zap.Logger
	span   opentracing.Span
}

func (sl spanLogger) Info(msg string, fields ...zapcore.Field) {
	sl.logToSpan("info", msg, fields...)
	sl.logger.Info(msg, fields...)
}

func (sl spanLogger) Error(msg string, fields ...zapcore.Field) {
	sl.logToSpan("error", msg, fields...)
	//tag.Error.Set(sl.span, true)
	sl.logger.Error(msg, fields...)
}

func (sl spanLogger) Fatal(msg string, fields ...zapcore.Field) {
	sl.logToSpan("fatal", msg, fields...)
	tag.Error.Set(sl.span, true)
	sl.logger.Fatal(msg, fields...)
}

// Debug logs a fatal error msg with fields
func (sl spanLogger) Debug(msg string, fields ...zapcore.Field) {
	sl.logToSpan("debug", msg, fields...)
	sl.logger.Debug(msg, fields...)
}

// Warn logs a fatal error msg with fields
func (sl spanLogger) Warn(msg string, fields ...zapcore.Field) {
	sl.logToSpan("warn", msg, fields...)
	sl.logger.Warn(msg, fields...)
}

// Panic logs a fatal error msg with fields
func (sl spanLogger) Panic(msg string, fields ...zapcore.Field) {
	sl.logToSpan("panic", msg, fields...)
	tag.Error.Set(sl.span, true)
	sl.logger.Panic(msg, fields...)
}

// With creates a child logger, and optionally adds some context fields to that logger.
func (sl spanLogger) With(fields ...zapcore.Field) log.Logger {
	return spanLogger{logger: sl.logger.With(fields...)}
}

func (sl spanLogger) logToSpan(level string, msg string, fields ...zapcore.Field) {
	// TODO rather than always converting the fields, we could wrap them into a lazy logger
	fa := fieldAdapter(make([]otlog.Field, 0, 2+len(fields)))
	fa = append(fa, otlog.String("event", msg))
	fa = append(fa, otlog.String("level", level))
	for _, field := range fields {
		field.AddTo(&fa)
	}
	sl.span.LogFields(fa...)
}

type fieldAdapter []otlog.Field

func (fa *fieldAdapter) AddBool(key string, value bool) {
	*fa = append(*fa, otlog.Bool(key, value))
}

func (fa *fieldAdapter) AddFloat64(key string, value float64) {
	*fa = append(*fa, otlog.Float64(key, value))
}

func (fa *fieldAdapter) AddFloat32(key string, value float32) {
	*fa = append(*fa, otlog.Float64(key, float64(value)))
}

func (fa *fieldAdapter) AddInt(key string, value int) {
	*fa = append(*fa, otlog.Int(key, value))
}

func (fa *fieldAdapter) AddInt64(key string, value int64) {
	*fa = append(*fa, otlog.Int64(key, value))
}

func (fa *fieldAdapter) AddInt32(key string, value int32) {
	*fa = append(*fa, otlog.Int64(key, int64(value)))
}

func (fa *fieldAdapter) AddInt16(key string, value int16) {
	*fa = append(*fa, otlog.Int64(key, int64(value)))
}

func (fa *fieldAdapter) AddInt8(key string, value int8) {
	*fa = append(*fa, otlog.Int64(key, int64(value)))
}

func (fa *fieldAdapter) AddUint(key string, value uint) {
	*fa = append(*fa, otlog.Uint64(key, uint64(value)))
}

func (fa *fieldAdapter) AddUint64(key string, value uint64) {
	*fa = append(*fa, otlog.Uint64(key, value))
}

func (fa *fieldAdapter) AddUint32(key string, value uint32) {
	*fa = append(*fa, otlog.Uint64(key, uint64(value)))
}

func (fa *fieldAdapter) AddUint16(key string, value uint16) {
	*fa = append(*fa, otlog.Uint64(key, uint64(value)))
}

func (fa *fieldAdapter) AddUint8(key string, value uint8) {
	*fa = append(*fa, otlog.Uint64(key, uint64(value)))
}

func (fa *fieldAdapter) AddUintptr(key string, value uintptr)                        {}
func (fa *fieldAdapter) AddArray(key string, marshaler zapcore.ArrayMarshaler) error { return nil }
func (fa *fieldAdapter) AddComplex128(key string, value complex128)                  {}
func (fa *fieldAdapter) AddComplex64(key string, value complex64)                    {}
func (fa *fieldAdapter) AddObject(key string, value zapcore.ObjectMarshaler) error   { return nil }
func (fa *fieldAdapter) AddReflected(key string, value interface{}) error            { return nil }
func (fa *fieldAdapter) OpenNamespace(key string)                                    {}

func (fa *fieldAdapter) AddDuration(key string, value time.Duration) {
	// TODO inefficient
	*fa = append(*fa, otlog.String(key, value.String()))
}

func (fa *fieldAdapter) AddTime(key string, value time.Time) {
	// TODO inefficient
	*fa = append(*fa, otlog.String(key, value.String()))
}

func (fa *fieldAdapter) AddBinary(key string, value []byte) {
	*fa = append(*fa, otlog.Object(key, value))
}

func (fa *fieldAdapter) AddByteString(key string, value []byte) {
	*fa = append(*fa, otlog.Object(key, value))
}

func (fa *fieldAdapter) AddString(key, value string) {
	if key != "" && value != "" {
		*fa = append(*fa, otlog.String(key, value))
	}
}
