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
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
)

var numConstraint = types.NumericConstraint()

func tv() types.Type { return types.Variable("T", &numConstraint) }

func polyFunc(inputs, outputs types.Params) types.Type {
	return types.Function(types.FunctionProperties{Inputs: inputs, Outputs: outputs})
}

func monoFunc(inputs, outputs types.Params) types.Type {
	return types.Function(types.FunctionProperties{Inputs: inputs, Outputs: outputs})
}

var i32 = types.I32()
var i64 = types.I64()

var SymbolResolver = symbol.MapResolver{
	"len": {
		Name: "len",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		}),
	},
}

// CompilerSymbolResolver contains the full set of symbol definitions used by
// the compiler for WASM coordinate derivation. This is the canonical source of
// truth for series host function type signatures.
var CompilerSymbolResolver = symbol.MapResolver{
	"element_add":       {Name: "element_add", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"element_sub":       {Name: "element_sub", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"element_mul":       {Name: "element_mul", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"element_div":       {Name: "element_div", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"element_mod":       {Name: "element_mod", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"element_radd":      {Name: "element_radd", Type: polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"element_rsub":      {Name: "element_rsub", Type: polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"element_rmul":      {Name: "element_rmul", Type: polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"element_rdiv":      {Name: "element_rdiv", Type: polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"element_rmod":      {Name: "element_rmod", Type: polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"series_add":        {Name: "series_add", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"series_sub":        {Name: "series_sub", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"series_mul":        {Name: "series_mul", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"series_div":        {Name: "series_div", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"series_mod":        {Name: "series_mod", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"compare_gt":        {Name: "compare_gt", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"compare_lt":        {Name: "compare_lt", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"compare_ge":        {Name: "compare_ge", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"compare_le":        {Name: "compare_le", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"compare_eq":        {Name: "compare_eq", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"compare_ne":        {Name: "compare_ne", Type: polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"compare_gt_scalar": {Name: "compare_gt_scalar", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"compare_lt_scalar": {Name: "compare_lt_scalar", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"compare_ge_scalar": {Name: "compare_ge_scalar", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"compare_le_scalar": {Name: "compare_le_scalar", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"compare_eq_scalar": {Name: "compare_eq_scalar", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"compare_ne_scalar": {Name: "compare_ne_scalar", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"create_empty":      {Name: "create_empty", Type: polyFunc(types.Params{{Name: "len", Type: i32}}, types.Params{{Name: "handle", Type: i32}})},
	"set_element":       {Name: "set_element", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "idx", Type: i32}, {Name: "value", Type: tv()}}, types.Params{{Name: "result", Type: i32}})},
	"index":             {Name: "index", Type: polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "idx", Type: i32}}, types.Params{{Name: "value", Type: tv()}})},
	"negate":            {Name: "negate", Type: polyFunc(types.Params{{Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"not_u8":            {Name: "not_u8", Type: monoFunc(types.Params{{Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
	"len":               {Name: "len", Type: monoFunc(types.Params{{Name: "handle", Type: i32}}, types.Params{{Name: "length", Type: i64}})},
	"slice":             {Name: "slice", Type: monoFunc(types.Params{{Name: "handle", Type: i32}, {Name: "start", Type: i32}, {Name: "end", Type: i32}}, types.Params{{Name: "result", Type: i32}})},
}

type Module struct {
	series *state.SeriesHandleStore
}

func NewModule(s *state.SeriesHandleStore) *Module { return &Module{series: s} }

var compilerModResolver = &symbol.ModuleResolver{
	Name:    "series",
	Members: CompilerSymbolResolver,
}

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	if sym, err := SymbolResolver.Resolve(ctx, name); err == nil {
		return sym, nil
	}
	return compilerModResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	syms1, _ := SymbolResolver.Search(ctx, term)
	syms2, _ := compilerModResolver.Search(ctx, term)
	return append(syms1, syms2...), nil
}

func (m *Module) Create(_ context.Context, _ node.Config) (node.Node, error) {
	return nil, query.ErrNotFound
}

func (m *Module) BindTo(_ context.Context, rt stl.HostRuntime) error {
	s := m.series
	bindU8(rt, s)
	bindU16(rt, s)
	bindU32(rt, s)
	bindU64(rt, s)
	bindI8(rt, s)
	bindI16(rt, s)
	bindI32(rt, s)
	bindI64(rt, s)
	bindF32(rt, s)
	bindF64(rt, s)

	// Untyped operations
	stl.MustExport(rt, "series", "len", func(_ context.Context, handle uint32) uint64 {
		if ser, ok := s.Get(handle); ok {
			return uint64(ser.Len())
		}
		return 0
	})
	stl.MustExport(rt, "series", "slice",
		func(_ context.Context, handle uint32, start uint32, end uint32) uint32 {
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
		})
	stl.MustExport(rt, "series", "not_u8", func(_ context.Context, handle uint32) uint32 {
		ser, ok := s.Get(handle)
		if !ok {
			return 0
		}
		result := telem.Series{DataType: telem.Uint8T}
		op.NotU8(ser, &result)
		return s.Store(result)
	})
	return nil
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

func bindI32Type[T i32Scalar](
	rt stl.HostRuntime,
	s *state.SeriesHandleStore,
	suffix string,
	ops seriesOps[T],
) {
	dt := ops.dt
	stl.MustExport(rt, "series", "create_empty_"+suffix,
		func(_ context.Context, length uint32) uint32 {
			return s.Store(telem.MakeSeries(dt, int(length)))
		})
	stl.MustExport(rt, "series", "set_element_"+suffix,
		func(_ context.Context, handle uint32, index uint32, value uint32) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), T(value))
				}
			}
			return handle
		})
	stl.MustExport(rt, "series", "index_"+suffix,
		func(_ context.Context, handle uint32, index uint32) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					return uint32(telem.ValueAt[T](ser, int(index)))
				}
			}
			return 0
		})

	bindElementOpsI32(rt, s, suffix, ops)
	bindSeriesOps(rt, s, suffix, ops)
	bindCompareOps(rt, s, suffix, ops)
	bindCompareScalarI32(rt, s, suffix, ops)
	if ops.negate != nil {
		bindNegate(rt, s, suffix, ops.negate)
	}
}

func bindElementOpsI32[T i32Scalar](
	rt stl.HostRuntime,
	s *state.SeriesHandleStore,
	suffix string,
	ops seriesOps[T],
) {
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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, handle uint32, scalar uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			})
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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, scalar uint32, handle uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			})
	}
	// Reverse add and mul are commutative - reuse add/mul scalar ops
	stl.MustExport(rt, "series", "element_radd_"+suffix,
		func(_ context.Context, scalar uint32, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.addScalar(ser, T(scalar), &result)
			return s.Store(result)
		})
	stl.MustExport(rt, "series", "element_rmul_"+suffix,
		func(_ context.Context, scalar uint32, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.mulScalar(ser, T(scalar), &result)
			return s.Store(result)
		})
}

func bindCompareScalarI32[T i32Scalar](
	rt stl.HostRuntime,
	s *state.SeriesHandleStore,
	suffix string,
	ops seriesOps[T],
) {
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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, handle uint32, scalar uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: telem.Uint8T}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			})
	}
}

func bindSeriesOps[T any](
	rt stl.HostRuntime,
	s *state.SeriesHandleStore,
	suffix string,
	ops seriesOps[T],
) {
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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, h1 uint32, h2 uint32) uint32 {
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
			})
	}
}

func bindCompareOps[T any](
	rt stl.HostRuntime,
	s *state.SeriesHandleStore,
	suffix string,
	ops seriesOps[T],
) {
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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, h1 uint32, h2 uint32) uint32 {
				s1, ok1 := s.Get(h1)
				s2, ok2 := s.Get(h2)
				if !ok1 || !ok2 {
					return 0
				}
				if s1.Len() != s2.Len() {
					panic("arc panic: series length mismatch in comparison")
				}
				result := telem.Series{DataType: telem.Uint8T}
				fn(s1, s2, &result)
				return s.Store(result)
			})
	}
}

func bindNegate(
	rt stl.HostRuntime,
	s *state.SeriesHandleStore,
	suffix string,
	fn func(telem.Series, *telem.Series),
) {
	stl.MustExport(rt, "series", "negate_"+suffix,
		func(_ context.Context, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			fn(ser, &result)
			return s.Store(result)
		})
}

// bindI64Type handles uint64 and int64 which use i64 in WASM.
func bindI64Type[T uint64 | int64](
	rt stl.HostRuntime,
	s *state.SeriesHandleStore,
	suffix string,
	dt telem.DataType,
	ops seriesOps[T],
) {
	stl.MustExport(rt, "series", "create_empty_"+suffix,
		func(_ context.Context, length uint32) uint32 {
			return s.Store(telem.MakeSeries(dt, int(length)))
		})
	stl.MustExport(rt, "series", "set_element_"+suffix,
		func(_ context.Context, handle uint32, index uint32, value uint64) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), T(value))
				}
			}
			return handle
		})
	stl.MustExport(rt, "series", "index_"+suffix,
		func(_ context.Context, handle uint32, index uint32) uint64 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					return uint64(telem.ValueAt[T](ser, int(index)))
				}
			}
			return 0
		})

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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, handle uint32, scalar uint64) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			})
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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, scalar uint64, handle uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			})
	}
	stl.MustExport(rt, "series", "element_radd_"+suffix,
		func(_ context.Context, scalar uint64, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.addScalar(ser, T(scalar), &result)
			return s.Store(result)
		})
	stl.MustExport(rt, "series", "element_rmul_"+suffix,
		func(_ context.Context, scalar uint64, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.mulScalar(ser, T(scalar), &result)
			return s.Store(result)
		})

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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, handle uint32, scalar uint64) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: telem.Uint8T}
				fn(ser, T(scalar), &result)
				return s.Store(result)
			})
	}

	bindSeriesOps(rt, s, suffix, ops)
	bindCompareOps(rt, s, suffix, ops)
	if ops.negate != nil {
		bindNegate(rt, s, suffix, ops.negate)
	}
}

