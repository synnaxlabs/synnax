// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package series

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
	"github.com/tetratelabs/wazero"
)

var numConstraint = types.NumericConstraint()

func tv() types.Type { return types.Variable("T", &numConstraint) }

var (
	i32 = types.I32()
	i64 = types.I64()
)

var lenDoc = doc.New(
	doc.Paragraph("Returns the length of a series or string as i64."),
	doc.Divider(),
	doc.Code("arc", "length := len(data)"),
)

func newUserLenSymbol() *symbol.Symbol {
	return &symbol.Symbol{
		Name: "len",
		Kind: symbol.KindFunction,
		Exec: symbol.ExecWASM,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)},
			},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		}),
		Trigger: symbol.TriggerOnly,
		Doc:     lenDoc,
	}
}

// Name is the module name.
const Name = "series"

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the series module plus the bare `len` global (a user-facing
// builtin reachable without an import).
func NewSymbols() []*symbol.Symbol {
	scalarArithIn := types.Params{
		{Name: "handle", Type: i32},
		{Name: "scalar", Type: tv()},
	}
	rScalarIn := types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}
	seriesBinIn := types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}
	out := types.Params{{Name: "result", Type: i32}}
	mod := &symbol.Symbol{Name: Name, Kind: symbol.KindModule, Internal: true}
	mod.AddChild(slices.Concat(
		hostFuncs(scalarArithIn, out, "element_add", "element_sub", "element_mul",
			"element_div", "element_mod"),
		hostFuncs(rScalarIn, out, "element_radd", "element_rsub", "element_rmul",
			"element_rdiv", "element_rmod"),
		hostFuncs(seriesBinIn, out, "series_add", "series_sub", "series_mul",
			"series_div", "series_mod"),
		hostFuncs(seriesBinIn, out, "compare_gt", "compare_lt", "compare_ge",
			"compare_le", "compare_eq", "compare_ne"),
		hostFuncs(scalarArithIn, out, "compare_gt_scalar", "compare_lt_scalar",
			"compare_ge_scalar", "compare_le_scalar", "compare_eq_scalar",
			"compare_ne_scalar"),
		hostFuncs(seriesBinIn, out, "and", "or"),
		hostFuncs(scalarArithIn, out, "and_scalar", "or_scalar"),
		handleSymbols(out),
	)...)
	return []*symbol.Symbol{mod, newUserLenSymbol()}
}

// hostFuncs returns an internal host function for each name, all with the same inputs
// and outputs.
func hostFuncs(in, out types.Params, names ...string) []*symbol.Symbol {
	syms := make([]*symbol.Symbol, len(names))
	for i, name := range names {
		syms[i] = symbol.InternalHostFunc(name, in, out)
	}
	return syms
}

func handleSymbols(resultOut types.Params) []*symbol.Symbol {
	handleIn := types.Params{{Name: "handle", Type: i32}}
	return []*symbol.Symbol{
		symbol.InternalHostFunc(
			"create_empty",
			types.Params{{Name: "len", Type: i32}},
			types.Params{{Name: "handle", Type: i32}},
		),
		symbol.InternalHostFunc(
			"set_element",
			types.Params{
				{Name: "handle", Type: i32},
				{Name: "idx", Type: i32},
				{Name: "value", Type: tv()},
			},
			resultOut,
		),
		symbol.InternalHostFunc(
			"index",
			types.Params{{Name: "handle", Type: i32}, {Name: "idx", Type: i32}},
			types.Params{{Name: "value", Type: tv()}},
		),
		symbol.InternalHostFunc("negate", handleIn, resultOut),
		symbol.InternalHostFunc("not", handleIn, resultOut),
		symbol.InternalHostFunc(
			"len",
			handleIn,
			types.Params{{Name: "length", Type: i64}},
		),
		symbol.InternalHostFunc(
			"index_from_i64",
			types.Params{{Name: "idx", Type: i64}},
			types.Params{{Name: "idx", Type: i32}},
		),
		symbol.InternalHostFunc(
			"index_from_u64",
			types.Params{{Name: "idx", Type: types.U64()}},
			types.Params{{Name: "idx", Type: i32}},
		),
		symbol.InternalHostFunc(
			"slice",
			types.Params{
				{Name: "handle", Type: i32},
				{Name: "start", Type: i32},
				{Name: "end", Type: i32},
			},
			resultOut,
		),
	}
}

