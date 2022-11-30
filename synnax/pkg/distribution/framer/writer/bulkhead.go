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

func newBulkhead() *bulkhead { return &bulkhead{} }

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
				block, err := b.gate(ctx, req)
				if err != nil {
					return err
				}
				if !block {
					if err := signal.SendUnderContext(ctx, b.Out.Inlet(), req); err != nil {
						return err
					}
				}
			}
		}
	}, o.Signal...)
}

func (b *bulkhead) gate(ctx context.Context, r Request) (bool, error) {
	shouldBlock := b.closed && (r.Command == Data || r.Command == Commit)
	var err error
	if shouldBlock {
		err = signal.SendUnderContext(ctx, b.responses.Out.Inlet(), Response{Command: r.Command, Ack: false})
	}
	return shouldBlock, err
}