// bindFloatType handles f32 or f64.
func bindFloatType[T float32 | float64](
	rt stl.HostRuntime,
	s *state.SeriesHandleStore,
	suffix string,
	dt telem.DataType,
	ops seriesOps[T],
) {
	stl.MustExport(rt, "series", "create_empty_"+suffix,
		func(_ context.Context, length uint32) uint32 {
			return s.Store(telem.MakeSeries(dt, int(length)))
		})
	stl.MustExport(rt, "series", "set_element_"+suffix,
		func(_ context.Context, handle uint32, index uint32, value T) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), value)
				}
			}
			return handle
		})
	stl.MustExport(rt, "series", "index_"+suffix,
		func(_ context.Context, handle uint32, index uint32) T {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					return telem.ValueAt[T](ser, int(index))
				}
			}
			return 0
		})

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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, handle uint32, scalar T) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, scalar, &result)
				return s.Store(result)
			})
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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, scalar T, handle uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: ser.DataType}
				fn(ser, scalar, &result)
				return s.Store(result)
			})
	}
	stl.MustExport(rt, "series", "element_radd_"+suffix,
		func(_ context.Context, scalar T, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.addScalar(ser, scalar, &result)
			return s.Store(result)
		})
	stl.MustExport(rt, "series", "element_rmul_"+suffix,
		func(_ context.Context, scalar T, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.mulScalar(ser, scalar, &result)
			return s.Store(result)
		})

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
		stl.MustExport(rt, "series", entry.name+suffix,
			func(_ context.Context, handle uint32, scalar T) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: telem.Uint8T}
				fn(ser, scalar, &result)
				return s.Store(result)
			})
	}

	bindSeriesOps(rt, s, suffix, ops)
	bindCompareOps(rt, s, suffix, ops)
	if ops.negate != nil {
		bindNegate(rt, s, suffix, ops.negate)
	}
}

func bindU8(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindI32Type[uint8](rt, s, "u8", seriesOps[uint8]{
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

func bindU16(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindI32Type[uint16](rt, s, "u16", seriesOps[uint16]{
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

func bindU32(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindI32Type[uint32](rt, s, "u32", seriesOps[uint32]{
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

func bindI8(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindI32Type[int8](rt, s, "i8", seriesOps[int8]{
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

func bindI16(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindI32Type[int16](rt, s, "i16", seriesOps[int16]{
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

func bindI32(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindI32Type[int32](rt, s, "i32", seriesOps[int32]{
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

func bindU64(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindI64Type[uint64](rt, s, "u64", telem.Uint64T, seriesOps[uint64]{
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

func bindI64(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindI64Type[int64](rt, s, "i64", telem.Int64T, seriesOps[int64]{
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

func bindF32(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindFloatType[float32](rt, s, "f32", telem.Float32T, seriesOps[float32]{
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

func bindF64(rt stl.HostRuntime, s *state.SeriesHandleStore) {
	bindFloatType[float64](rt, s, "f64", telem.Float64T, seriesOps[float64]{
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

var _ stl.Module = (*Module)(nil)