// Host is the runtime host-side support for the series module: it
// registers the WASM host bindings that allocate and manipulate series
// handles against a series ProgramState.
type Host struct {
	series *ProgramState
}

// NewHost registers the series module's WASM host bindings with rt. The
// bindings allocate, read, and mutate series handles against ps.
func NewHost(
	ctx context.Context,
	rt wazero.Runtime,
	ps *ProgramState,
) (*Host, error) {
	h := &Host{series: ps}
	s := ps
	if rt == nil {
		return h, nil
	}
	builder := rt.NewHostModuleBuilder(Name)
	builder = bindU8(builder, s)
	builder = bindU16(builder, s)
	builder = bindU32(builder, s)
	builder = bindU64(builder, s)
	builder = bindI8(builder, s)
	builder = bindI16(builder, s)
	builder = bindI32(builder, s)
	builder = bindI64(builder, s)
	builder = bindF32(builder, s)
	builder = bindF64(builder, s)
	builder = bindBool(builder, s)
	builder = bindCreateEmpty(builder, s, "str", telem.StringT)
	builder = bindIndexConversions(builder)
	builder = bindHandleOps(builder, s)
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

// bindIndexConversions registers the narrowing of 64-bit indices to the i32 that series
// indexing takes. Both trap on an index outside the i32 range.
func bindIndexConversions(builder wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, index int64) int32 {
			if index < math.MinInt32 || index > math.MaxInt32 {
				panic(fmt.Sprintf(indexOverflow, index))
			}
			return int32(index)
		}).Export("index_from_i64")
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, index uint64) int32 {
			if index > math.MaxInt32 {
				panic(fmt.Sprintf(indexOverflow, index))
			}
			return int32(index)
		}).Export("index_from_u64")
}

// bindHandleOps registers len and slice, which work on a series of any type, and not,
// which works on a bool series.
func bindHandleOps(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32) uint64 {
			if ser, ok := s.Get(handle); ok {
				return uint64(ser.Len())
			}
			return 0
		}).Export("len")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, start, end uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			length := int(ser.Len())
			startIdx := sliceBound(start, length)
			endIdx := sliceBound(end, length)
			if startIdx >= endIdx {
				return 0
			}
			density := int(ser.DataType.Density())
			return s.Store(telem.Series{
				DataType: ser.DataType,
				Data:     slices.Clone(ser.Data[startIdx*density : endIdx*density]),
			})
		}).Export("slice")
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: telem.BooleanT}
			op.Not(ser, &result)
			return s.Store(result)
		}).Export("not")
}

const indexOverflow = "index %d out of bounds for an i32 series index"

// mustIndex returns index as an offset into ser, where a negative index counts from
// the end. It panics when index is outside ser, which traps the guest call.
func mustIndex(ser telem.Series, index uint32) int {
	i, n := int64(int32(index)), ser.Len()
	if i < -n || i >= n {
		panic(fmt.Sprintf("index %d out of bounds for series of size %d", i, n))
	}
	return int(i)
}

// sliceBound returns bound as an offset into a series of length samples. A negative
// bound counts from the end, and a bound outside the series stops at the nearer end.
func sliceBound(bound uint32, length int) int {
	b := int(int32(bound))
	if b < 0 {
		b += length
	}
	return min(max(b, 0), length)
}

// bindCreateEmpty registers create_empty_<suffix>, the allocator shared by
// every series element type.
func bindCreateEmpty(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	dt telem.DataType,
) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, length uint32) uint32 {
			return s.Store(telem.MakeSeries(dt, int(length)))
		}).Export("create_empty_" + suffix)
}

