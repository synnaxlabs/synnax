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
			Config: &types.Params{
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
	snode         *state.Node
	key           uint32
	highWaterMark xtelem.Alignment
}

func (s *source) Init(context.Context, func(output string)) {}

func (s *source) Next(_ context.Context, onOutputChange func(param string)) {
	data, indexData, ok := s.snode.ReadChan(s.key)
	if !ok || len(data.Series) == 0 {
		return
	}
	for i, ser := range data.Series {
		ab := ser.AlignmentBounds()
		if ab.Lower >= s.highWaterMark {
			var timeSeries xtelem.Series
			if indexData.DataType() == xtelem.UnknownT {
				timeSeries = xtelem.Arange[xtelem.TimeStamp](
					xtelem.Now(),
					int(data.Len()),
					1*xtelem.NanosecondTS,
				)
				timeSeries.Alignment = ser.Alignment
			} else if len(indexData.Series) > i {
				timeSeries = indexData.Series[i]
			} else {
				return
			}
			if timeSeries.Alignment != ser.Alignment {
				return
			}
			*s.snode.Output(0) = ser
			*s.snode.OutputTime(0) = timeSeries
			s.highWaterMark = ab.Upper
			onOutputChange(ir.DefaultOutputParam)
			return
		}
	}
}

type sink struct {
	state *state.Node
	key   uint32
}

func (s *sink) Init(context.Context, func(output string)) {}

func (s *sink) Next(context.Context, func(param string)) {
	if !s.state.RefreshInputs() {
		return
	}
	data := s.state.Input(0)
	time := s.state.InputTime(0)
	if data.Len() == 0 {
		return
	}
	s.state.WriteChan(s.key, data, time)
}

type telemFactory struct{}

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
		return &source{snode: cfg.State, key: nodeCfg.Channel, highWaterMark: 0}, nil
	}
	return &sink{state: cfg.State, key: nodeCfg.Channel}, nil
}

func NewTelemFactory() node.Factory {
	return &telemFactory{}
}
