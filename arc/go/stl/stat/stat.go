// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/stl"
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
			Config: types.Params{
				{Name: durationConfigParam, Type: types.TimeSpan(), Value: telem.TimeSpanZero},
				{Name: countConfigParam, Type: types.I64(), Value: 0},
			},
			Inputs: types.Params{
				{Name: ir.DefaultInputParam, Type: types.Variable("T", &constraint)},
				{Name: resetInputParam, Type: types.U8(), Value: 0},
			},
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)},
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

type Module struct{}

var _ stl.Module = (*Module)(nil)

func NewModule() *Module { return &Module{} }

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	return SymbolResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	return SymbolResolver.Search(ctx, term)
}

func (m *Module) Create(_ context.Context, nodeCfg node.Config) (node.Node, error) {
	reductionMap, ok := ops[nodeCfg.Node.Type]
	if !ok {
		return nil, query.ErrNotFound
	}
	var (
		inputData   = nodeCfg.State.Input(0)
		reductionFn = reductionMap[inputData.DataType]
		resetIdx    = -1
	)
	if _, found := nodeCfg.Module.Edges.FindByTarget(ir.Handle{
		Node:  nodeCfg.Node.Key,
		Param: resetInputParam,
	}); found {
		resetIdx = 1
		nodeCfg.State.InitInput(
			resetIdx,
			telem.NewSeriesV[uint8](0),
			telem.NewSeriesV[telem.TimeStamp](1),
		)
	}
	var cfg ConfigValues
	if err := configSchema.Parse(nodeCfg.Node.Config.ValueMap(), &cfg); err != nil {
		return nil, err
	}
	return &statNode{
		Node:        nodeCfg.State,
		resetIdx:    resetIdx,
		reductionFn: reductionFn,
		sampleCount: 0,
		cfg:         cfg,
	}, nil
}

func (m *Module) BindTo(_ context.Context, _ stl.HostRuntime) error {
	return nil
}

type ConfigValues struct {
	Duration telem.TimeSpan `json:"duration" msgpack:"duration"`
	Count    int64          `json:"count" msgpack:"count"`
}

var configSchema = zyn.Object(map[string]zyn.Schema{
	durationConfigParam: zyn.Int64().Optional().Coerce(),
	countConfigParam:    zyn.Int64().Optional().Coerce(),
})

type statNode struct {
	*state.Node
	reductionFn   func(telem.Series, int64, *telem.Series) int64
	cfg           ConfigValues
	resetIdx      int
	sampleCount   int64
	startTime     telem.TimeStamp
	lastResetTime telem.TimeStamp
}

var _ node.Node = (*statNode)(nil)

func (r *statNode) Reset() {
	r.Node.Reset()
	r.sampleCount = 0
	r.startTime = 0
	r.lastResetTime = 0
}

func (r *statNode) Next(ctx node.Context) {
	if !r.RefreshInputs() {
		return
	}

	inputTime := r.InputTime(0)
	if r.startTime == 0 && inputTime.Len() > 0 {
		r.startTime = telem.ValueAt[telem.TimeStamp](inputTime, 0)
	}

	shouldReset := false

	if r.resetIdx >= 0 {
		resetData := r.Input(r.resetIdx)
		resetTime := r.InputTime(r.resetIdx)
		for i := int64(0); i < resetData.Len(); i++ {
			ts := telem.ValueAt[telem.TimeStamp](resetTime, int(i))
			if ts > r.lastResetTime && telem.ValueAt[uint8](resetData, int(i)) == 1 {
				shouldReset = true
				break
			}
		}
		if resetTime.Len() > 0 {
			r.lastResetTime = telem.ValueAt[telem.TimeStamp](resetTime, -1)
		}
	}

	if r.cfg.Duration > 0 && inputTime.Len() > 0 {
		currentTime := telem.ValueAt[telem.TimeStamp](inputTime, -1)
		if telem.TimeSpan(currentTime-r.startTime) >= r.cfg.Duration {
			shouldReset = true
			r.startTime = currentTime
		}
	}

	if r.cfg.Count > 0 && r.sampleCount >= r.cfg.Count {
		shouldReset = true
	}

	if shouldReset {
		r.sampleCount = 0
		r.Output(0).Resize(0)
		inputTime = r.InputTime(0)
	}
	inputData := r.Input(0)
	if inputData.Len() == 0 {
		return
	}
	r.sampleCount = r.reductionFn(inputData, r.sampleCount, r.Output(0))
	if inputTime.Len() > 0 {
		lastTimestamp := telem.ValueAt[telem.TimeStamp](inputTime, -1)
		*r.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](lastTimestamp)
	}
	alignment := inputData.Alignment
	timeRange := inputData.TimeRange
	if r.resetIdx >= 0 {
		resetData := r.Input(r.resetIdx)
		alignment += resetData.Alignment
		if !resetData.TimeRange.Start.IsZero() && (timeRange.Start.IsZero() || resetData.TimeRange.Start < timeRange.Start) {
			timeRange.Start = resetData.TimeRange.Start
		}
		if resetData.TimeRange.End > timeRange.End {
			timeRange.End = resetData.TimeRange.End
		}
	}
	r.Output(0).Alignment = alignment
	r.Output(0).TimeRange = timeRange
	r.OutputTime(0).Alignment = alignment
	r.OutputTime(0).TimeRange = timeRange
	ctx.MarkChanged(ir.DefaultOutputParam)
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
