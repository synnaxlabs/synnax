// Copyright 2025 Synnax Labs, Inc.
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
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

// Receiver wraps freighter.StreamReceiver to provide a confluence compatible interface
// for receiving messages from a network freighter.
type Receiver[M freighter.Payload] struct {
	Receiver freighter.StreamReceiver[M]
	AbstractUnarySource[M]
}

var _ Flow = (*Receiver[any])(nil)

// Flow implements Flow.
func (r *Receiver[M]) Flow(ctx signal.Context, opts ...Option) {
	fo := NewOptions(opts)
	fo.AttachClosables(r.Out)
	ctx.Go(r.receive, fo.Signal...)
}

func (r *Receiver[M]) receive(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, rErr := r.Receiver.Receive()
			if errors.Is(rErr, freighter.ErrEOF) {
				return nil
			}
			if rErr != nil {
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
	Transform TransformFunc[M, I]
}

var _ Flow = (*TransformReceiver[any, any])(nil)

// Flow implements Flow.
func (tr *TransformReceiver[I, M]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(tr.Out)
	ctx.Go(tr.receive, o.Signal...)
}

func (tr *TransformReceiver[I, M]) receive(ctx context.Context) error {
o:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			res, err := tr.Receiver.Receive()
			if errors.Is(err, freighter.ErrEOF) {
				return nil
			}
			if err != nil {
				return err
			}
			tRes, ok, err := tr.Transform(ctx, res)
			if !ok {
				continue o
			}
			if err != nil {
				return err
			}
			if err := signal.SendUnderContext(ctx, tr.Out.Inlet(), tRes); err != nil {
				return err
			}
		}
	}
}

type FilterReceiver[I freighter.Payload] struct {
	Receiver freighter.StreamReceiver[I]
	AbstractUnarySource[I]
	Filter  FilterFunc[I]
	Rejects Inlet[I]
}

var _ Flow = (*FilterReceiver[any])(nil)

func (fr *FilterReceiver[I]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(fr.Out, fr.Rejects)
	ctx.Go(fr.receive, o.Signal...)
}

func (fr *FilterReceiver[I]) OutTo(inlets ...Inlet[I]) {
	if len(inlets) > 2 || len(inlets) == 0 {
		panic("[confluence.ApplySink] - provide at most two and at least one inlet")
	}

	if len(inlets) == 1 {
		if fr.AbstractUnarySource.Out != nil {
			fr.Rejects = inlets[0]
			return
		}
	}

	fr.AbstractUnarySource.OutTo(inlets[0])
	if len(inlets) == 2 {
		fr.Rejects = inlets[1]
	}
}

func (fr *FilterReceiver[I]) receive(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			res, err := fr.Receiver.Receive()
			if errors.Is(err, freighter.ErrEOF) {
				return nil
			}
			if err != nil {
				return err
			}
			if err := fr.filter(ctx, res); err != nil {
				return err
			}
		}
	}
}

func (fr *FilterReceiver[I]) filter(ctx context.Context, res I) error {
	ok, err := fr.Filter(ctx, res)
	if err != nil {
		return err
	}
	if ok {
		return signal.SendUnderContext(ctx, fr.Out.Inlet(), res)
	} else if fr.Rejects != nil {
		return signal.SendUnderContext(ctx, fr.Rejects.Inlet(), res)
	}
	return nil
}