// bindBool registers the series operations valid on a bool series: allocation,
// element access, and indexing. Bool series arise from element-wise comparisons
// and support no arithmetic.
func bindBool(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	builder = bindCreateEmpty(builder, s, "bool", telem.BooleanT)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index, value uint32) uint32 {
			if ser, ok := s.Get(handle); ok {
				ser.SetValueAt(mustIndex(ser, index), bool(value != 0))
			}
			return handle
		}).Export("set_element_bool")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32) uint32 {
			if ser, ok := s.Get(handle); ok &&
				ser.ValueAt[bool](mustIndex(ser, index)) {
				return 1
			}
			return 0
		}).Export("index_bool")
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, telem.Series, *telem.Series)
	}{
		{"and", op.And},
		{"or", op.Or},
	} {
		builder = bindSeriesBinary(
			builder, s, entry.name, "logical operation", telem.BooleanT, entry.fn,
		)
	}
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, bool, *telem.Series)
	}{
		{"and_scalar", op.AndScalar},
		{"or_scalar", op.OrScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle, scalar uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: telem.BooleanT}
				fn(ser, scalar != 0, &result)
				return s.Store(result)
			}).Export(entry.name)
	}
	return builder
}

// i32Scalar is used for types that map to i32 in WASM.
type i32Scalar interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

// seriesOps bundles the op package function references for a single type.
type seriesOps[T any] struct {
	dt         telem.DataType
	addScalar  func(telem.Series, T, *telem.Series)
	subScalar  func(telem.Series, T, *telem.Series)
	mulScalar  func(telem.Series, T, *telem.Series)
	divScalar  func(telem.Series, T, *telem.Series)
	modScalar  func(telem.Series, T, *telem.Series)
	rSubScalar func(telem.Series, T, *telem.Series)
	rDivScalar func(telem.Series, T, *telem.Series)
	rModScalar func(telem.Series, T, *telem.Series)
	add        func(telem.Series, telem.Series, *telem.Series)
	sub        func(telem.Series, telem.Series, *telem.Series)
	mul        func(telem.Series, telem.Series, *telem.Series)
	div        func(telem.Series, telem.Series, *telem.Series)
	mod        func(telem.Series, telem.Series, *telem.Series)
	gt         func(telem.Series, telem.Series, *telem.Series)
	lt         func(telem.Series, telem.Series, *telem.Series)
	ge         func(telem.Series, telem.Series, *telem.Series)
	le         func(telem.Series, telem.Series, *telem.Series)
	eq         func(telem.Series, telem.Series, *telem.Series)
	ne         func(telem.Series, telem.Series, *telem.Series)
	gtScalar   func(telem.Series, T, *telem.Series)
	ltScalar   func(telem.Series, T, *telem.Series)
	geScalar   func(telem.Series, T, *telem.Series)
	leScalar   func(telem.Series, T, *telem.Series)
	eqScalar   func(telem.Series, T, *telem.Series)
	neScalar   func(telem.Series, T, *telem.Series)
	negate     func(telem.Series, *telem.Series) // nil for unsigned types
}

// numeric is a series element type, and wasmValue is the WASM value type that carries
// it across the host boundary.
type (
	numeric interface {
		i32Scalar | uint64 | int64 | float32 | float64
	}
	wasmValue interface {
		uint32 | uint64 | float32 | float64
	}
)

// bindNumericType registers every series operation for element type T, whose values
// cross the host boundary as W.
func bindNumericType[T numeric, W wasmValue](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	builder = bindCreateEmpty(builder, s, suffix, ops.dt)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32, value W) uint32 {
			if ser, ok := s.Get(handle); ok {
				ser.SetValueAt(mustIndex(ser, index), T(value))
			}
			return handle
		}).Export("set_element_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32) W {
			if ser, ok := s.Get(handle); ok {
				return W(ser.ValueAt[T](mustIndex(ser, index)))
			}
			return 0
		}).Export("index_" + suffix)
	builder = bindScalarOps[T, W](builder, s, suffix, ops)
	builder = bindSeriesOps(builder, s, suffix, ops)
	builder = bindCompareOps(builder, s, suffix, ops)
	if ops.negate != nil {
		builder = bindNegate(builder, s, suffix, ops.negate)
	}
	return builder
}

// scalarOp is a host function that combines a series with a scalar. UnknownT as
// resultDT keeps the series' type.
type scalarOp[T any] struct {
	name     string
	fn       func(telem.Series, T, *telem.Series)
	resultDT telem.DataType
}

