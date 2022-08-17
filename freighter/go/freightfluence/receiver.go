package freightfluence

import (
	"context"
	"github.com/arya-analytics/freighter"
	. "github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
)

// Receiver wraps freighter.StreamReceiver to provide a confluence compatible
// interface for receiving messages from a network freighter.
type Receiver[M freighter.Payload] struct {
	Receiver freighter.StreamReceiver[M]
	AbstractUnarySource[M]
}

// Flow implements Flow.
func (r *Receiver[M]) Flow(ctx signal.Context, opts ...Option) {
	fo := NewOptions(opts)
	fo.AttachInletCloser(r)
	ctx.Go(r.receive, fo.Signal...)
}

func (r *Receiver[M]) receive(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, rErr := r.Receiver.Receive()
			if errors.Is(rErr, freighter.EOF) {
				return nil
			}
			if rErr != nil {
				if rErr.Error() == "EOF" {
					return nil
				}
				return rErr
			}
			if err := signal.SendUnderContext(ctx, r.Out.Inlet(), msg); err != nil {
				return err
			}
		}
	}
}

type TransformReceiver[I Value, M freighter.Payload] struct {
	Receiver freighter.StreamReceiver[M]
	AbstractUnarySource[I]
	TransformFunc[M, I]
}

// Flow implements Flow.
func (r *TransformReceiver[I, M]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachInletCloser(r)
	ctx.Go(r.receive, o.Signal...)
}

func (r *TransformReceiver[I, M]) receive(ctx context.Context) error {
o:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			res, err := r.Receiver.Receive()
			if errors.Is(err, freighter.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			tRes, ok, err := r.ApplyTransform(ctx, res)
			if !ok {
				continue o
			}
			if err != nil {
				return err
			}
			if err := signal.SendUnderContext(ctx, r.Out.Inlet(), tRes); err != nil {
				return err
			}
		}
	}
}
