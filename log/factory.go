package log

import (
	"context"
	"go.uber.org/zap"
)

type Factory func(ctx context.Context) Logger

func ZapLoggerFactory(logger *zap.Logger) Factory {
	return func(ctx context.Context) Logger {
		return zapLogger{logger}
	}
}