// bindScalarOps registers the element-wise arithmetic and comparison operations between
// a series and a scalar, in both operand orders.
func bindScalarOps[T numeric, W wasmValue](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	seriesFirst := []scalarOp[T]{
		{"element_add_", ops.addScalar, telem.UnknownT},
		{"element_sub_", ops.subScalar, telem.UnknownT},
		{"element_mul_", ops.mulScalar, telem.UnknownT},
		{"element_div_", ops.divScalar, telem.UnknownT},
		{"element_mod_", ops.modScalar, telem.UnknownT},
		{"compare_gt_scalar_", ops.gtScalar, telem.BooleanT},
		{"compare_lt_scalar_", ops.ltScalar, telem.BooleanT},
		{"compare_ge_scalar_", ops.geScalar, telem.BooleanT},
		{"compare_le_scalar_", ops.leScalar, telem.BooleanT},
		{"compare_eq_scalar_", ops.eqScalar, telem.BooleanT},
		{"compare_ne_scalar_", ops.neScalar, telem.BooleanT},
	}
	// Add and mul commute, so their reverse forms reuse the forward ops.
	scalarFirst := []scalarOp[T]{
		{"element_radd_", ops.addScalar, telem.UnknownT},
		{"element_rsub_", ops.rSubScalar, telem.UnknownT},
		{"element_rmul_", ops.mulScalar, telem.UnknownT},
		{"element_rdiv_", ops.rDivScalar, telem.UnknownT},
		{"element_rmod_", ops.rModScalar, telem.UnknownT},
	}
	for _, o := range seriesFirst {
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle uint32, scalar W) uint32 {
				return applyScalarOp(s, o, handle, T(scalar))
			}).Export(o.name + suffix)
	}
	for _, o := range scalarFirst {
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, scalar W, handle uint32) uint32 {
				return applyScalarOp(s, o, handle, T(scalar))
			}).Export(o.name + suffix)
	}
	return builder
}

func applyScalarOp[T any](
	s *ProgramState,
	o scalarOp[T],
	handle uint32,
	scalar T,
) uint32 {
	ser, ok := s.Get(handle)
	if !ok {
		return 0
	}
	dt := o.resultDT
	if dt == telem.UnknownT {
		dt = ser.DataType
	}
	result := telem.Series{DataType: dt}
	o.fn(ser, scalar, &result)
	return s.Store(result)
}

// bindSeriesBinary registers a host function that takes two series handles and
// stores the result of fn. UnknownT as resultDT keeps the left operand's type.
func bindSeriesBinary(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	name, opName string,
	resultDT telem.DataType,
	fn func(telem.Series, telem.Series, *telem.Series),
) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, h1, h2 uint32) uint32 {
			s1, ok1 := s.Get(h1)
			s2, ok2 := s.Get(h2)
			if !ok1 || !ok2 {
				return 0
			}
			if s1.Len() != s2.Len() {
				panic("arc panic: series length mismatch in " + opName)
			}
			dt := resultDT
			if dt == telem.UnknownT {
				dt = s1.DataType
			}
			result := telem.Series{DataType: dt}
			fn(s1, s2, &result)
			return s.Store(result)
		}).Export(name)
}

func bindSeriesOps[T any](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, telem.Series, *telem.Series)
		op   string
	}{
		{"series_add_", ops.add, "addition"},
		{"series_sub_", ops.sub, "subtraction"},
		{"series_mul_", ops.mul, "multiplication"},
		{"series_div_", ops.div, "division"},
		{"series_mod_", ops.mod, "modulo"},
	} {
		builder = bindSeriesBinary(
			builder, s, entry.name+suffix, entry.op, telem.UnknownT, entry.fn,
		)
	}
	return builder
}

func bindCompareOps[T any](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, telem.Series, *telem.Series)
	}{
		{"compare_gt_", ops.gt},
		{"compare_lt_", ops.lt},
		{"compare_ge_", ops.ge},
		{"compare_le_", ops.le},
		{"compare_eq_", ops.eq},
		{"compare_ne_", ops.ne},
	} {
		builder = bindSeriesBinary(
			builder, s, entry.name+suffix, "comparison", telem.BooleanT, entry.fn,
		)
	}
	return builder
}

func bindNegate(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	fn func(telem.Series, *telem.Series),
) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			fn(ser, &result)
			return s.Store(result)
		}).Export("negate_" + suffix)
	return builder
}

