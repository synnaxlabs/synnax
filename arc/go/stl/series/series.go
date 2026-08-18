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
	resultOut := types.Params{{Name: "result", Type: i32}}
	mod := &symbol.Symbol{Name: Name, Kind: symbol.KindModule, Internal: true}
	mod.AddChild(
		symbol.InternalHostFunc("element_add", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_sub", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_mul", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_div", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_mod", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_radd", rScalarIn, resultOut),
		symbol.InternalHostFunc("element_rsub", rScalarIn, resultOut),
		symbol.InternalHostFunc("element_rmul", rScalarIn, resultOut),
		symbol.InternalHostFunc("element_rdiv", rScalarIn, resultOut),
		symbol.InternalHostFunc("element_rmod", rScalarIn, resultOut),
		symbol.InternalHostFunc("series_add", seriesBinIn, resultOut),
		symbol.InternalHostFunc("series_sub", seriesBinIn, resultOut),
		symbol.InternalHostFunc("series_mul", seriesBinIn, resultOut),
		symbol.InternalHostFunc("series_div", seriesBinIn, resultOut),
		symbol.InternalHostFunc("series_mod", seriesBinIn, resultOut),
		symbol.InternalHostFunc("compare_gt", seriesBinIn, resultOut),
		symbol.InternalHostFunc("compare_lt", seriesBinIn, resultOut),
		symbol.InternalHostFunc("compare_ge", seriesBinIn, resultOut),
		symbol.InternalHostFunc("compare_le", seriesBinIn, resultOut),
		symbol.InternalHostFunc("compare_eq", seriesBinIn, resultOut),
		symbol.InternalHostFunc("compare_ne", seriesBinIn, resultOut),
		symbol.InternalHostFunc("compare_gt_scalar", scalarArithIn, resultOut),
		symbol.InternalHostFunc("compare_lt_scalar", scalarArithIn, resultOut),
		symbol.InternalHostFunc("compare_ge_scalar", scalarArithIn, resultOut),
		symbol.InternalHostFunc("compare_le_scalar", scalarArithIn, resultOut),
		symbol.InternalHostFunc("compare_eq_scalar", scalarArithIn, resultOut),
		symbol.InternalHostFunc("compare_ne_scalar", scalarArithIn, resultOut),
		symbol.InternalHostFunc("and", seriesBinIn, resultOut),
		symbol.InternalHostFunc("or", seriesBinIn, resultOut),
		symbol.InternalHostFunc("and_scalar", scalarArithIn, resultOut),
		symbol.InternalHostFunc("or_scalar", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_band", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_bor", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_bxor", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_shl", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_shr", scalarArithIn, resultOut),
		symbol.InternalHostFunc("element_rband", rScalarIn, resultOut),
		symbol.InternalHostFunc("element_rbor", rScalarIn, resultOut),
		symbol.InternalHostFunc("element_rbxor", rScalarIn, resultOut),
		symbol.InternalHostFunc("element_rshl", rScalarIn, resultOut),
		symbol.InternalHostFunc("element_rshr", rScalarIn, resultOut),
		symbol.InternalHostFunc("series_band", seriesBinIn, resultOut),
		symbol.InternalHostFunc("series_bor", seriesBinIn, resultOut),
		symbol.InternalHostFunc("series_bxor", seriesBinIn, resultOut),
		symbol.InternalHostFunc("series_shl", seriesBinIn, resultOut),
		symbol.InternalHostFunc("series_shr", seriesBinIn, resultOut),
		symbol.InternalHostFunc(
			"bnot",
			types.Params{{Name: "handle", Type: i32}},
			resultOut,
		),
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
		symbol.InternalHostFunc(
			"negate",
			types.Params{{Name: "handle", Type: i32}},
			resultOut,
		),
		symbol.InternalHostFunc(
			"not_bool",
			types.Params{{Name: "handle", Type: i32}},
			resultOut,
		),
		symbol.InternalHostFunc(
			"len",
			types.Params{{Name: "handle", Type: i32}},
			types.Params{{Name: "length", Type: i64}},
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
	)
	return []*symbol.Symbol{mod, newUserLenSymbol()}
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
			startIdx := int(start)
			endIdx := int(end)
			if endIdx < 0 || endIdx > length {
				endIdx = length
			}
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx >= endIdx {
				return 0
			}
			density := int(ser.DataType.Density())
			sliced := telem.Series{
				DataType: ser.DataType,
				Data:     ser.Data[startIdx*density : endIdx*density],
			}
			return s.Store(sliced)
		}).Export("slice")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: telem.BooleanT}
			op.NotBool(ser, &result)
			return s.Store(result)
		}).Export("not_bool")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return h, nil
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
			if ser, ok := s.Get(handle); ok && int64(index) < ser.Len() {
				telem.SetValueAt[bool](ser, int(index), value != 0)
			}
			return handle
		}).Export("set_element_bool")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32) uint32 {
			if ser, ok := s.Get(handle); ok && int64(index) < ser.Len() &&
				telem.ValueAt[bool](ser, int(index)) {
				return 1
			}
			return 0
		}).Export("index_bool")
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, telem.Series, *telem.Series)
	}{
		{"and_bool", op.AndBool},
		{"or_bool", op.OrBool},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, h1, h2 uint32) uint32 {
				s1, ok1 := s.Get(h1)
				s2, ok2 := s.Get(h2)
				if !ok1 || !ok2 {
					return 0
				}
				result := telem.Series{DataType: telem.BooleanT}
				fn(s1, s2, &result)
				return s.Store(result)
			}).Export(entry.name)
	}
	for _, entry := range []struct {
		name string
		or   bool
	}{
		{"and_scalar_bool", false},
		{"or_scalar_bool", true},
	} {
		isOr := entry.or
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle, scalar uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				sv := scalar != 0
				result := telem.MakeSeries(telem.BooleanT, int(ser.Len()))
				for i := 0; i < int(ser.Len()); i++ {
					v := telem.ValueAt[bool](ser, i)
					if isOr {
						v = v || sv
					} else {
						v = v && sv
					}
					telem.SetValueAt[bool](result, i, v)
				}
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

// bitwiseOps bundles the op package bitwise kernels for a single integer type.
type bitwiseOps[T any] struct {
	bandScalar func(telem.Series, T, *telem.Series)
	borScalar  func(telem.Series, T, *telem.Series)
	bxorScalar func(telem.Series, T, *telem.Series)
	shlScalar  func(telem.Series, T, *telem.Series)
	shrScalar  func(telem.Series, T, *telem.Series)
	rShlScalar func(telem.Series, T, *telem.Series)
	rShrScalar func(telem.Series, T, *telem.Series)
	band       func(telem.Series, telem.Series, *telem.Series)
	bor        func(telem.Series, telem.Series, *telem.Series)
	bxor       func(telem.Series, telem.Series, *telem.Series)
	shl        func(telem.Series, telem.Series, *telem.Series)
	shr        func(telem.Series, telem.Series, *telem.Series)
	bnot       func(telem.Series, *telem.Series)
}

// bindBitwiseSeries registers the series-to-series bitwise host functions.
func bindBitwiseSeries[T any](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops bitwiseOps[T],
) wazero.HostModuleBuilder {
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, telem.Series, *telem.Series)
	}{
		{"series_band_", ops.band},
		{"series_bor_", ops.bor},
		{"series_bxor_", ops.bxor},
		{"series_shl_", ops.shl},
		{"series_shr_", ops.shr},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, h1, h2 uint32) uint32 {
				s1, ok1 := s.Get(h1)
				s2, ok2 := s.Get(h2)
				if !ok1 || !ok2 {
					return 0
				}
				if s1.Len() != s2.Len() {
					panic("arc panic: series length mismatch in bitwise operation")
				}
				result := telem.Series{DataType: s1.DataType}
				fn(s1, s2, &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	return builder
}

// bindBitNot registers bnot_<suffix>, the element-wise bitwise complement.
func bindBitNot(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	fn func(telem.Series, *telem.Series),
) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			fn(ser, &result)
			return s.Store(result)
		}).Export("bnot_" + suffix)
}

