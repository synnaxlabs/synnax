package api

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"go.uber.org/zap"
	"time"
)

func logMiddleware(z *zap.SugaredLogger) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx context.Context,
		md freighter.MD,
		next freighter.Next,
	) error {
		t0 := time.Now()
		err := next(ctx, md)
		logFunc(z, err)("api request", "protocol", md.Protocol, "target", md.Target, "requestDur", time.Since(t0), "err", err)
		return err
	})
}

func logFunc(z *zap.SugaredLogger, err error) func(msg string, args ...interface{}) {
	if err != nil {
		return z.Errorw
	}
	return z.Infow
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