func bindU8(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[uint8, uint32](builder, s, "u8", seriesOps[uint8]{
		dt:        telem.Uint8T,
		addScalar: op.AddScalarU8, subScalar: op.SubtractScalarU8,
		mulScalar: op.MultiplyScalarU8, divScalar: op.DivideScalarU8,
		modScalar:  op.ModuloScalarU8,
		rSubScalar: op.ReverseSubtractScalarU8, rDivScalar: op.ReverseDivideScalarU8,
		rModScalar: op.ReverseModuloScalarU8,
		add:        op.AddU8, sub: op.SubtractU8, mul: op.MultiplyU8,
		div: op.DivideU8, mod: op.ModuloU8,
		gt: op.GreaterThanU8, lt: op.LessThanU8,
		ge: op.GreaterThanOrEqualU8, le: op.LessThanOrEqualU8,
		eq: op.EqualU8, ne: op.NotEqualU8,
		gtScalar: op.GreaterThanScalarU8, ltScalar: op.LessThanScalarU8,
		geScalar: op.GreaterThanOrEqualScalarU8,
		leScalar: op.LessThanOrEqualScalarU8,
		eqScalar: op.EqualScalarU8, neScalar: op.NotEqualScalarU8,
	})
}

func bindU16(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[uint16, uint32](builder, s, "u16", seriesOps[uint16]{
		dt:        telem.Uint16T,
		addScalar: op.AddScalarU16, subScalar: op.SubtractScalarU16,
		mulScalar: op.MultiplyScalarU16, divScalar: op.DivideScalarU16,
		modScalar:  op.ModuloScalarU16,
		rSubScalar: op.ReverseSubtractScalarU16, rDivScalar: op.ReverseDivideScalarU16,
		rModScalar: op.ReverseModuloScalarU16,
		add:        op.AddU16, sub: op.SubtractU16, mul: op.MultiplyU16,
		div: op.DivideU16, mod: op.ModuloU16,
		gt: op.GreaterThanU16, lt: op.LessThanU16,
		ge: op.GreaterThanOrEqualU16, le: op.LessThanOrEqualU16,
		eq: op.EqualU16, ne: op.NotEqualU16,
		gtScalar: op.GreaterThanScalarU16, ltScalar: op.LessThanScalarU16,
		geScalar: op.GreaterThanOrEqualScalarU16,
		leScalar: op.LessThanOrEqualScalarU16,
		eqScalar: op.EqualScalarU16, neScalar: op.NotEqualScalarU16,
	})
}

func bindU32(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[uint32, uint32](builder, s, "u32", seriesOps[uint32]{
		dt:        telem.Uint32T,
		addScalar: op.AddScalarU32, subScalar: op.SubtractScalarU32,
		mulScalar: op.MultiplyScalarU32, divScalar: op.DivideScalarU32,
		modScalar:  op.ModuloScalarU32,
		rSubScalar: op.ReverseSubtractScalarU32, rDivScalar: op.ReverseDivideScalarU32,
		rModScalar: op.ReverseModuloScalarU32,
		add:        op.AddU32, sub: op.SubtractU32, mul: op.MultiplyU32,
		div: op.DivideU32, mod: op.ModuloU32,
		gt: op.GreaterThanU32, lt: op.LessThanU32,
		ge: op.GreaterThanOrEqualU32, le: op.LessThanOrEqualU32,
		eq: op.EqualU32, ne: op.NotEqualU32,
		gtScalar: op.GreaterThanScalarU32, ltScalar: op.LessThanScalarU32,
		geScalar: op.GreaterThanOrEqualScalarU32,
		leScalar: op.LessThanOrEqualScalarU32,
		eqScalar: op.EqualScalarU32, neScalar: op.NotEqualScalarU32,
	})
}

