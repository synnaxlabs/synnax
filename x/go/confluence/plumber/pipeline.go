// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package plumber

import (
	"go/types"

	"github.com/synnaxlabs/x/address"
	cfs "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

type Segment[I, O cfs.Value] struct {
	*Pipeline
	cfs.UnarySink[I]
	cfs.AbstractUnarySource[O]
	RouteInletsTo    []address.Address
	RouteOutletsFrom []address.Address
}

func (s *Segment[I, O]) constructEndpointRoutes() {
	for _, addr := range s.RouteInletsTo {
		sink, err := GetSink[I](s.Pipeline, addr)
		if err != nil {
			panic(err)
		}
		sink.InFrom(s.In)
	}
	for _, addr := range s.RouteOutletsFrom {
		source, err := GetSource[O](s.Pipeline, addr)
		if err != nil {
			panic(err)
		}
		source.OutTo(s.Out)
	}
}

func (s *Segment[I, O]) RouteInletTo(targets ...address.Address) error {
	s.RouteInletsTo = targets
	for _, addr := range s.RouteInletsTo {
		if _, err := GetSink[I](s.Pipeline, addr); err != nil {
			return err
		}
	}
	return nil
}

func (s *Segment[I, O]) RouteOutletFrom(targets ...address.Address) error {
	s.RouteOutletsFrom = targets
	for _, addr := range targets {
		if _, err := GetSource[O](s.Pipeline, addr); err != nil {
			return err
		}
	}
	return nil
}

func (s *Segment[I, O]) Flow(ctx signal.Context, opts ...cfs.Option) {
	s.constructEndpointRoutes()
	s.Pipeline.Flow(ctx, opts...)
}

type Pipeline struct {
	Sources map[address.Address]entry
	Sinks   map[address.Address]entry
}

type entry struct {
	flow    cfs.Flow
	options []cfs.Option
}

func (e entry) Flow(ctx signal.Context, opts ...cfs.Option) {
	e.flow.Flow(ctx, append(e.options, opts...)...)
}

func (p *Pipeline) Flow(ctx signal.Context, opts ...cfs.Option) {
	for addr, e := range p.Sources {
		e.Flow(ctx, append(opts, cfs.WithAddress(addr))...)
	}
	for addr, e := range p.Sinks {
		if _, ok := p.Sources[addr]; !ok {
			e.Flow(ctx, append(opts, cfs.WithAddress(addr))...)
		}
	}
}

func New() *Pipeline {
	return &Pipeline{
		Sources: make(map[address.Address]entry),
		Sinks:   make(map[address.Address]entry),
	}
}

func SetSource[V cfs.Value](
	p *Pipeline,
	addr address.Address,
	source cfs.Source[V],
	opts ...cfs.Option,
) {
	p.Sources[addr] = entry{flow: source, options: opts}
}

func SetSegment[I, O cfs.Value](
	p *Pipeline,
	addr address.Address,
	segment cfs.Segment[I, O],
	opts ...cfs.Option,
) {
	SetSink(p, addr, segment, opts...)
	SetSource(p, addr, segment, opts...)
}

func SetSink[V cfs.Value](
	p *Pipeline,
	addr address.Address,
	sink cfs.Sink[V],
	opts ...cfs.Option,
) {
	p.Sinks[addr] = entry{flow: sink, options: opts}
}

func GetSource[V cfs.Value](p *Pipeline, addr address.Address) (cfs.Source[V], error) {
	rs, ok := p.Sources[addr]
	if !ok {
		return nil, notFound(addr)
	}
	s, ok := rs.flow.(cfs.Source[V])
	if !ok {
		return nil, wrongType[types.Nil, V](addr, rs.flow)
	}
	return s, nil
}

func GetSink[V cfs.Value](p *Pipeline, addr address.Address) (cfs.Sink[V], error) {
	rs, ok := p.Sinks[addr]
	if !ok {
		return nil, notFound(addr)
	}
	s, ok := rs.flow.(cfs.Sink[V])
	if !ok {
		return nil, wrongType[V, types.Nil](addr, rs.flow)
	}
	return s, nil
}

func notFound(addr address.Address) error {
	return errors.Newf(
		"[plumber] - entity (segment, source, sink) at address %s not found",
		addr,
	)
}

func wrongType[I, O cfs.Value](addr address.Address, actual any) error {
	return errors.Newf(
		`[plumber] - Expected entity (segment, source, sink)  at address %s to have
				inlet type %T and outlet type %T, but got entity of type %T`,
		addr,
		*new(I),
		*new(O),
		actual,
	)
}
