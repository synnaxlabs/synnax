package alamos

import (
	"go.uber.org/zap"
)

var nopLogger = &Logger{Logger: zap.NewNop()}

type Logger struct {
	*zap.Logger
}

func L(i Instrumentation) *Logger {
	ins, ok := Extract(ctx)
	if !ok {
		return nopLogger
	}
	return ins.L
}

func newDevLogger(key string) *Logger {
	return &Logger{Logger: zap.NewNop()}
}
