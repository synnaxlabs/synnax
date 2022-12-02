package writer

import (
	"context"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type bulkhead struct {
	signal    chan bool
	closed    bool
	responses struct {
		confluence.AbstractUnarySource[Response]
		confluence.EmptyFlow
	}
	confluence.AbstractLinear[Request, Request]
}

// Flow implements the confluence.Flow interface.
func (b *bulkhead) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(b.responses.Out, b.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case b.closed = <-b.signal:
			case req, ok := <-b.In.Outlet():
				if !ok {
					return nil
				}
				block := b.closed && (req.Command == Data || req.Command == Commit)
				if block {
					if err := signal.SendUnderContext(
						ctx,
						b.responses.Out.Inlet(),
						Response{Command: req.Command, Ack: false},
					); err != nil {
						return err
					}
				} else {
					if err := signal.SendUnderContext(ctx, b.Out.Inlet(), req); err != nil {
						return err
					}
				}
			}
		}
	}, o.Signal...)
}
