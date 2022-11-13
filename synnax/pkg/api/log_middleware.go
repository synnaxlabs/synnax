package api

import (
	"context"
	"time"

	"github.com/synnaxlabs/freighter"
	"go.uber.org/zap"
)

func logMiddleware(z *zap.SugaredLogger) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx context.Context,
		md freighter.MD,
		next freighter.Next,
	) (freighter.MD, error) {
		t0 := time.Now()
		oMd, err := next(ctx, md)
		logFunc(z, err)(
			"api request",
			"protocol", md.Protocol,
			"target", md.Target,
			"duration", time.Since(t0),
			"err",
			err,
		)
		return oMd, err
	})
}

func logFunc(z *zap.SugaredLogger, err error) func(msg string, args ...interface{}) {
	if err != nil {
		return z.Errorw
	}
	return z.Infow
}