// bindBitwiseI32 registers the bitwise host functions for integer types that
// map to i32 in WASM. The commutative reverse forms reuse the forward scalar
// kernels; only the shifts have dedicated reverse kernels.
func bindBitwiseI32[T i32Scalar](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops bitwiseOps[T],
) wazero.HostModuleBuilder {
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_band_", ops.bandScalar},
		{"element_bor_", ops.borScalar},
		{"element_bxor_", ops.bxorScalar},
		{"element_shl_", ops.shlScalar},
		{"element_shr_", ops.shrScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle, scalar uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_rband_", ops.bandScalar},
		{"element_rbor_", ops.borScalar},
		{"element_rbxor_", ops.bxorScalar},
		{"element_rshl_", ops.rShlScalar},
		{"element_rshr_", ops.rShrScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, scalar, handle uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	builder = bindBitwiseSeries(builder, s, suffix, ops)
	return bindBitNot(builder, s, suffix, ops.bnot)
}

// bindBitwiseI64 registers the bitwise host functions for integer types that
// map to i64 in WASM.
func bindBitwiseI64[T uint64 | int64](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops bitwiseOps[T],
) wazero.HostModuleBuilder {
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_band_", ops.bandScalar},
		{"element_bor_", ops.borScalar},
		{"element_bxor_", ops.bxorScalar},
		{"element_shl_", ops.shlScalar},
		{"element_shr_", ops.shrScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle uint32, scalar uint64) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_rband_", ops.bandScalar},
		{"element_rbor_", ops.borScalar},
		{"element_rbxor_", ops.bxorScalar},
		{"element_rshl_", ops.rShlScalar},
		{"element_rshr_", ops.rShrScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, scalar uint64, handle uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	builder = bindBitwiseSeries(builder, s, suffix, ops)
	return bindBitNot(builder, s, suffix, ops.bnot)
}

func bindI32Type[T i32Scalar](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops seriesOps[T],
	bw bitwiseOps[T],
) wazero.HostModuleBuilder {
	dt := ops.dt
	builder = bindCreateEmpty(builder, s, suffix, dt)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index, value uint32) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), T(value))
				}
			}
			return handle
		}).Export("set_element_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					return uint32(telem.ValueAt[T](ser, int(index)))
				}
			}
			return 0
		}).Export("index_" + suffix)

	builder = bindElementOpsI32(builder, s, suffix, ops)
	builder = bindSeriesOps(builder, s, suffix, ops)
	builder = bindCompareOps(builder, s, suffix, ops)
	builder = bindCompareScalarI32(builder, s, suffix, ops)
	builder = bindBitwiseI32(builder, s, suffix, bw)
	if ops.negate != nil {
		builder = bindNegate(builder, s, suffix, ops.negate)
	}
	return builder
}

