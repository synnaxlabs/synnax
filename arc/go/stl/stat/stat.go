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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
	"github.com/synnaxlabs/x/zyn"
)

const (
	resetInputParam      = "reset"
	durationConfigParam  = "duration"
	countConfigParam     = "count"
	avgSymbolName        = "avg"
	minSymbolName        = "min"
	maxSymbolName        = "max"
	derivativeSymbolName = "derivative"
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
	avgSymbol        = createBaseSymbol(avgSymbolName)
	minSymbol        = createBaseSymbol(minSymbolName)
	maxSymbol        = createBaseSymbol(maxSymbolName)
	derivativeSymbol = func() symbol.Symbol {
		constraint := types.NumericConstraint()
		return symbol.Symbol{
			Name: derivativeSymbolName,
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{Name: ir.DefaultInputParam, Type: types.Variable("T", &constraint)},
				},
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)},
				},
			}),
		}
	}()
	SymbolResolver = symbol.MapResolver{
		avgSymbolName:        avgSymbol,
		minSymbolName:        minSymbol,
		maxSymbolName:        maxSymbol,
		derivativeSymbolName: derivativeSymbol,
	}
)

type Module struct{}

func (m *Module) Create(_ context.Context, nodeCfg node.Config) (node.Node, error) {
	if nodeCfg.Node.Type == derivativeSymbolName {
		return m.createDerivative(nodeCfg)
	}
	reductionMap, ok := ops[nodeCfg.Node.Type]
	if !ok {
		return nil, query.ErrNotFound
	}
	var (
		inputData   = nodeCfg.State.Input(0)
		reductionFn = reductionMap[inputData.DataType]
		resetIdx    = -1
	)
	if _, found := nodeCfg.Program.Edges.FindByTarget(ir.Handle{
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
		State:       nodeCfg.State,
		resetIdx:    resetIdx,
		reductionFn: reductionFn,
		sampleCount: 0,
		cfg:         cfg,
	}, nil
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
	*node.State
	reductionFn   func(telem.Series, int64, *telem.Series) int64
	cfg           ConfigValues
	resetIdx      int
	sampleCount   int64
	startTime     telem.TimeStamp
	lastResetTime telem.TimeStamp
}

var _ node.Node = (*statNode)(nil)

func (r *statNode) Reset() {
	r.State.Reset()
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

func (m *Module) createDerivative(cfg node.Config) (node.Node, error) {
	inputData := cfg.State.Input(0)
	derivFn, ok := derivOps[inputData.DataType]
	if !ok {
		return nil, query.ErrNotFound
	}
	return &derivativeNode{State: cfg.State, derivFn: derivFn}, nil
}

type derivativeNode struct {
	*node.State
	derivFn       func(telem.Series, telem.Series, *float64, *telem.TimeStamp, *bool) (telem.Series, telem.Series)
	prevValue     float64
	prevTimestamp telem.TimeStamp
	hasPrev       bool
}

var _ node.Node = (*derivativeNode)(nil)

func (d *derivativeNode) Reset() {
	d.State.Reset()
	d.prevValue = 0
	d.prevTimestamp = 0
	d.hasPrev = false
}

func (d *derivativeNode) Next(ctx node.Context) {
	if !d.RefreshInputs() {
		return
	}
	inputData := d.Input(0)
	inputTime := d.InputTime(0)
	if inputData.Len() == 0 {
		return
	}
	outputData, outputTime := d.derivFn(
		inputData, inputTime,
		&d.prevValue, &d.prevTimestamp, &d.hasPrev,
	)
	outputData.Alignment = inputData.Alignment
	outputData.TimeRange = inputData.TimeRange
	outputTime.Alignment = inputData.Alignment
	outputTime.TimeRange = inputData.TimeRange
	*d.Output(0) = outputData
	*d.OutputTime(0) = outputTime
	ctx.MarkChanged(ir.DefaultOutputParam)
}

func derivF64(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]float64, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := telem.ValueAt[float64](input, int(i))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = (cur - *prevVal) / dtSeconds
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivF32(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]float32, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[float32](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = float32((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivI64(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]int64, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[int64](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = int64((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivI32(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]int32, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[int32](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = int32((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivI16(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]int16, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[int16](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = int16((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivI8(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]int8, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[int8](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = int8((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivU64(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]uint64, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[uint64](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = uint64((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivU32(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]uint32, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[uint32](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = uint32((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivU16(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]uint16, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[uint16](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = uint16((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

func derivU8(input, inputTime telem.Series, prevVal *float64, prevTS *telem.TimeStamp, hasPrev *bool) (telem.Series, telem.Series) {
	n := input.Len()
	out := make([]uint8, n)
	outTime := make([]telem.TimeStamp, n)
	for i := int64(0); i < n; i++ {
		cur := float64(telem.ValueAt[uint8](input, int(i)))
		ts := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
		outTime[i] = ts
		if !*hasPrev {
			out[i] = 0
		} else {
			dtSeconds := float64(ts-*prevTS) / 1e9
			if dtSeconds <= 0 {
				out[i] = 0
			} else {
				out[i] = uint8((cur - *prevVal) / dtSeconds)
			}
		}
		*prevVal = cur
		*prevTS = ts
		*hasPrev = true
	}
	return telem.NewSeriesV(out...), telem.NewSeriesV(outTime...)
}

var derivOps = map[telem.DataType]func(telem.Series, telem.Series, *float64, *telem.TimeStamp, *bool) (telem.Series, telem.Series){
	telem.Float64T: derivF64,
	telem.Float32T: derivF32,
	telem.Int64T:   derivI64,
	telem.Int32T:   derivI32,
	telem.Int16T:   derivI16,
	telem.Int8T:    derivI8,
	telem.Uint64T:  derivU64,
	telem.Uint32T:  derivU32,
	telem.Uint16T:  derivU16,
	telem.Uint8T:   derivU8,
}
