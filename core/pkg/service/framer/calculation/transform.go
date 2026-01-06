// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

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
	streamerRequests confluence.Inlet[framer.StreamerRequest]
	calculators      calculator.Group
	onStatusChange   OnStatusChange
}

var _ confluence.Segment[framer.StreamerResponse, framer.WriterRequest] = (*transform)(nil)

func (g *transform) Flow(sCtx signal.Context, opts ...confluence.Option) {
	opts = append(opts, confluence.DeferErr(g.calculators.Close))
	o := confluence.NewOptions(opts)
	o.AttachClosables(g.Out)
	writeTo := g.calculators.WriteTo()
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-g.In.Outlet():
				if !ok {
					return nil
				}
				output, changed, statuses := g.calculators.Next(ctx, req.Frame)
				if len(statuses) > 0 {
					g.onStatusChange(ctx, statuses...)
				}
				if !changed {
					continue
				}
				if err := signal.SendUnderContext(ctx, g.Out.Inlet(), framer.WriterRequest{
					Command: writer.Write,
					Frame:   output.KeepKeys(writeTo),
				}); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}