func bindElementOpsI32[T i32Scalar](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_add_", ops.addScalar},
		{"element_sub_", ops.subScalar},
		{"element_mul_", ops.mulScalar},
		{"element_div_", ops.divScalar},
		{"element_mod_", ops.modScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle, scalar uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_rsub_", ops.rSubScalar},
		{"element_rdiv_", ops.rDivScalar},
		{"element_rmod_", ops.rModScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, scalar, handle uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	// Reverse add and mul are commutative - reuse add/mul scalar ops
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, scalar, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.addScalar(ser, T(scalar), &result)
			return s.Store(result)
		}).Export("element_radd_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, scalar, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.mulScalar(ser, T(scalar), &result)
			return s.Store(result)
		}).Export("element_rmul_" + suffix)
	return builder
}

func bindCompareScalarI32[T i32Scalar](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"compare_gt_scalar_", ops.gtScalar},
		{"compare_lt_scalar_", ops.ltScalar},
		{"compare_ge_scalar_", ops.geScalar},
		{"compare_le_scalar_", ops.leScalar},
		{"compare_eq_scalar_", ops.eqScalar},
		{"compare_ne_scalar_", ops.neScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle, scalar uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: telem.BooleanT}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	return builder
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
		fn := entry.fn
		opName := entry.op
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, h1, h2 uint32) uint32 {
				s1, ok1 := s.Get(h1)
				s2, ok2 := s.Get(h2)
				if !ok1 || !ok2 {
					return 0
				}
				if s1.Len() != s2.Len() {
					panic("arc panic: series length mismatch in " + opName)
				}
				result := telem.Series{DataType: s1.DataType}
				fn(s1, s2, &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
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
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, h1, h2 uint32) uint32 {
				s1, ok1 := s.Get(h1)
				s2, ok2 := s.Get(h2)
				if !ok1 || !ok2 {
					return 0
				}
				if s1.Len() != s2.Len() {
					panic("arc panic: series length mismatch in comparison")
				}
				result := telem.Series{DataType: telem.BooleanT}
				fn(s1, s2, &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
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

// bindI64Type handles uint64 and int64 which use i64 in WASM.
func bindI64Type[T uint64 | int64](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	dt telem.DataType,
	ops seriesOps[T],
	bw bitwiseOps[T],
) wazero.HostModuleBuilder {
	builder = bindCreateEmpty(builder, s, suffix, dt)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32, value uint64) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), T(value))
				}
			}
			return handle
		}).Export("set_element_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32) uint64 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					return uint64(telem.ValueAt[T](ser, int(index)))
				}
			}
			return 0
		}).Export("index_" + suffix)

	// Element scalar ops
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_add_", ops.addScalar},
		{"element_sub_", ops.subScalar},
		{"element_mul_", ops.mulScalar},
		{"element_div_", ops.divScalar},
		{"element_mod_", ops.modScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle uint32, scalar uint64) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_rsub_", ops.rSubScalar},
		{"element_rdiv_", ops.rDivScalar},
		{"element_rmod_", ops.rModScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, scalar uint64, handle uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, scalar uint64, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.addScalar(ser, T(scalar), &result)
			return s.Store(result)
		}).Export("element_radd_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, scalar uint64, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.mulScalar(ser, T(scalar), &result)
			return s.Store(result)
		}).Export("element_rmul_" + suffix)

	// Scalar comparisons for i64 types
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"compare_gt_scalar_", ops.gtScalar},
		{"compare_lt_scalar_", ops.ltScalar},
		{"compare_ge_scalar_", ops.geScalar},
		{"compare_le_scalar_", ops.leScalar},
		{"compare_eq_scalar_", ops.eqScalar},
		{"compare_ne_scalar_", ops.neScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle uint32, scalar uint64) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: telem.BooleanT}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}

	builder = bindSeriesOps(builder, s, suffix, ops)
	builder = bindCompareOps(builder, s, suffix, ops)
	builder = bindBitwiseI64(builder, s, suffix, bw)
	if ops.negate != nil {
		builder = bindNegate(builder, s, suffix, ops.negate)
	}
	return builder
}

