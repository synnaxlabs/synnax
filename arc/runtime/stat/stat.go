// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stat

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
)

const (
	resetParam    = "reset"
	durationParam = "duration"
	countParam    = "count"
	avgSymbolName = "avg"
	minSymbolName = "min"
	maxSymbolName = "max"
)

func createBaseSymbol(name string) symbol.Symbol {
	constraint := types.NumericConstraint()
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: &types.Params{
				Keys:   []string{durationParam, countParam},
				Values: []types.Type{types.TimeSpan(), types.I64()},
			},
			Inputs: &types.Params{
				Keys:   []string{ir.DefaultInputParam, resetParam},
				Values: []types.Type{types.NewTypeVariable("T", &constraint), types.U8()},
			},
			Outputs: &types.Params{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []types.Type{types.NewTypeVariable("T", &constraint)},
			},
		}),
	}
}

var (
	avgSymbol      = createBaseSymbol(avgSymbolName)
	minSymbol      = createBaseSymbol(minSymbolName)
	maxSymbol      = createBaseSymbol(maxSymbolName)
	SymbolResolver = symbol.MapResolver{
		avgSymbolName: avgSymbol,
		minSymbolName: minSymbol,
		maxSymbolName: maxSymbol,
	}
)

type reduction struct {
	state       *state.State
	output      ir.Handle
	input       ir.Edge
	reset       *ir.Edge
	reductionFn func(telem.Series, int64, *telem.Series) int64
	sampleCount int64
	duration    telem.TimeSpan
	resetCount  int64
	startTime   telem.TimeStamp
	now         func() telem.TimeStamp
}

func (r *reduction) Init(_ context.Context, _ func(output string)) {
	r.startTime = r.now()
}

func (r *reduction) Next(_ context.Context, onOutputChange func(output string)) {
	shouldReset := false
	if r.reset != nil {
		resetOutput := r.state.Outputs[r.reset.Source]
		if resetOutput.Data.Len() > 0 {
			resetValue := telem.ValueAt[uint8](resetOutput.Data, -1)
			if resetValue == 1 {
				shouldReset = true
			}
		}
	}

	if r.duration > 0 {
		currentTime := r.now()
		if telem.TimeSpan(currentTime-r.startTime) >= r.duration {
			shouldReset = true
			r.startTime = currentTime
		}
	}

	if r.resetCount > 0 && r.sampleCount >= r.resetCount {
		shouldReset = true
	}

	if shouldReset {
		r.sampleCount = 0
	}

	inputOutput := r.state.Outputs[r.input.Source]
	if inputOutput.Data.Len() == 0 {
		return
	}

	outputState := r.state.Outputs[r.output]
	r.sampleCount = r.reductionFn(inputOutput.Data, r.sampleCount, &outputState.Data)
	// Copy time from input to output
	outputState.Time = inputOutput.Time
	r.state.Outputs[r.output] = outputState
	onOutputChange(ir.DefaultOutputParam)
}

type Config struct {
	Now func() telem.TimeStamp
}

type reductionFactory struct {
	cfg Config
}

type NodeConfig = node.Config

func (f *reductionFactory) Create(_ context.Context, cfg NodeConfig) (node.Node, error) {
	reductionMap, ok := reductions[cfg.Node.Type]
	if !ok {
		return nil, query.NotFound
	}

	inputEdge := cfg.Module.IR.Edges.GetByTarget(ir.Handle{Node: cfg.Node.Key, Param: ir.DefaultInputParam})
	outputHandle := ir.Handle{Node: cfg.Node.Key, Param: ir.DefaultOutputParam}
	inputOutput := cfg.State.Outputs[inputEdge.Source]
	reductionFn := reductionMap[inputOutput.Data.DataType]

	// Optional reset signal
	var resetEdge *ir.Edge
	if resetEdgeVal, found := cfg.Module.IR.Edges.FindByTarget(ir.Handle{Node: cfg.Node.Key, Param: resetParam}); found {
		resetEdge = &resetEdgeVal
	}

	// Optional duration (default 0 means no duration-based reset)
	var duration telem.TimeSpan
	if durationVal, ok := cfg.Node.ConfigValues[durationParam]; ok {
		duration = durationVal.(telem.TimeSpan)
	}

	// Optional count (default 0 means no count-based reset)
	var resetCount int64
	if countVal, ok := cfg.Node.ConfigValues[countParam]; ok {
		resetCount = countVal.(int64)
	}

	// Use configured time source or default to telem.Now
	nowFn := f.cfg.Now
	if nowFn == nil {
		nowFn = telem.Now
	}

	return &reduction{
		state:       cfg.State,
		output:      outputHandle,
		input:       inputEdge,
		reset:       resetEdge,
		reductionFn: reductionFn,
		sampleCount: 0,
		duration:    duration,
		resetCount:  resetCount,
		now:         nowFn,
	}, nil
}

func NewFactory(cfg Config) node.Factory {
	return &reductionFactory{cfg: cfg}
}

var reductions = map[string]map[telem.DataType]func(telem.Series, int64, *telem.Series) int64{
	avgSymbolName: {
		telem.Float64T: op.AvgF64,
		telem.Float32T: op.AvgF32,
		telem.Int64T:   op.AvgI64,
		telem.Int32T:   op.AvgI32,
		telem.Int16T:   op.AvgI16,
		telem.Int8T:    op.AvgI8,
		telem.Uint64T:  op.AvgU64,
		telem.Uint32T:  op.AvgU32,
		telem.Uint16T:  op.AvgU16,
		telem.Uint8T:   op.AvgU8,
	},
	minSymbolName: {
		telem.Float64T: op.MinF64,
		telem.Float32T: op.MinF32,
		telem.Int64T:   op.MinI64,
		telem.Int32T:   op.MinI32,
		telem.Int16T:   op.MinI16,
		telem.Int8T:    op.MinI8,
		telem.Uint64T:  op.MinU64,
		telem.Uint32T:  op.MinU32,
		telem.Uint16T:  op.MinU16,
		telem.Uint8T:   op.MinU8,
	},
	maxSymbolName: {
		telem.Float64T: op.MaxF64,
		telem.Float32T: op.MaxF32,
		telem.Int64T:   op.MaxI64,
		telem.Int32T:   op.MaxI32,
		telem.Int16T:   op.MaxI16,
		telem.Int8T:    op.MaxI8,
		telem.Uint64T:  op.MaxU64,
		telem.Uint32T:  op.MaxU32,
		telem.Uint16T:  op.MaxU16,
		telem.Uint8T:   op.MaxU8,
	},
}
