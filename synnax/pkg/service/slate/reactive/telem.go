// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package reactive

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type telemSource struct {
	confluence.AbstractUnarySource[spec.Value]
	channel channel.Key
	framer  *framer.Service
}

func newTelemSource(_ context.Context, cfg factoryConfig) (bool, error) {
	if cfg.node.Type != spec.TelemSourceType {
		return false, nil
	}
	chKey, _ := schema.Get[float64](schema.Resource{Data: cfg.node.Data}, "channel")
	source := &telemSource{
		channel: channel.Key(chKey),
		framer:  cfg.Framer,
	}
	plumber.SetSource[spec.Value](cfg.pipeline, address.Address(cfg.node.Key), source)
	return true, nil
}

func (n *telemSource) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	sCtx.Go(func(ctx context.Context) error {
		streamer, err := n.framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: []channel.Key{n.channel},
		})
		if err != nil {
			return err
		}
		_, streamerOut := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer)

		streamer.Flow(sCtx)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case v, ok := <-streamerOut.Outlet():
				if !ok {
					return nil
				}
				s := v.Frame.Get(n.channel)
				n.Out.Inlet() <- spec.Value{
					DataType: string(s.DataType()),
					Value:    s.AtAny(-1),
				}
			}
		}
	}, o.Signal...)
}

type telemSink struct {
	confluence.UnarySink[spec.Value]
	channel channel.Key
	framer  *framer.Service
	inlet   confluence.Inlet[framer.WriterRequest]
	outlet  confluence.Outlet[framer.WriterResponse]
}

func (n *telemSink) sink(ctx context.Context, value spec.Value) error {
	if n.inlet == nil || n.outlet == nil {
		w, err := n.framer.NewStreamWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{n.channel},
		})
		if err != nil {
			return err
		}
		n.inlet, n.outlet = confluence.Attach(w)
		sCtx := signal.Wrap(ctx)
		w.Flow(sCtx)
	}
	dt := telem.DataType(value.DataType)
	data := make([]byte, dt.Density())
	telem.MarshalAnyF(dt)(data, value.Value)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-n.outlet.Outlet():
		return res.Err
	case n.inlet.Inlet() <- framer.WriterRequest{
		Command: writer.Write,
		Frame: core.UnaryFrame(n.channel, telem.Series{
			DataType: dt,
			Data:     data,
		}),
	}:
	}
	return nil
}

func newTelemSink(_ context.Context, cfg factoryConfig) (bool, error) {
	if cfg.node.Type != spec.TelemSinkType {
		return false, nil
	}
	chKey, _ := schema.Get[float64](schema.Resource{Data: cfg.node.Data}, "channel")
	source := &telemSink{
		channel: channel.Key(chKey),
		framer:  cfg.Framer,
	}
	source.Sink = source.sink
	plumber.SetSink[spec.Value](cfg.pipeline, address.Address(cfg.node.Key), source)
	return true, nil
}
