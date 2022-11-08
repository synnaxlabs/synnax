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
		"[writerClient] - cannot write segment to channelClient %s because it was not specified when opening the writerClient",
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
	var res Response
	if err == nil {
		res = Response{Command: Data, Ack: false}
	} else {
		res = Response{Command: Error, Ack: false, Err: err}
	}
	return signal.SendUnderContext(ctx, t.Out.Inlet(), res)
}

func sendBadAck(ctx context.Context, transient chan<- error) error {
	return signal.SendUnderContext(ctx, transient, nil)
}

func sendErrorAck(ctx context.Context, transient chan<- error, err error) error {
	return signal.SendUnderContext(ctx, transient, err)
}
