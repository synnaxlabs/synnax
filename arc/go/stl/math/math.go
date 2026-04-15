// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math

import (
	"context"
	"math"

	"github.com/tetratelabs/wazero"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	xmath "github.com/synnaxlabs/x/math"
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
	addSymbolName        = "add"
	subSymbolName        = "subtract"
	mulSymbolName        = "multiply"
	divSymbolName        = "divide"
	modSymbolName        = "mod"
	negSymbolName        = "neg"
)

var numConstraint = types.NumericConstraint()

type (
	reductionFn  = func(telem.Series, int64, *telem.Series) int64
	derivativeFn = func(
		telem.Series,
		telem.Series,
		*float64,
		*telem.TimeStamp,
		*bool,
		*telem.Series,
		*telem.Series,
	)
)

func createBaseSymbol(name string) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{
				{Name: durationConfigParam, Type: types.TimeSpan(), Value: telem.TimeSpanZero},
				{Name: countConfigParam, Type: types.I64(), Value: 0},
			},
			Inputs: types.Params{
				{Name: ir.DefaultInputParam, Type: types.Variable("T", &numConstraint)},
				{Name: resetInputParam, Type: types.U8(), Value: 0},
			},
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.Variable("T", &numConstraint)},
			},
		}),
	}
}

var (
	powSymbol = symbol.Symbol{
		Name: "pow",
		Kind: symbol.KindFunction,
		Exec: symbol.ExecWASM,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "base", Type: types.Variable("T", &numConstraint)}, {Name: "exp", Type: types.Variable("T", &numConstraint)}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", &numConstraint)}},
		}),
	}
	avgSymbol        = createBaseSymbol(avgSymbolName)
	minSymbol        = createBaseSymbol(minSymbolName)
	maxSymbol        = createBaseSymbol(maxSymbolName)
	derivativeSymbol = symbol.Symbol{
		Name: derivativeSymbolName,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: ir.DefaultInputParam, Type: types.Variable("T", &numConstraint)},
			},
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.F64()},
			},
		}),
	}
)

var deprecatedBareResolver = symbol.MapResolver{
	avgSymbolName:        avgSymbol,
	minSymbolName:        minSymbol,
	maxSymbolName:        maxSymbol,
	derivativeSymbolName: derivativeSymbol,
	addSymbolName:        addSymbol,
	subSymbolName:        subSymbol,
	mulSymbolName:        mulSymbol,
	divSymbolName:        divSymbol,
	modSymbolName:        modSymbol,
	negSymbolName:        negSymbol,
}

var moduleMembers = symbol.MapResolver{
	"pow":                powSymbol,
	avgSymbolName:        avgSymbol,
	minSymbolName:        minSymbol,
	maxSymbolName:        maxSymbol,
	derivativeSymbolName: derivativeSymbol,
	addSymbolName:        addSymbol,
	subSymbolName:        subSymbol,
	mulSymbolName:        mulSymbol,
	divSymbolName:        divSymbol,
	modSymbolName:        modSymbol,
	negSymbolName:        negSymbol,
}

var SymbolResolver = symbol.CompoundResolver{
	deprecatedBareResolver,
	&symbol.ModuleResolver{Name: "math", Members: moduleMembers},
}

type Module struct{}