// bindFloatType handles f32 or f64.
func bindFloatType[T float32 | float64](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	dt telem.DataType,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	builder = bindCreateEmpty(builder, s, suffix, dt)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32, value T) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), value)
				}
			}
			return handle
		}).Export("set_element_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, index uint32) T {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					return telem.ValueAt[T](ser, int(index))
				}
			}
			return 0
		}).Export("index_" + suffix)

	// Element scalar ops
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_add_", ops.addScalar},
		{"element_sub_", ops.subScalar},
		{"element_mul_", ops.mulScalar},
		{"element_div_", ops.divScalar},
		{"element_mod_", ops.modScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle uint32, scalar T) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, scalar, &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"element_rsub_", ops.rSubScalar},
		{"element_rdiv_", ops.rDivScalar},
		{"element_rmod_", ops.rModScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, scalar T, handle uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, scalar, &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, scalar T, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.addScalar(ser, scalar, &result)
			return s.Store(result)
		}).Export("element_radd_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, scalar T, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.mulScalar(ser, scalar, &result)
			return s.Store(result)
		}).Export("element_rmul_" + suffix)

	// Scalar comparisons for float types
	for _, entry := range []struct {
		name string
		fn   func(telem.Series, T, *telem.Series)
	}{
		{"compare_gt_scalar_", ops.gtScalar},
		{"compare_lt_scalar_", ops.ltScalar},
		{"compare_ge_scalar_", ops.geScalar},
		{"compare_le_scalar_", ops.leScalar},
		{"compare_eq_scalar_", ops.eqScalar},
		{"compare_ne_scalar_", ops.neScalar},
	} {
		fn := entry.fn
		builder = builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, handle uint32, scalar T) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: telem.BooleanT}
				fn(ser, scalar, &result)
				return s.Store(result)
			}).Export(entry.name + suffix)
	}

	builder = bindSeriesOps(builder, s, suffix, ops)
	builder = bindCompareOps(builder, s, suffix, ops)
	if ops.negate != nil {
		builder = bindNegate(builder, s, suffix, ops.negate)
	}
	return builder
}