func bindI8(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[int8, uint32](builder, s, "i8", seriesOps[int8]{
		dt:        telem.Int8T,
		addScalar: op.AddScalarI8, subScalar: op.SubtractScalarI8,
		mulScalar: op.MultiplyScalarI8, divScalar: op.DivideScalarI8,
		modScalar:  op.ModuloScalarI8,
		rSubScalar: op.ReverseSubtractScalarI8, rDivScalar: op.ReverseDivideScalarI8,
		rModScalar: op.ReverseModuloScalarI8,
		add:        op.AddI8, sub: op.SubtractI8, mul: op.MultiplyI8,
		div: op.DivideI8, mod: op.ModuloI8,
		gt: op.GreaterThanI8, lt: op.LessThanI8,
		ge: op.GreaterThanOrEqualI8, le: op.LessThanOrEqualI8,
		eq: op.EqualI8, ne: op.NotEqualI8,
		gtScalar: op.GreaterThanScalarI8, ltScalar: op.LessThanScalarI8,
		geScalar: op.GreaterThanOrEqualScalarI8,
		leScalar: op.LessThanOrEqualScalarI8,
		eqScalar: op.EqualScalarI8, neScalar: op.NotEqualScalarI8,
		negate: op.NegateI8,
	})
}

func bindI16(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[int16, uint32](builder, s, "i16", seriesOps[int16]{
		dt:        telem.Int16T,
		addScalar: op.AddScalarI16, subScalar: op.SubtractScalarI16,
		mulScalar: op.MultiplyScalarI16, divScalar: op.DivideScalarI16,
		modScalar:  op.ModuloScalarI16,
		rSubScalar: op.ReverseSubtractScalarI16, rDivScalar: op.ReverseDivideScalarI16,
		rModScalar: op.ReverseModuloScalarI16,
		add:        op.AddI16, sub: op.SubtractI16, mul: op.MultiplyI16,
		div: op.DivideI16, mod: op.ModuloI16,
		gt: op.GreaterThanI16, lt: op.LessThanI16,
		ge: op.GreaterThanOrEqualI16, le: op.LessThanOrEqualI16,
		eq: op.EqualI16, ne: op.NotEqualI16,
		gtScalar: op.GreaterThanScalarI16, ltScalar: op.LessThanScalarI16,
		geScalar: op.GreaterThanOrEqualScalarI16,
		leScalar: op.LessThanOrEqualScalarI16,
		eqScalar: op.EqualScalarI16, neScalar: op.NotEqualScalarI16,
		negate: op.NegateI16,
	})
}

func bindI32(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[int32, uint32](builder, s, "i32", seriesOps[int32]{
		dt:        telem.Int32T,
		addScalar: op.AddScalarI32, subScalar: op.SubtractScalarI32,
		mulScalar: op.MultiplyScalarI32, divScalar: op.DivideScalarI32,
		modScalar:  op.ModuloScalarI32,
		rSubScalar: op.ReverseSubtractScalarI32, rDivScalar: op.ReverseDivideScalarI32,
		rModScalar: op.ReverseModuloScalarI32,
		add:        op.AddI32, sub: op.SubtractI32, mul: op.MultiplyI32,
		div: op.DivideI32, mod: op.ModuloI32,
		gt: op.GreaterThanI32, lt: op.LessThanI32,
		ge: op.GreaterThanOrEqualI32, le: op.LessThanOrEqualI32,
		eq: op.EqualI32, ne: op.NotEqualI32,
		gtScalar: op.GreaterThanScalarI32, ltScalar: op.LessThanScalarI32,
		geScalar: op.GreaterThanOrEqualScalarI32,
		leScalar: op.LessThanOrEqualScalarI32,
		eqScalar: op.EqualScalarI32, neScalar: op.NotEqualScalarI32,
		negate: op.NegateI32,
	})
}

func bindU64(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[uint64, uint64](builder, s, "u64", seriesOps[uint64]{
		dt:        telem.Uint64T,
		addScalar: op.AddScalarU64, subScalar: op.SubtractScalarU64,
		mulScalar: op.MultiplyScalarU64, divScalar: op.DivideScalarU64,
		modScalar:  op.ModuloScalarU64,
		rSubScalar: op.ReverseSubtractScalarU64, rDivScalar: op.ReverseDivideScalarU64,
		rModScalar: op.ReverseModuloScalarU64,
		add:        op.AddU64, sub: op.SubtractU64, mul: op.MultiplyU64,
		div: op.DivideU64, mod: op.ModuloU64,
		gt: op.GreaterThanU64, lt: op.LessThanU64,
		ge: op.GreaterThanOrEqualU64, le: op.LessThanOrEqualU64,
		eq: op.EqualU64, ne: op.NotEqualU64,
		gtScalar: op.GreaterThanScalarU64, ltScalar: op.LessThanScalarU64,
		geScalar: op.GreaterThanOrEqualScalarU64,
		leScalar: op.LessThanOrEqualScalarU64,
		eqScalar: op.EqualScalarU64, neScalar: op.NotEqualScalarU64,
	})
}

