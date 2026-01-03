// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics

import (
	"context"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type collector struct {
	ins      alamos.Instrumentation
	interval time.Duration
	idx      channel.Channel
	metrics  []metric
	stop     chan struct{}
	confluence.AbstractUnarySource[framer.WriterRequest]
}

var _ confluence.Source[framer.WriterRequest] = (*collector)(nil)

func (c *collector) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(c.Out)
	sCtx.Go(func(ctx context.Context) error {
		t := time.NewTicker(c.interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c.stop:
				return nil
			case currTime := <-t.C:
				frame := core.UnaryFrame(
					c.idx.Key(),
					telem.NewSeriesV[telem.TimeStamp](telem.NewTimeStamp(currTime)),
				)
				for _, metric := range c.metrics {
					value, err := metric.collect()
					if err != nil {
						c.ins.L.Warn("failed to collect metric from host", zap.Error(err), zap.String("name", metric.ch.Name))
						continue
					}
					frame = frame.Append(metric.ch.Key(), telem.NewSeriesFromAny(value, metric.ch.DataType))
				}
				if err := signal.SendUnderContext(ctx, c.Out.Inlet(), framer.WriterRequest{
					Command: writer.Write,
					Frame:   frame,
				}); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}