func bindU8(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindI32Type[uint8](builder, s, "u8", seriesOps[uint8]{
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
	}, bitwiseOps[uint8]{
		bandScalar: op.BitAndScalarU8, borScalar: op.BitOrScalarU8,
		bxorScalar: op.BitXorScalarU8, shlScalar: op.ShiftLeftScalarU8,
		shrScalar:  op.ShiftRightScalarU8,
		rShlScalar: op.ReverseShiftLeftScalarU8,
		rShrScalar: op.ReverseShiftRightScalarU8,
		band:       op.BitAndU8, bor: op.BitOrU8, bxor: op.BitXorU8,
		shl: op.ShiftLeftU8, shr: op.ShiftRightU8,
		bnot: op.BitNotU8,
	})
}

func bindU16(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindI32Type[uint16](builder, s, "u16", seriesOps[uint16]{
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
	}, bitwiseOps[uint16]{
		bandScalar: op.BitAndScalarU16, borScalar: op.BitOrScalarU16,
		bxorScalar: op.BitXorScalarU16, shlScalar: op.ShiftLeftScalarU16,
		shrScalar:  op.ShiftRightScalarU16,
		rShlScalar: op.ReverseShiftLeftScalarU16,
		rShrScalar: op.ReverseShiftRightScalarU16,
		band:       op.BitAndU16, bor: op.BitOrU16, bxor: op.BitXorU16,
		shl: op.ShiftLeftU16, shr: op.ShiftRightU16,
		bnot: op.BitNotU16,
	})
}

func bindU32(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindI32Type[uint32](builder, s, "u32", seriesOps[uint32]{
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
	}, bitwiseOps[uint32]{
		bandScalar: op.BitAndScalarU32, borScalar: op.BitOrScalarU32,
		bxorScalar: op.BitXorScalarU32, shlScalar: op.ShiftLeftScalarU32,
		shrScalar:  op.ShiftRightScalarU32,
		rShlScalar: op.ReverseShiftLeftScalarU32,
		rShrScalar: op.ReverseShiftRightScalarU32,
		band:       op.BitAndU32, bor: op.BitOrU32, bxor: op.BitXorU32,
		shl: op.ShiftLeftU32, shr: op.ShiftRightU32,
		bnot: op.BitNotU32,
	})
}

