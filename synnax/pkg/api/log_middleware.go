package api

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"go.uber.org/zap"
)

func logMiddleware(z *zap.SugaredLogger) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx context.Context,
		md freighter.MD,
		next freighter.Next,
	) error {
		err := next(ctx, md)
		logFunc(z, err)("protocol", md.Protocol, "target", md.Target, "err", err)
		return err
	})
}

func logFunc(z *zap.SugaredLogger, err error) func(args ...interface{}) {
	if err != nil {
		return z.Error
	}
	return z.Info
}

func constructLogArgs(md freighter.MD, err error) []interface{} {
	args := []interface{}{
		"protocol", md.Protocol,
		"target", md.Target,
	}
	if err != nil {
		args = append(args, "err", err)
	}
	return args
}
