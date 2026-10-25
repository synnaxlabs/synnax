// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freightfluence

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

// Receiver wraps freighter.StreamReceiver to provide a confluence compatible
// interface for receiving messages from a network freighter.
type Receiver[M freighter.Payload] struct {
	Receiver freighter.StreamReceiver[M]
	confluence.AbstractUnarySource[M]
}

// Flow implements Flow.
func (r *Receiver[M]) Flow(ctx signal.Context, opts ...confluence.Option) {
	fo := confluence.NewOptions(opts)
	fo.AttachClosables(r.Out)
	ctx.Go(r.receive, fo.Signal...)
}

func (r *Receiver[M]) receive(ctx context.Context) error {
	results := receiveUnderContext(ctx, r.Receiver)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-results:
			if errors.Is(res.err, freighter.EOF) {
				return nil
			}
			if errors.Is(res.err, freighter.ErrStreamClosed) {
				return context.Canceled
			}
			if res.err != nil {
				return res.err
			}
			if err := signal.SendUnderContext(ctx, r.Out.Inlet(), res.msg); err != nil {
				return err
			}
		}
	}
}

type TransformReceiver[I confluence.Value, M freighter.Payload] struct {
	Receiver freighter.StreamReceiver[M]
	confluence.AbstractUnarySource[I]
	Transform confluence.TransformFunc[M, I]
}

// Flow implements Flow.
func (r *TransformReceiver[I, M]) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(r.Out)
	ctx.Go(r.receive, o.Signal...)
}

func (r *TransformReceiver[I, M]) receive(ctx context.Context) error {
	results := receiveUnderContext(ctx, r.Receiver)
o:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-results:
			if errors.Is(res.err, freighter.EOF) {
				return nil
			}
			if errors.Is(res.err, freighter.ErrStreamClosed) {
				return context.Canceled
			}
			if res.err != nil {
				return res.err
			}
			tRes, ok, err := r.Transform(ctx, res.msg)
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

type received[M freighter.Payload] struct {
	msg M
	err error
}

// receiveUnderContext calls r.Receive on its own goroutine and sends each result on the
// returned channel, so a receiver can exit while Receive blocks. The goroutine exits
// after the first error, or when ctx is done and Receive returns.
func receiveUnderContext[M freighter.Payload](
	ctx context.Context,
	r freighter.StreamReceiver[M],
) <-chan received[M] {
	results := make(chan received[M])
	go func() {
		for {
			msg, err := r.Receive()
			select {
			case results <- received[M]{msg: msg, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return results
}
