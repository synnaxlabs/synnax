package errors

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
)

func Middleware() freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx context.Context,
		md freighter.MD,
		next freighter.Next,
	) error {
		err := next(ctx, md)
		var t Typed
		if roacherrors.As(err, &t) {
			if !t.Occurred() {
				return nil
			}
		}
		return err
	})
}
