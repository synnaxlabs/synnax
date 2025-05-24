// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package event

import (
	"context"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
)

func Create(ctx context.Context, cfg spec.Config, g spec.Graph) (confluence.Flow, error) {
	p := plumber.New()
	nodeMap := make(map[string]spec.Node, len(g.Nodes))
	for _, n := range g.Nodes {
		if err := create(ctx, p, cfg, n); err != nil {
			return nil, err
		}
		nodeMap[n.Key] = n
	}
	for _, e := range g.Edges {
		sourceF, _ := plumber.GetSource[spec.Value](p, address.Address(e.Source.Node))
		sourceNode := nodeMap[e.Source.Node]
		sinkF, _ := plumber.GetSink[spec.Value](p, address.Address(e.Sink.Node))
		sinkNode := nodeMap[e.Sink.Node]
		stream := confluence.NewStream[spec.Value](1)
		output, _ := sourceNode.Schema.GetOutput(e.Source.Key)
		input, _ := sinkNode.Schema.GetInput(e.Sink.Key)
		stream.SetInletAddress(address.Address(output.Key))
		stream.SetOutletAddress(address.Address(input.Key))
		sourceF.OutTo(stream)
		sinkF.InFrom(stream)
	}
	return p, nil
}

type factory = func(
	ctx context.Context,
	p *plumber.Pipeline,
	cfg spec.Config,
	node spec.Node,
) (bool, error)

func create(ctx context.Context,
	p *plumber.Pipeline,
	cfg spec.Config,
	node spec.Node,
) error {
	for _, f := range factories {
		if ok, err := f(ctx, p, cfg, node); err != nil || ok {
			return err
		}
	}
	return errors.New("could not find node for")
}

var factories = []factory{
	newConstant,
	newComparison,
	newTelemSource,
	newTelemSink,
}

type constant struct {
	confluence.AbstractUnarySource[spec.Value]
	value spec.Value
}

func newConstant(
	_ context.Context,
	p *plumber.Pipeline,
	_ spec.Config,
	node spec.Node,
) (bool, error) {
	if node.Type != "constant" {
		return false, nil
	}
	value := node.Data["value"]
	c := &constant{
		value: spec.Value{
			DataType: string(node.Schema.Data["value"].Type),
			Value:    value,
		},
	}
	plumber.SetSource[spec.Value](p, address.Address(node.Key), c)
	return true, nil
}

func (n *constant) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	sCtx.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case n.Out.Inlet() <- n.value:
		}
		<-ctx.Done()
		return ctx.Err()
	}, o.Signal...)
}

type telemSource struct {
	confluence.AbstractUnarySource[spec.Value]
	channel channel.Key
	framer  *framer.Service
}

func newTelemSource(
	_ context.Context,
	p *plumber.Pipeline,
	cfg spec.Config,
	node spec.Node,
) (bool, error) {
	if node.Type != "telem_source" {
		return false, nil
	}
	chKey, _ := schema.Get[uint32](schema.Resource{Data: node.Data}, "channel")
	source := &telemSource{
		channel: channel.Key(chKey),
		framer:  cfg.Framer,
	}
	plumber.SetSource[spec.Value](p, address.Address(node.Key), source)
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

type comparison struct {
	confluence.MultiSink[spec.Value]
	confluence.AbstractUnarySource[spec.Value]
	x       *spec.Value
	y       *spec.Value
	compare func(a, b spec.Value) bool
}

func newComparison(
	_ context.Context,
	p *plumber.Pipeline,
	_ spec.Config,
	node spec.Node,
) (bool, error) {
	if !strings.HasPrefix(node.Schema.Type, "comparison") {
		return false, nil
	}
	suffix := strings.Split(node.Schema.Type, ".")[1]
	c := &comparison{}
	if suffix == "ge" {
		c.compare = func(a, b spec.Value) bool {
			return false
		}
	}
	plumber.SetSegment[spec.Value, spec.Value](p, address.Address(node.Key), c)
	c.Sink = c.sink
	return true, nil
}

func (n *comparison) sink(ctx context.Context, origin address.Address, value spec.Value) error {
	if origin == "x" {
		n.x = &value
	}
	if origin == "y" {
		n.y = &value
	}
	if n.y == nil || n.x == nil {
		return nil
	}
	res := n.compare(*n.x, *n.y)
	return signal.SendUnderContext(ctx, n.Out.Inlet(), spec.Value{
		DataType: "uint8",
		Value:    types.BoolToUint8(res),
	})
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

func newTelemSink(
	_ context.Context,
	p *plumber.Pipeline,
	cfg spec.Config,
	node spec.Node,
) (bool, error) {
	if node.Type != "telem_sink" {
		return false, nil
	}
	chKey, _ := schema.Get[uint32](schema.Resource{Data: node.Data}, "channel")
	source := &telemSink{
		channel: channel.Key(chKey),
		framer:  cfg.Framer,
	}
	source.Sink = source.sink
	plumber.SetSink[spec.Value](p, address.Address(node.Key), source)
	return true, nil
}
