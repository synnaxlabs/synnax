// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type transform struct {
	confluence.AbstractLinear[framer.StreamerResponse, framer.WriterRequest]
	streamerRequests            confluence.Inlet[framer.StreamerRequest]
	addRequests, removeRequests chan *calculator.Calculator
	base                        *calculator.Group
	onStatusChange              OnStatusChange
}

var _ confluence.Segment[framer.StreamerResponse, framer.WriterRequest] = (*transform)(nil)

func (g *transform) processAdd(ctx context.Context, c *calculator.Calculator) error {
	g.base.Add(c)
	return g.updateStreamer(ctx)
}

func (g *transform) updateStreamer(ctx context.Context) error {
	streamReq := framer.StreamerRequest{Keys: g.base.ReadFrom()}
	return signal.SendUnderContext(ctx, g.streamerRequests.Inlet(), streamReq)
}

func (g *transform) processRemove(ctx context.Context, c *calculator.Calculator) error {
	g.base.Remove(c)
	return g.updateStreamer(ctx)
}

func (g *transform) Flow(sCtx signal.Context, opts ...confluence.Option) {
	opts = append(opts, confluence.DeferErr(g.base.Close))
	o := confluence.NewOptions(opts)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case toAdd := <-g.addRequests:
				if err := g.processAdd(ctx, toAdd); err != nil {
					return err
				}
			case toRemove := <-g.removeRequests:
				if err := g.processRemove(ctx, toRemove); err != nil {
					return err
				}
			case req := <-g.In.Outlet():
				output, changed, err := g.base.Next(ctx, req.Frame)
				if err != nil {
					//g.onStatusChange(ctx, Status{
					//	Key:         c.ch.Key().String(),
					//	Variant:     status.ErrorVariant,
					//	Message:     fmt.Sprintf("calculation for %s failed", c.ch),
					//	Description: err.Error(),
					//
					//})
					if !changed {
						continue
					}
					if err := signal.SendUnderContext(ctx, g.Out.Inlet(), framer.WriterRequest{
						Command: writer.Write,
						Frame:   output,
					}); err != nil {
						return err
					}
				}
			}
		}
	}, o.Signal...)
}
