// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stable

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	symbol2 "github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var (
	symbolName = "stable_for"
	symbol     = symbol2.Symbol{
		Name: symbolName,
		Kind: symbol2.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: &types.Params{
				Keys:   []string{"duration"},
				Values: []types.Type{types.TimeSpan()},
			},
			Inputs: &types.Params{
				Keys:   []string{ir.DefaultInputParam},
				Values: []types.Type{types.NewTypeVariable("T", nil)},
			},
			Outputs: &types.Params{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []types.Type{types.NewTypeVariable("T", nil)},
			},
		}),
	}
	SymbolResolver = symbol2.MapResolver{symbolName: symbol}
)

type stableFor struct {
	state       *state.State
	input       ir.Edge
	output      ir.Handle
	duration    telem.TimeSpan
	value       *uint8
	lastSent    *uint8
	lastChanged telem.TimeStamp
	now         func() telem.TimeStamp
}

func (s *stableFor) Init(context.Context, func(string)) {}

func (s *stableFor) Next(ctx context.Context, onOutput func(string)) {
	inputSeries := s.state.Outputs[s.input.Source]
	if inputSeries.Data.Len() == 0 {
		return
	}

	// Check all values in the input series
	for _, currentValue := range inputSeries.Data.Data {
		// Check if value has changed
		if s.value == nil || *s.value != currentValue {
			s.value = &currentValue
			s.lastChanged = s.now()
		}
	}

	if s.value == nil {
		return
	}
	// After processing all values, check if the current value has been stable for the
	// duration
	currentValue := *s.value
	if telem.TimeSpan(s.now()-s.lastChanged) >= s.duration {
		if s.lastSent == nil || *s.lastSent != currentValue {
			// Output the stable value
			outputSeries := s.state.Outputs[s.output]
			outputSeries.Data.Resize(1)
			outputSeries.Data.Data[0] = currentValue
			outputSeries.Time.Resize(1)
			now := s.now()
			marshalF := telem.MarshalF[telem.TimeStamp](telem.TimeStampT)
			marshalF(outputSeries.Time.Data[0:8], now)
			s.state.Outputs[s.output] = outputSeries

			s.lastSent = &currentValue
			onOutput(ir.DefaultOutputParam)
		}
	}
}

type stableFactory struct {
	cfg FactoryConfig
}

type FactoryConfig struct {
	Now func() telem.TimeStamp
}

func NewFactory(cfg FactoryConfig) node.Factory {
	return &stableFactory{cfg: cfg}
}

func (f *stableFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}

	var duration telem.TimeSpan
	switch v := cfg.Node.ConfigValues["duration"].(type) {
	case float64:
		duration = telem.TimeSpan(v)
	case int:
		duration = telem.TimeSpan(v)
	case int64:
		duration = telem.TimeSpan(v)
	default:
		duration = telem.TimeSpan(0)
	}

	inputEdge := cfg.Module.Edges.GetByTarget(ir.Handle{
		Node:  cfg.Node.Key,
		Param: ir.DefaultInputParam,
	})
	outputHandle := ir.Handle{Node: cfg.Node.Key, Param: ir.DefaultOutputParam}

	now := f.cfg.Now
	if now == nil {
		now = telem.Now
	}

	return &stableFor{
		state:    cfg.State,
		input:    inputEdge,
		output:   outputHandle,
		duration: duration,
		now:      now,
	}, nil
}
