package writer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

func (s *Service) newFree(ctx context.Context) StreamWriter {
	return &freeWriter{freeWrites: s.FreeWrites}
}

type freeWriter struct {
	confluence.AbstractLinear[Request, Response]
	freeWrites confluence.Inlet[relay.Response]
}

func (w *freeWriter) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(w.Out)
	signal.GoRange(ctx, w.In.Outlet(), func(ctx context.Context, r Request) error {
		return signal.SendUnderContext(ctx, w.freeWrites.Inlet(), relay.Response{Frame: r.Frame})
	})
}