func NewModule(
	ctx context.Context,
	rt wazero.Runtime,
) (*Module, error) {
	m := &Module{}
	if rt == nil {
		return m, nil
	}
	builder := rt.NewHostModuleBuilder("math")
	// i32-compatible types: WASM uses uint32, convert internally
	builder = bindI32Pow[uint8](builder, "u8")
	builder = bindI32Pow[uint16](builder, "u16")
	builder = bindI32Pow[uint32](builder, "u32")
	builder = bindI32Pow[int8](builder, "i8")
	builder = bindI32Pow[int16](builder, "i16")
	builder = bindI32Pow[int32](builder, "i32")
	// i64-compatible types
	builder = bindI64Pow[uint64](builder, "u64")
	builder = bindI64Pow[int64](builder, "i64")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, base float32, exp float32) float32 {
			return float32(math.Pow(float64(base), float64(exp)))
		}).Export("pow_f32")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, base float64, exp float64) float64 {
			return math.Pow(base, exp)
		}).Export("pow_f64")

	builder = bindI32Binary[uint8](builder, "add", "u8", func(a, b uint8) uint8 { return a + b })
	builder = bindI32Binary[uint16](builder, "add", "u16", func(a, b uint16) uint16 { return a + b })
	builder = bindI32Binary[uint32](builder, "add", "u32", func(a, b uint32) uint32 { return a + b })
	builder = bindI32Binary[int8](builder, "add", "i8", func(a, b int8) int8 { return a + b })
	builder = bindI32Binary[int16](builder, "add", "i16", func(a, b int16) int16 { return a + b })
	builder = bindI32Binary[int32](builder, "add", "i32", func(a, b int32) int32 { return a + b })
	builder = bindI64Binary[uint64](builder, "add", "u64", func(a, b uint64) uint64 { return a + b })
	builder = bindI64Binary[int64](builder, "add", "i64", func(a, b int64) int64 { return a + b })
	builder = bindF32Binary(builder, "add", func(a, b float32) float32 { return a + b })
	builder = bindF64Binary(builder, "add", func(a, b float64) float64 { return a + b })

	builder = bindI32Binary[uint8](builder, "subtract", "u8", func(a, b uint8) uint8 { return a - b })
	builder = bindI32Binary[uint16](builder, "subtract", "u16", func(a, b uint16) uint16 { return a - b })
	builder = bindI32Binary[uint32](builder, "subtract", "u32", func(a, b uint32) uint32 { return a - b })
	builder = bindI32Binary[int8](builder, "subtract", "i8", func(a, b int8) int8 { return a - b })
	builder = bindI32Binary[int16](builder, "subtract", "i16", func(a, b int16) int16 { return a - b })
	builder = bindI32Binary[int32](builder, "subtract", "i32", func(a, b int32) int32 { return a - b })
	builder = bindI64Binary[uint64](builder, "subtract", "u64", func(a, b uint64) uint64 { return a - b })
	builder = bindI64Binary[int64](builder, "subtract", "i64", func(a, b int64) int64 { return a - b })
	builder = bindF32Binary(builder, "subtract", func(a, b float32) float32 { return a - b })
	builder = bindF64Binary(builder, "subtract", func(a, b float64) float64 { return a - b })

	builder = bindI32Binary[uint8](builder, "multiply", "u8", func(a, b uint8) uint8 { return a * b })
	builder = bindI32Binary[uint16](builder, "multiply", "u16", func(a, b uint16) uint16 { return a * b })
	builder = bindI32Binary[uint32](builder, "multiply", "u32", func(a, b uint32) uint32 { return a * b })
	builder = bindI32Binary[int8](builder, "multiply", "i8", func(a, b int8) int8 { return a * b })
	builder = bindI32Binary[int16](builder, "multiply", "i16", func(a, b int16) int16 { return a * b })
	builder = bindI32Binary[int32](builder, "multiply", "i32", func(a, b int32) int32 { return a * b })
	builder = bindI64Binary[uint64](builder, "multiply", "u64", func(a, b uint64) uint64 { return a * b })
	builder = bindI64Binary[int64](builder, "multiply", "i64", func(a, b int64) int64 { return a * b })
	builder = bindF32Binary(builder, "multiply", func(a, b float32) float32 { return a * b })
	builder = bindF64Binary(builder, "multiply", func(a, b float64) float64 { return a * b })

	builder = bindI32Binary[uint8](builder, "divide", "u8", safeDiv[uint8])
	builder = bindI32Binary[uint16](builder, "divide", "u16", safeDiv[uint16])
	builder = bindI32Binary[uint32](builder, "divide", "u32", safeDiv[uint32])
	builder = bindI32Binary[int8](builder, "divide", "i8", safeDiv[int8])
	builder = bindI32Binary[int16](builder, "divide", "i16", safeDiv[int16])
	builder = bindI32Binary[int32](builder, "divide", "i32", safeDiv[int32])
	builder = bindI64Binary[uint64](builder, "divide", "u64", safeDiv[uint64])
	builder = bindI64Binary[int64](builder, "divide", "i64", safeDiv[int64])
	builder = bindF32Binary(builder, "divide", func(a, b float32) float32 { return a / b })
	builder = bindF64Binary(builder, "divide", func(a, b float64) float64 { return a / b })

	builder = bindI32Binary[uint8](builder, "mod", "u8", safeMod[uint8])
	builder = bindI32Binary[uint16](builder, "mod", "u16", safeMod[uint16])
	builder = bindI32Binary[uint32](builder, "mod", "u32", safeMod[uint32])
	builder = bindI32Binary[int8](builder, "mod", "i8", safeMod[int8])
	builder = bindI32Binary[int16](builder, "mod", "i16", safeMod[int16])
	builder = bindI32Binary[int32](builder, "mod", "i32", safeMod[int32])
	builder = bindI64Binary[uint64](builder, "mod", "u64", safeMod[uint64])
	builder = bindI64Binary[int64](builder, "mod", "i64", safeMod[int64])
	builder = bindF32Binary(builder, "mod", func(a, b float32) float32 { return float32(math.Mod(float64(a), float64(b))) })
	builder = bindF64Binary(builder, "mod", func(a, b float64) float64 { return math.Mod(a, b) })

	builder = bindI32Unary[int8](builder, "neg", "i8", func(a int8) int8 { return -a })
	builder = bindI32Unary[int16](builder, "neg", "i16", func(a int16) int16 { return -a })
	builder = bindI32Unary[int32](builder, "neg", "i32", func(a int32) int32 { return -a })
	builder = bindI64Unary[int64](builder, "neg", "i64", func(a int64) int64 { return -a })
	builder = bindF32Unary(builder, "neg", func(a float32) float32 { return -a })
	builder = bindF64Unary(builder, "neg", func(a float64) float64 { return -a })

	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Module) Create(_ context.Context, nodeCfg node.Config) (node.Node, error) {
	if nodeCfg.Node.Type == derivativeSymbolName {
		return createDerivative(nodeCfg)
	}
	if cat, ok := arithmeticOps[nodeCfg.Node.Type]; ok {
		return &arithmeticBinary{
			State: nodeCfg.State,
			op:    cat[nodeCfg.State.Input(0).DataType],
		}, nil
	}
	if cat, ok := arithmeticUnaryOps[nodeCfg.Node.Type]; ok {
		return &arithmeticUnary{
			State: nodeCfg.State,
			op:    cat[nodeCfg.State.Input(0).DataType],
		}, nil
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
	var cfg WindowConfig
	if err := windowConfigSchema.Parse(nodeCfg.Node.Config.ValueMap(), &cfg); err != nil {
		return nil, err
	}
	return &avgNode{
		State:       nodeCfg.State,
		resetIdx:    resetIdx,
		process:     reductionFn,
		sampleCount: 0,
		cfg:         cfg,
	}, nil
}

type WindowConfig struct {
	Duration telem.TimeSpan `json:"duration" msgpack:"duration"`
	Count    int64          `json:"count" msgpack:"count"`
}

var windowConfigSchema = zyn.Object(map[string]zyn.Schema{
	durationConfigParam: zyn.Int64().Optional().Coerce(),
	countConfigParam:    zyn.Int64().Optional().Coerce(),
})

type avgNode struct {
	*node.State
	process       reductionFn
	cfg           WindowConfig
	resetIdx      int
	sampleCount   int64
	startTime     telem.TimeStamp
	lastResetTime telem.TimeStamp
}

var _ node.Node = (*avgNode)(nil)

func (r *avgNode) Reset() {
	r.State.Reset()
	r.sampleCount = 0
	r.startTime = 0
	r.lastResetTime = 0
}

func (r *avgNode) Next(ctx node.Context) {
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
	r.sampleCount = r.process(inputData, r.sampleCount, r.Output(0))
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

var (
	ops = map[string]map[telem.DataType]reductionFn{
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
	derivOps = map[telem.DataType]derivativeFn{
		telem.Float64T: op.DerivativeF64,
		telem.Float32T: op.DerivativeF32,
		telem.Int64T:   op.DerivativeI64,
		telem.Int32T:   op.DerivativeI32,
		telem.Int16T:   op.DerivativeI16,
		telem.Int8T:    op.DerivativeI8,
		telem.Uint64T:  op.DerivativeU64,
		telem.Uint32T:  op.DerivativeU32,
		telem.Uint16T:  op.DerivativeU16,
		telem.Uint8T:   op.DerivativeU8,
	}
)

func createDerivative(cfg node.Config) (node.Node, error) {
	inputData := cfg.State.Input(0)
	derivFn, ok := derivOps[inputData.DataType]
	if !ok {
		return nil, query.ErrNotFound
	}
	return &derivativeNode{State: cfg.State, process: derivFn}, nil
}

type derivativeNode struct {
	*node.State
	process       derivativeFn
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
	d.process(
		inputData, inputTime,
		&d.prevValue, &d.prevTimestamp, &d.hasPrev,
		d.Output(0), d.OutputTime(0),
	)
	d.Output(0).Alignment = inputData.Alignment
	d.Output(0).TimeRange = inputData.TimeRange
	d.OutputTime(0).Alignment = inputData.Alignment
	d.OutputTime(0).TimeRange = inputData.TimeRange
	ctx.MarkChanged(ir.DefaultOutputParam)
}

func createArithmeticSymbol(name string) symbol.Symbol {
	constraint := types.NumericConstraint()
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecBoth,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: ir.LHSInputParam, Type: types.Variable("T", &constraint)},
				{Name: ir.RHSInputParam, Type: types.Variable("T", &constraint)},
			},
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)},
			},
		}),
	}
}

