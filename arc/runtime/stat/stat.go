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
	"github.com/synnaxlabs/x/zyn"
)

const (
	resetInputParam     = "reset"
	durationConfigParam = "duration"
	countConfigParam    = "count"
	avgSymbolName       = "avg"
	minSymbolName       = "min"
	maxSymbolName       = "max"
)

func createBaseSymbol(name string) symbol.Symbol {
	constraint := types.NumericConstraint()
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: &types.Params{
				Keys:   []string{durationConfigParam, countConfigParam},
				Values: []types.Type{types.TimeSpan(), types.I64()},
			},
			Inputs: &types.Params{
				Keys:   []string{ir.DefaultInputParam, resetInputParam},
				Values: []types.Type{types.Variable("T", &constraint), types.U8()},
			},
			Outputs: &types.Params{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []types.Type{types.Variable("T", &constraint)},
			},
			InputDefaults: map[string]any{resetInputParam: 0},
			ConfigDefaults: map[string]any{
				durationConfigParam: telem.TimeSpan(0),
				countConfigParam:    0,
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

type stat struct {
	cfg           ConfigValues
	state         *state.Node
	resetIdx      int
	reductionFn   func(telem.Series, int64, *telem.Series) int64
	sampleCount   int64
	startTime     telem.TimeStamp
	lastResetTime telem.TimeStamp // Track last processed reset timestamp to avoid re-processing
}

func (r *stat) Init(_ node.Context) {
}

func (r *stat) Next(ctx node.Context) {
	if !r.state.RefreshInputs() {
		return
	}

	// Initialize start time from first input data timestamp
	inputTime := r.state.InputTime(0)
	if r.startTime == 0 && inputTime.Len() > 0 {
		r.startTime = telem.ValueAt[telem.TimeStamp](inputTime, 0)
	}

	shouldReset := false

	// Signal-based reset
	if r.resetIdx >= 0 {
		resetData := r.state.Input(r.resetIdx)
		resetTime := r.state.InputTime(r.resetIdx)
		// Check if any NEW value in the reset series is 1 (catches fast pulses)
		// Only look at values with timestamps > lastResetTime to avoid re-processing
		for i := int64(0); i < resetData.Len(); i++ {
			ts := telem.ValueAt[telem.TimeStamp](resetTime, int(i))
			if ts > r.lastResetTime && telem.ValueAt[uint8](resetData, int(i)) == 1 {
				shouldReset = true
				break
			}
		}
		// Update lastResetTime to the last timestamp in this series
		if resetTime.Len() > 0 {
			r.lastResetTime = telem.ValueAt[telem.TimeStamp](resetTime, -1)
		}
	}

	// Duration-based reset (using input data timestamp)
	if r.cfg.Duration > 0 && inputTime.Len() > 0 {
		currentTime := telem.ValueAt[telem.TimeStamp](inputTime, -1)
		if telem.TimeSpan(currentTime-r.startTime) >= r.cfg.Duration {
			shouldReset = true
			r.startTime = currentTime
		}
	}

	// Count-based reset
	if r.cfg.Count > 0 && r.sampleCount >= r.cfg.Count {
		shouldReset = true
	}

	if shouldReset {
		r.sampleCount = 0
		r.state.Output(0).Resize(0)
		// Re-read input time after reset
		inputTime = r.state.InputTime(0)
	}
	inputData := r.state.Input(0)
	if inputData.Len() == 0 {
		return
	}
	r.sampleCount = r.reductionFn(inputData, r.sampleCount, r.state.Output(0))
	// Set output timestamp to the last (most recent) input timestamp
	// Output has 1 value, so output time must also have 1 timestamp
	if inputTime.Len() > 0 {
		lastTimestamp := telem.ValueAt[telem.TimeStamp](inputTime, -1)
		*r.state.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](lastTimestamp)
	}
	ctx.MarkChanged(ir.DefaultOutputParam)
}

type ConfigValues struct {
	Duration telem.TimeSpan `json:"duration" msgpack:"duration"`
	Count    int64          `json:"count" msgpack:"count"`
}

var configSchema = zyn.Object(map[string]zyn.Schema{
	durationConfigParam: zyn.Int64().Optional().Coerce(),
	countConfigParam:    zyn.Int64().Optional().Coerce(),
})

type statFactory struct {
}

func (f *statFactory) Create(_ context.Context, nodeCfg node.Config) (node.Node, error) {
	reductionMap, ok := ops[nodeCfg.Node.Type]
	if !ok {
		return nil, query.NotFound
	}
	var (
		inputData   = nodeCfg.State.Input(0)
		reductionFn = reductionMap[inputData.DataType]
		resetIdx    = -1
	)
	if _, found := nodeCfg.Module.Edges.FindByTarget(ir.Handle{
		Node:  nodeCfg.Node.Type,
		Param: resetInputParam,
	}); found {
		resetIdx = 1
	}
	var cfg ConfigValues
	if err := configSchema.Parse(nodeCfg.Node.ConfigValues, &cfg); err != nil {
		return nil, err
	}
	return &stat{
		state:       nodeCfg.State,
		resetIdx:    resetIdx,
		reductionFn: reductionFn,
		sampleCount: 0,
		cfg:         cfg,
	}, nil
}

func NewFactory() node.Factory {
	return &statFactory{}
}

var ops = map[string]map[telem.DataType]func(telem.Series, int64, *telem.Series) int64{
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
