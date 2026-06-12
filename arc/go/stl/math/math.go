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
	"github.com/synnaxlabs/x/lsp/doc"
	xmath "github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
	"github.com/synnaxlabs/x/zyn"
)

const (
	avgSymbolName        = "avg"
	countConfigParam     = "count"
	derivativeSymbolName = "derivative"
	durationConfigParam  = "duration"
	maxSymbolName        = "max"
	minSymbolName        = "min"
	powSymbolName        = "pow"
	resetInputParam      = "reset"
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

func createBaseSymbol(name string, doc doc.Doc) *symbol.Symbol {
	return &symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: ir.DefaultInputParam, Type: types.Variable("T", &numConstraint)},
				{Name: durationConfigParam, Type: types.TimeSpan(), Value: telem.TimeSpanZero},
				{Name: countConfigParam, Type: types.I64(), Value: 0},
				{Name: resetInputParam, Type: types.U8(), Value: 0},
			},
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.Variable("T", &numConstraint)},
			},
		}),
		Trigger: symbol.TriggerInput(ir.DefaultInputParam),
		Doc:     doc,
	}
}

var (
	avgDoc = doc.New(
		doc.Paragraph("Computes a running average of input values."),
		doc.Divider(),
		doc.Code("arc", "sensor -> math.avg{} -> output"),
		doc.Divider(),
		doc.Paragraph("Reset after a fixed number of samples or a time window:"),
		doc.Divider(),
		doc.Code("arc", "sensor -> math.avg{count=100} -> output\nsensor -> math.avg{duration=5s} -> output"),
		doc.Divider(),
		doc.Paragraph("An optional reset input clears the accumulated average:"),
		doc.Divider(),
		doc.Code("arc", "sensor -> math.avg{} -> output\nreset_signal -> math.avg{}.reset"),
	)
	minDoc = doc.New(
		doc.Paragraph("Tracks the running minimum of input values."),
		doc.Divider(),
		doc.Code("arc", "sensor -> math.min{} -> output"),
		doc.Divider(),
		doc.Paragraph("Reset after a fixed number of samples or a time window:"),
		doc.Divider(),
		doc.Code("arc", "sensor -> math.min{count=100} -> output\nsensor -> math.min{duration=5s} -> output"),
	)
	maxDoc = doc.New(
		doc.Paragraph("Tracks the running maximum of input values."),
		doc.Divider(),
		doc.Code("arc", "sensor -> math.max{} -> output"),
		doc.Divider(),
		doc.Paragraph("Reset after a fixed number of samples or a time window:"),
		doc.Divider(),
		doc.Code("arc", "sensor -> math.max{count=100} -> output\nsensor -> math.max{duration=5s} -> output"),
	)
	derivativeDoc = doc.New(
		doc.Paragraph("Computes the rate of change (derivative) of input values. Output is always f64."),
		doc.Divider(),
		doc.Code("arc", "sensor -> math.derivative{} -> rate_output"),
	)
	moduleDoc = doc.New(
		doc.Paragraph("Numerical primitives: running averages, running min/max, derivatives, and arithmetic helpers."),
	)
)

func newPowSymbol() *symbol.Symbol {
	return &symbol.Symbol{
		Name:     powSymbolName,
		Kind:     symbol.KindFunction,
		Exec:     symbol.ExecWASM,
		Internal: true,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "base", Type: types.Variable("T", &numConstraint)}, {Name: "exp", Type: types.Variable("T", &numConstraint)}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", &numConstraint)}},
		}),
		Trigger: symbol.TriggerOnly,
	}
}

func newDerivativeSymbol() *symbol.Symbol {
	return &symbol.Symbol{
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
		Trigger: symbol.TriggerInput(ir.DefaultInputParam),
		Doc:     derivativeDoc,
	}
}

