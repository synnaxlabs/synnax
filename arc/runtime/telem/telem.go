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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	xtelem "github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
)

var (
	sourceSymbolName = "on"
	sourceSymbol     = symbol.Symbol{
		Name: sourceSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: &types.Params{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []types.Type{types.NewTypeVariable("T", nil)},
			},
			Config: &types.Params{
				Keys:   []string{"channel"},
				Values: []types.Type{types.Chan(types.NewTypeVariable("T", nil))},
			},
		}),
	}
	sinkSymbolName = "write"
	sinkSymbol     = symbol.Symbol{
		Name: sinkSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: &types.Params{
				Keys:   []string{ir.DefaultInputParam},
				Values: []types.Type{types.NewTypeVariable("T", nil)},
			},
			Outputs: &types.Params{
				Keys:   []string{"channel"},
				Values: []types.Type{types.Chan(types.NewTypeVariable("T", nil))},
			},
		}),
	}
	SymbolResolver = symbol.MapResolver{
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
	indexData := s.telem.Data[entry.IndexKey]
	if len(entry.Series) == 0 {
		return
	}
	for i, ser := range entry.Series {
		ab := ser.AlignmentBounds()
		if ab.Upper > s.highWaterMark {
			output := state.Output{Data: ser}
			if entry.IndexKey == 0 {
				output.Time = xtelem.NewSeriesV[xtelem.TimeStamp](xtelem.Now())
				output.Time.Alignment = ser.Alignment
			} else if len(indexData.Series) > i {
				output.Time = s.telem.Data[entry.IndexKey].Series[i]
			} else {
				return
			}

			s.state.Outputs[ir.Handle{Param: ir.DefaultOutputParam, Node: s.node.Key}] = output
			if output.Time.Alignment != ser.Alignment {
				return
			}
			s.highWaterMark = ab.Upper
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

var schema = zyn.Object(map[string]zyn.Schema{
	"channel": zyn.Uint32().Coerce(),
})

type config struct {
	Channel uint32
}

func (t telemFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	isSource := cfg.Node.Type == sourceSymbolName
	isSink := cfg.Node.Type == sinkSymbolName
	if !isSource && !isSink {
		return nil, query.NotFound
	}
	var nodeCfg config
	if err := schema.Parse(cfg.Node.ConfigValues, &nodeCfg); err != nil {
		return nil, err
	}
	if isSource {
		t.telem.registerReader(nodeCfg.Channel, cfg.Node.Key)
		return &source{
			node:          cfg.Node,
			telem:         t.telem,
			state:         cfg.State,
			key:           nodeCfg.Channel,
			highWaterMark: 0,
		}, nil
	}
	t.telem.registerWriter(nodeCfg.Channel, cfg.Node.Key)
	inputEdge := cfg.Module.Edges.GetByTarget(ir.Handle{
		Node:  cfg.Node.Key,
		Param: ir.DefaultInputParam,
	})
	return &sink{
		node:  cfg.Node,
		telem: t.telem,
		state: cfg.State,
		input: inputEdge,
		key:   nodeCfg.Channel,
	}, nil
}

func NewTelemFactory(state *State) node.Factory {
	return &telemFactory{telem: state}
}
