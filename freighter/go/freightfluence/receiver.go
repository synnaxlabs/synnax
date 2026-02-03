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
				return rErr
			}
			if err := signal.SendUnderContext(ctx, r.Out.Inlet(), msg); err != nil {
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
			tRes, ok, err := r.Transform(ctx, res)
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
