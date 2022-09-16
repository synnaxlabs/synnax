package writer

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
)

func unspecifiedChannelError(key channel.Key) error {
	return errors.Wrapf(
		query.NotFound,
		"[writer] - cannot write segment to channel %s because it was not specified when opening the writer",
		key,
	)
}

type TransientSource struct {
	transient confluence.Outlet[error]
	confluence.AbstractUnarySource[Response]
}

func (t *TransientSource) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(t.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err, ok := <-t.transient.Outlet():
				if !ok {
					return nil
				}
				if err := t.sendError(ctx, err); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

func (t *TransientSource) sendError(ctx context.Context, err error) error {
	return signal.SendUnderContext(ctx, t.Out.Inlet(), Response{Err: err})
}