func createNegateSymbol(name string) symbol.Symbol {
	constraint := types.SignedNumericConstraint()
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecBoth,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: ir.DefaultInputParam, Type: types.Variable("T", &constraint)},
			},
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)},
			},
		}),
	}
}

var (
	addSymbol = createArithmeticSymbol(addSymbolName)
	subSymbol = createArithmeticSymbol(subSymbolName)
	mulSymbol = createArithmeticSymbol(mulSymbolName)
	divSymbol = createArithmeticSymbol(divSymbolName)
	modSymbol = createArithmeticSymbol(modSymbolName)
	negSymbol = createNegateSymbol(negSymbolName)
)

var (
	arithmeticOps = map[string]map[telem.DataType]op.Binary{
		addSymbolName: {
			telem.Float64T: op.AddF64,
			telem.Float32T: op.AddF32,
			telem.Int64T:   op.AddI64,
			telem.Int32T:   op.AddI32,
			telem.Int16T:   op.AddI16,
			telem.Int8T:    op.AddI8,
			telem.Uint64T:  op.AddU64,
			telem.Uint32T:  op.AddU32,
			telem.Uint16T:  op.AddU16,
			telem.Uint8T:   op.AddU8,
		},
		subSymbolName: {
			telem.Float64T: op.SubtractF64,
			telem.Float32T: op.SubtractF32,
			telem.Int64T:   op.SubtractI64,
			telem.Int32T:   op.SubtractI32,
			telem.Int16T:   op.SubtractI16,
			telem.Int8T:    op.SubtractI8,
			telem.Uint64T:  op.SubtractU64,
			telem.Uint32T:  op.SubtractU32,
			telem.Uint16T:  op.SubtractU16,
			telem.Uint8T:   op.SubtractU8,
		},
		mulSymbolName: {
			telem.Float64T: op.MultiplyF64,
			telem.Float32T: op.MultiplyF32,
			telem.Int64T:   op.MultiplyI64,
			telem.Int32T:   op.MultiplyI32,
			telem.Int16T:   op.MultiplyI16,
			telem.Int8T:    op.MultiplyI8,
			telem.Uint64T:  op.MultiplyU64,
			telem.Uint32T:  op.MultiplyU32,
			telem.Uint16T:  op.MultiplyU16,
			telem.Uint8T:   op.MultiplyU8,
		},
		divSymbolName: {
			telem.Float64T: op.DivideF64,
			telem.Float32T: op.DivideF32,
			telem.Int64T:   op.DivideI64,
			telem.Int32T:   op.DivideI32,
			telem.Int16T:   op.DivideI16,
			telem.Int8T:    op.DivideI8,
			telem.Uint64T:  op.DivideU64,
			telem.Uint32T:  op.DivideU32,
			telem.Uint16T:  op.DivideU16,
			telem.Uint8T:   op.DivideU8,
		},
		modSymbolName: {
			telem.Float64T: op.ModuloF64,
			telem.Float32T: op.ModuloF32,
			telem.Int64T:   op.ModuloI64,
			telem.Int32T:   op.ModuloI32,
			telem.Int16T:   op.ModuloI16,
			telem.Int8T:    op.ModuloI8,
			telem.Uint64T:  op.ModuloU64,
			telem.Uint32T:  op.ModuloU32,
			telem.Uint16T:  op.ModuloU16,
			telem.Uint8T:   op.ModuloU8,
		},
	}
	arithmeticUnaryOps = map[string]map[telem.DataType]op.Unary{
		negSymbolName: {
			telem.Float64T: op.NegateF64,
			telem.Float32T: op.NegateF32,
			telem.Int64T:   op.NegateI64,
			telem.Int32T:   op.NegateI32,
			telem.Int16T:   op.NegateI16,
			telem.Int8T:    op.NegateI8,
			telem.Uint64T:  op.NegateU64,
			telem.Uint32T:  op.NegateU32,
			telem.Uint16T:  op.NegateU16,
			telem.Uint8T:   op.NegateU8,
		},
	}
)

