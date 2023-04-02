package alamos

import (
	"context"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

var nopLogger = &Logger{Logger: zap.NewNop()}

type Logger struct {
	*zap.Logger
}

func L(ctx context.Context) *zap.Logger {
	ins, ok := fromContext(ctx)
	return lo.Ternary(ok, ins.logger, nopLogger).Logger
}
