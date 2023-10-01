package writer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

func (s *Service) newFree(ctx context.Context) StreamWriter {
	w := &freeWriter{freeWrites: s.FreeWrites}
	w.Transform = w.transform
	return w
}

type freeWriter struct {
	confluence.LinearTransform[Request, Response]
	freeWrites confluence.Inlet[relay.Response]
	seqNum     int
}

func (w *freeWriter) transform(ctx context.Context, req Request) (res Response, ok bool, err error) {
	if req.Command == Data {
		err = signal.SendUnderContext(ctx, w.freeWrites.Inlet(), relay.Response{Frame: req.Frame})
		return
	}
	w.seqNum++
	return Response{Command: req.Command, Ack: true, SeqNum: w.seqNum}, true, nil
}
