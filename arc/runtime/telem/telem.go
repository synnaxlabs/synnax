// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/query"
	xtelem "github.com/synnaxlabs/x/telem"
)

var (
	sourceSymbolName = "on"
	sourceSymbol     = ir.Symbol{
		Name: sourceSymbolName,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: ir.NamedTypes{
				Keys:   []string{"channel"},
				Values: []ir.Type{ir.Chan{ValueType: ir.NewTypeVariable("T", nil)}},
			},
			Outputs: ir.NamedTypes{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []ir.Type{ir.NewTypeVariable("T", nil)},
			},
		},
	}
	sinkSymbolName = "write"
	sinkSymbol     = ir.Symbol{
		Name: sinkSymbolName,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: ir.NamedTypes{
				Keys:   []string{"channel"},
				Values: []ir.Type{ir.Chan{ValueType: ir.NewTypeVariable("T", nil)}},
			},
			Params: ir.NamedTypes{
				Keys:   []string{ir.DefaultInputParam},
				Values: []ir.Type{ir.NewTypeVariable("T", nil)},
			},
		},
	}
	SymbolResolver = ir.MapResolver{
		sourceSymbolName: sourceSymbol,
		sinkSymbolName:   sinkSymbol,
	}
)

type source struct {
	node          ir.Node
	telem         *State
	state         *state.State
	key           uint32
	highWaterMark xtelem.Alignment
}

func (s *source) Init(context.Context, func(output string)) {}

func (s *source) Next(_ context.Context, onOutputChange func(param string)) {
	entry := s.telem.Data[s.key]
	for i, ser := range entry.Series {
		ab := ser.AlignmentBounds()
		if ab.Upper > s.highWaterMark {
			s.highWaterMark = ab.Upper
			s.state.Outputs[ir.Handle{Param: ir.DefaultOutputParam, Node: s.node.Key}] = state.Output{
				Data: ser,
				Time: s.telem.Data[entry.IndexKey].Series[i],
			}
			onOutputChange(ir.DefaultOutputParam)
		}
	}
}

type sink struct {
	node  ir.Node
	telem *State
	state *state.State
	input ir.Edge
	key   uint32
}

func (s *sink) Init(context.Context, func(output string)) {}

func (s *sink) Next(_ context.Context, _ func(param string)) {
	data := s.state.Outputs[s.input.Source]
	s.telem.Writes[s.key] = data.Data
}

type telemFactory struct {
	telem *State
}

func (t telemFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type == sourceSymbolName {
		key := cfg.Node.Channels.Read.Keys()[0]
		t.telem.registerReader(key, cfg.Node.Key)
		return &source{
			node:          cfg.Node,
			telem:         t.telem,
			state:         cfg.State,
			key:           key,
			highWaterMark: 0,
		}, nil
	}
	if cfg.Node.Type == sinkSymbolName {
		key := cfg.Node.Channels.Write.Keys()[0]
		t.telem.registerWriter(key, cfg.Node.Key)
		inputEdge := cfg.Module.GetEdgeByTargetHandle(ir.Handle{
			Node:  cfg.Node.Key,
			Param: ir.DefaultInputParam,
		})
		return &sink{
			node:  cfg.Node,
			telem: t.telem,
			state: cfg.State,
			input: inputEdge,
			key:   key,
		}, nil
	}
	return nil, query.NotFound
}

func NewTelemFactory(state *State) node.Factory {
	return &telemFactory{telem: state}
}