const name = "math"

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the math module and the deprecated bare aliases (avg, min,
// max, derivative) whose Deprecated field points at the canonical module
// member. Every call allocates new Symbol values so analyses can mutate
// them (e.g. apply type substitutions) without corrupting other analyses.
func NewSymbols() []*symbol.Symbol {
	avg := createBaseSymbol(avgSymbolName, avgDoc)
	min := createBaseSymbol(minSymbolName, minDoc)
	max := createBaseSymbol(maxSymbolName, maxDoc)
	derivative := newDerivativeSymbol()
	mod := &symbol.Symbol{Name: name, Kind: symbol.KindModule, Doc: moduleDoc}
	mod.AddChild(newPowSymbol(), avg, min, max, derivative)
	avgBare := *avg
	avgBare.Deprecated = avg
	minBare := *min
	minBare.Deprecated = min
	maxBare := *max
	maxBare.Deprecated = max
	derivativeBare := *derivative
	derivativeBare.Deprecated = derivative
	return []*symbol.Symbol{mod, &avgBare, &minBare, &maxBare, &derivativeBare}
}

// Host is the runtime host-side support for math: it registers the WASM
// host-function bindings (pow_*, neg_*) and acts as the node factory for
// avg / min / max / derivative.
type Host struct{}

// NewHost registers math's WASM host bindings with rt and returns a Host
// handle that satisfies node.Factory for math's runtime nodes.
func NewHost(ctx context.Context, rt wazero.Runtime) (*Host, error) {
	h := &Host{}
	if rt == nil {
		return h, nil
	}
	builder := rt.NewHostModuleBuilder(name)
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

	builder = bindI32Unary[int8](builder, "neg", "i8", func(a int8) int8 { return -a })
	builder = bindI32Unary[int16](builder, "neg", "i16", func(a int16) int16 { return -a })
	builder = bindI32Unary[int32](builder, "neg", "i32", func(a int32) int32 { return -a })
	builder = bindI64Unary[int64](builder, "neg", "i64", func(a int64) int64 { return -a })
	builder = bindF32Unary(builder, "neg", func(a float32) float32 { return -a })
	builder = bindF64Unary(builder, "neg", func(a float64) float64 { return -a })

	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Host) Create(_ context.Context, nodeCfg node.Config) (node.Node, error) {
	if nodeCfg.Node.Type == derivativeSymbolName {
		return createDerivative(nodeCfg)
	}
	reductionMap, ok := ops[nodeCfg.Node.Type]
	if !ok {
		return nil, query.ErrNotFound
	}
	var (
		inputData      = nodeCfg.State.InputNamed(ir.DefaultInputParam)
		reductionFn    = reductionMap[inputData.DataType]
		resetConnected = false
	)
	if _, found := nodeCfg.Program.Edges.FindByTarget(ir.Handle{
		Node:  nodeCfg.Node.Key,
		Param: resetInputParam,
	}); found {
		resetConnected = true
		nodeCfg.State.InitInputNamed(
			resetInputParam,
			telem.NewSeriesV[uint8](0),
			telem.NewSeriesV[telem.TimeStamp](1),
		)
	}
	var cfg WindowConfig
	if err := windowConfigSchema.Parse(nodeCfg.Node.Inputs.ValueMap(), &cfg); err != nil {
		return nil, err
	}
	return &avgNode{
		State:          nodeCfg.State,
		resetConnected: resetConnected,
		process:        reductionFn,
		sampleCount:    0,
		cfg:            cfg,
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
	process        reductionFn
	cfg            WindowConfig
	resetConnected bool
	sampleCount    int64
	startTime      telem.TimeStamp
	lastResetTime  telem.TimeStamp
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

	inputTime := r.InputTimeNamed(ir.DefaultInputParam)
	if r.startTime == 0 && inputTime.Len() > 0 {
		r.startTime = telem.ValueAt[telem.TimeStamp](inputTime, 0)
	}

	shouldReset := false

	if r.resetConnected {
		resetData := r.InputNamed(resetInputParam)
		resetTime := r.InputTimeNamed(resetInputParam)
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
		inputTime = r.InputTimeNamed(ir.DefaultInputParam)
	}
	inputData := r.InputNamed(ir.DefaultInputParam)
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
	if r.resetConnected {
		resetData := r.InputNamed(resetInputParam)
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
	ctx.MarkChanged(0)
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
	inputData := cfg.State.InputNamed(ir.DefaultInputParam)
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
	inputData := d.InputNamed(ir.DefaultInputParam)
	inputTime := d.InputTimeNamed(ir.DefaultInputParam)
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
	ctx.MarkChanged(0)
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
