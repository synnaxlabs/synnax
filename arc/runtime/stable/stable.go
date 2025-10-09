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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var (
	symbolName = "stable_for"
	symbol     = ir.Symbol{
		Name: symbolName,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: ir.NamedTypes{
				Keys:   []string{"duration"},
				Values: []ir.Type{ir.TimeSpan{}},
			},
			Params: ir.NamedTypes{
				Keys:   []string{ir.DefaultInputParam},
				Values: []ir.Type{ir.NewTypeVariable("T", nil)},
			},
			Outputs: ir.NamedTypes{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []ir.Type{ir.NewTypeVariable("T", nil)},
			},
		},
	}
	SymbolResolver = ir.MapResolver{symbolName: symbol}
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
	if inputSeries.Len() == 0 {
		return
	}

	// Check all values in the input series
	for _, currentValue := range inputSeries.Data {
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
			outputSeries.Resize(1)
			outputSeries.Data[0] = currentValue
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
	switch v := cfg.Node.Config["duration"].(type) {
	case float64:
		duration = telem.TimeSpan(v)
	case int:
		duration = telem.TimeSpan(v)
	case int64:
		duration = telem.TimeSpan(v)
	default:
		duration = telem.TimeSpan(0)
	}

	inputEdge := cfg.Module.GetEdgeByTargetHandle(ir.Handle{
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