func bindI8(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindI32Type[int8](builder, s, "i8", seriesOps[int8]{
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
	}, bitwiseOps[int8]{
		bandScalar: op.BitAndScalarI8, borScalar: op.BitOrScalarI8,
		bxorScalar: op.BitXorScalarI8, shlScalar: op.ShiftLeftScalarI8,
		shrScalar:  op.ShiftRightScalarI8,
		rShlScalar: op.ReverseShiftLeftScalarI8,
		rShrScalar: op.ReverseShiftRightScalarI8,
		band:       op.BitAndI8, bor: op.BitOrI8, bxor: op.BitXorI8,
		shl: op.ShiftLeftI8, shr: op.ShiftRightI8,
		bnot: op.BitNotI8,
	})
}

func bindI16(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindI32Type[int16](builder, s, "i16", seriesOps[int16]{
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
	}, bitwiseOps[int16]{
		bandScalar: op.BitAndScalarI16, borScalar: op.BitOrScalarI16,
		bxorScalar: op.BitXorScalarI16, shlScalar: op.ShiftLeftScalarI16,
		shrScalar:  op.ShiftRightScalarI16,
		rShlScalar: op.ReverseShiftLeftScalarI16,
		rShrScalar: op.ReverseShiftRightScalarI16,
		band:       op.BitAndI16, bor: op.BitOrI16, bxor: op.BitXorI16,
		shl: op.ShiftLeftI16, shr: op.ShiftRightI16,
		bnot: op.BitNotI16,
	})
}

func bindI32(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindI32Type[int32](builder, s, "i32", seriesOps[int32]{
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
	}, bitwiseOps[int32]{
		bandScalar: op.BitAndScalarI32, borScalar: op.BitOrScalarI32,
		bxorScalar: op.BitXorScalarI32, shlScalar: op.ShiftLeftScalarI32,
		shrScalar:  op.ShiftRightScalarI32,
		rShlScalar: op.ReverseShiftLeftScalarI32,
		rShrScalar: op.ReverseShiftRightScalarI32,
		band:       op.BitAndI32, bor: op.BitOrI32, bxor: op.BitXorI32,
		shl: op.ShiftLeftI32, shr: op.ShiftRightI32,
		bnot: op.BitNotI32,
	})
}

func bindU64(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindI64Type[uint64](builder, s, "u64", telem.Uint64T, seriesOps[uint64]{
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
	}, bitwiseOps[uint64]{
		bandScalar: op.BitAndScalarU64, borScalar: op.BitOrScalarU64,
		bxorScalar: op.BitXorScalarU64, shlScalar: op.ShiftLeftScalarU64,
		shrScalar:  op.ShiftRightScalarU64,
		rShlScalar: op.ReverseShiftLeftScalarU64,
		rShrScalar: op.ReverseShiftRightScalarU64,
		band:       op.BitAndU64, bor: op.BitOrU64, bxor: op.BitXorU64,
		shl: op.ShiftLeftU64, shr: op.ShiftRightU64,
		bnot: op.BitNotU64,
	})
}

func bindI64(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindI64Type[int64](builder, s, "i64", telem.Int64T, seriesOps[int64]{
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
	}, bitwiseOps[int64]{
		bandScalar: op.BitAndScalarI64, borScalar: op.BitOrScalarI64,
		bxorScalar: op.BitXorScalarI64, shlScalar: op.ShiftLeftScalarI64,
		shrScalar:  op.ShiftRightScalarI64,
		rShlScalar: op.ReverseShiftLeftScalarI64,
		rShrScalar: op.ReverseShiftRightScalarI64,
		band:       op.BitAndI64, bor: op.BitOrI64, bxor: op.BitXorI64,
		shl: op.ShiftLeftI64, shr: op.ShiftRightI64,
		bnot: op.BitNotI64,
	})
}

func bindF32(
	builder wazero.HostModuleBuilder,
	s *ProgramState,
) wazero.HostModuleBuilder {
	return bindFloatType[float32](builder, s, "f32", telem.Float32T, seriesOps[float32]{
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
	return bindFloatType[float64](builder, s, "f64", telem.Float64T, seriesOps[float64]{
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