type arithmeticBinary struct {
	*node.State
	op op.Binary
}

func (n *arithmeticBinary) Next(ctx node.Context) {
	if !n.RefreshInputs() {
		return
	}
	lhs, rhs := n.Input(0), n.Input(1)
	n.op(lhs, rhs, n.Output(0))
	*n.OutputTime(0) = n.InputTime(0)
	alignment := lhs.Alignment + rhs.Alignment
	timeRange := telem.TimeRange{Start: lhs.TimeRange.Start, End: lhs.TimeRange.End}
	if !rhs.TimeRange.Start.IsZero() && (timeRange.Start.IsZero() || rhs.TimeRange.Start < timeRange.Start) {
		timeRange.Start = rhs.TimeRange.Start
	}
	if rhs.TimeRange.End > timeRange.End {
		timeRange.End = rhs.TimeRange.End
	}
	n.Output(0).Alignment = alignment
	n.Output(0).TimeRange = timeRange
	n.OutputTime(0).Alignment = alignment
	n.OutputTime(0).TimeRange = timeRange
	ctx.MarkChanged(ir.DefaultOutputParam)
}

type arithmeticUnary struct {
	*node.State
	op op.Unary
}

var _ node.Node = (*arithmeticUnary)(nil)

func (n *arithmeticUnary) Next(ctx node.Context) {
	if !n.RefreshInputs() {
		return
	}
	input := n.Input(0)
	n.op(input, n.Output(0))
	*n.OutputTime(0) = n.InputTime(0)
	n.Output(0).Alignment = input.Alignment
	n.Output(0).TimeRange = input.TimeRange
	n.OutputTime(0).Alignment = input.Alignment
	n.OutputTime(0).TimeRange = input.TimeRange
	ctx.MarkChanged(ir.DefaultOutputParam)
}