func bindI64(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[int64, uint64](builder, s, "i64", seriesOps[int64]{
		dt:        telem.Int64T,
		addScalar: op.AddScalarI64, subScalar: op.SubtractScalarI64,
		mulScalar: op.MultiplyScalarI64, divScalar: op.DivideScalarI64,
		modScalar:  op.ModuloScalarI64,
		rSubScalar: op.ReverseSubtractScalarI64, rDivScalar: op.ReverseDivideScalarI64,
		rModScalar: op.ReverseModuloScalarI64,
		add:        op.AddI64, sub: op.SubtractI64, mul: op.MultiplyI64,
		div: op.DivideI64, mod: op.ModuloI64,
		gt: op.GreaterThanI64, lt: op.LessThanI64,
		ge: op.GreaterThanOrEqualI64, le: op.LessThanOrEqualI64,
		eq: op.EqualI64, ne: op.NotEqualI64,
		gtScalar: op.GreaterThanScalarI64, ltScalar: op.LessThanScalarI64,
		geScalar: op.GreaterThanOrEqualScalarI64,
		leScalar: op.LessThanOrEqualScalarI64,
		eqScalar: op.EqualScalarI64, neScalar: op.NotEqualScalarI64,
		negate: op.NegateI64,
	})
}

func bindF32(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[float32, float32](builder, s, "f32", seriesOps[float32]{
		dt:        telem.Float32T,
		addScalar: op.AddScalarF32, subScalar: op.SubtractScalarF32,
		mulScalar: op.MultiplyScalarF32, divScalar: op.DivideScalarF32,
		modScalar:  op.ModuloScalarF32,
		rSubScalar: op.ReverseSubtractScalarF32, rDivScalar: op.ReverseDivideScalarF32,
		rModScalar: op.ReverseModuloScalarF32,
		add:        op.AddF32, sub: op.SubtractF32, mul: op.MultiplyF32,
		div: op.DivideF32, mod: op.ModuloF32,
		gt: op.GreaterThanF32, lt: op.LessThanF32,
		ge: op.GreaterThanOrEqualF32, le: op.LessThanOrEqualF32,
		eq: op.EqualF32, ne: op.NotEqualF32,
		gtScalar: op.GreaterThanScalarF32, ltScalar: op.LessThanScalarF32,
		geScalar: op.GreaterThanOrEqualScalarF32,
		leScalar: op.LessThanOrEqualScalarF32,
		eqScalar: op.EqualScalarF32, neScalar: op.NotEqualScalarF32,
		negate: op.NegateF32,
	})
}

func bindF64(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindNumericType[float64, float64](builder, s, "f64", seriesOps[float64]{
		dt:        telem.Float64T,
		addScalar: op.AddScalarF64, subScalar: op.SubtractScalarF64,
		mulScalar: op.MultiplyScalarF64, divScalar: op.DivideScalarF64,
		modScalar:  op.ModuloScalarF64,
		rSubScalar: op.ReverseSubtractScalarF64, rDivScalar: op.ReverseDivideScalarF64,
		rModScalar: op.ReverseModuloScalarF64,
		add:        op.AddF64, sub: op.SubtractF64, mul: op.MultiplyF64,
		div: op.DivideF64, mod: op.ModuloF64,
		gt: op.GreaterThanF64, lt: op.LessThanF64,
		ge: op.GreaterThanOrEqualF64, le: op.LessThanOrEqualF64,
		eq: op.EqualF64, ne: op.NotEqualF64,
		gtScalar: op.GreaterThanScalarF64, ltScalar: op.LessThanScalarF64,
		geScalar: op.GreaterThanOrEqualScalarF64,
		leScalar: op.LessThanOrEqualScalarF64,
		eqScalar: op.EqualScalarF64, neScalar: op.NotEqualScalarF64,
		negate: op.NegateF64,
	})
}