type integer interface {
	uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64
}

func safeDiv[T integer](a, b T) T {
	if b == 0 {
		return 0
	}
	return a / b
}

func safeMod[T integer](a, b T) T {
	if b == 0 {
		return 0
	}
	return a % b
}

type i32Powable interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

type i64Powable interface {
	uint64 | int64
}

// bindI32Pow binds an integer power function for a WASM i32-compatible type.
// The exponent arrives as uint32 from WASM, so negative Arc exponents appear as
// large positive values (e.g. -1 becomes 4294967295). On 64-bit platforms,
// int(uint32(x)) is always non-negative, making the 0^(-n) panic in IntPow
// unreachable through this interface.
func bindI32Pow[T i32Powable](builder wazero.HostModuleBuilder, suffix string) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, base uint32, exp uint32) uint32 {
			return uint32(xmath.IntPow(T(base), int(exp)))
		}).Export("pow_" + suffix)
}

// bindI64Pow binds an integer power function for a WASM i64-compatible type.
// Same unsigned exponent representation as bindI32Pow.
func bindI64Pow[T i64Powable](builder wazero.HostModuleBuilder, suffix string) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, base uint64, exp uint64) uint64 {
			return uint64(xmath.IntPow(T(base), int(exp)))
		}).Export("pow_" + suffix)
}

func bindI32Binary[T i32Powable](builder wazero.HostModuleBuilder, name, suffix string, fn func(T, T) T) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, a uint32, b uint32) uint32 {
			return uint32(fn(T(a), T(b)))
		}).Export(name + "_" + suffix)
}

func bindI64Binary[T i64Powable](builder wazero.HostModuleBuilder, name, suffix string, fn func(T, T) T) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, a uint64, b uint64) uint64 {
			return uint64(fn(T(a), T(b)))
		}).Export(name + "_" + suffix)
}

func bindF32Binary(builder wazero.HostModuleBuilder, name string, fn func(float32, float32) float32) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, a float32, b float32) float32 {
			return fn(a, b)
		}).Export(name + "_f32")
}

func bindF64Binary(builder wazero.HostModuleBuilder, name string, fn func(float64, float64) float64) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, a float64, b float64) float64 {
			return fn(a, b)
		}).Export(name + "_f64")
}

func bindI32Unary[T i32Powable](builder wazero.HostModuleBuilder, name, suffix string, fn func(T) T) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, a uint32) uint32 {
			return uint32(fn(T(a)))
		}).Export(name + "_" + suffix)
}

func bindI64Unary[T i64Powable](builder wazero.HostModuleBuilder, name, suffix string, fn func(T) T) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, a uint64) uint64 {
			return uint64(fn(T(a)))
		}).Export(name + "_" + suffix)
}

func bindF32Unary(builder wazero.HostModuleBuilder, name string, fn func(float32) float32) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, a float32) float32 {
			return fn(a)
		}).Export(name + "_f32")
}

func bindF64Unary(builder wazero.HostModuleBuilder, name string, fn func(float64) float64) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, a float64) float64 {
			return fn(a)
		}).Export(name + "_f64")
}
