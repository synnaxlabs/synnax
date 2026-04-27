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
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
	"github.com/tetratelabs/wazero"
)

var numConstraint = types.NumericConstraint()

func tv() types.Type { return types.Variable("T", &numConstraint) }

func polyFunc(inputs, outputs types.Params) types.Type {
	return types.Function(types.FunctionProperties{Inputs: inputs, Outputs: outputs})
}

var i32 = types.I32()
var i64 = types.I64()

var userSymbols = symbol.MapResolver{
	"len": {
		Name: "len",
		Kind: symbol.KindFunction,
		Exec: symbol.ExecWASM,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		}),
	},
}

func hostSym(name string, t types.Type) symbol.Symbol {
	return symbol.Symbol{Name: name, Kind: symbol.KindFunction, Exec: symbol.ExecWASM, Internal: true, Type: t}
}

var hostSymbols = symbol.MapResolver{
	"element_add":       hostSym("element_add", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"element_sub":       hostSym("element_sub", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"element_mul":       hostSym("element_mul", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"element_div":       hostSym("element_div", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"element_mod":       hostSym("element_mod", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"element_radd":      hostSym("element_radd", polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"element_rsub":      hostSym("element_rsub", polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"element_rmul":      hostSym("element_rmul", polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"element_rdiv":      hostSym("element_rdiv", polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"element_rmod":      hostSym("element_rmod", polyFunc(types.Params{{Name: "scalar", Type: tv()}, {Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"series_add":        hostSym("series_add", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"series_sub":        hostSym("series_sub", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"series_mul":        hostSym("series_mul", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"series_div":        hostSym("series_div", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"series_mod":        hostSym("series_mod", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"compare_gt":        hostSym("compare_gt", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"compare_lt":        hostSym("compare_lt", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"compare_ge":        hostSym("compare_ge", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"compare_le":        hostSym("compare_le", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"compare_eq":        hostSym("compare_eq", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"compare_ne":        hostSym("compare_ne", polyFunc(types.Params{{Name: "a", Type: i32}, {Name: "b", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"compare_gt_scalar": hostSym("compare_gt_scalar", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"compare_lt_scalar": hostSym("compare_lt_scalar", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"compare_ge_scalar": hostSym("compare_ge_scalar", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"compare_le_scalar": hostSym("compare_le_scalar", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"compare_eq_scalar": hostSym("compare_eq_scalar", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"compare_ne_scalar": hostSym("compare_ne_scalar", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "scalar", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"create_empty":      hostSym("create_empty", polyFunc(types.Params{{Name: "len", Type: i32}}, types.Params{{Name: "handle", Type: i32}})),
	"set_element":       hostSym("set_element", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "idx", Type: i32}, {Name: "value", Type: tv()}}, types.Params{{Name: "result", Type: i32}})),
	"index":             hostSym("index", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "idx", Type: i32}}, types.Params{{Name: "value", Type: tv()}})),
	"negate":            hostSym("negate", polyFunc(types.Params{{Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"not_u8":            hostSym("not_u8", polyFunc(types.Params{{Name: "handle", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
	"len":               hostSym("len", polyFunc(types.Params{{Name: "handle", Type: i32}}, types.Params{{Name: "length", Type: i64}})),
	"slice":             hostSym("slice", polyFunc(types.Params{{Name: "handle", Type: i32}, {Name: "start", Type: i32}, {Name: "end", Type: i32}}, types.Params{{Name: "result", Type: i32}})),
}

// moduleMembers contains symbols accessible via the "series" module namespace.
var moduleMembers = symbol.MapResolver{
	"len": {
		Name:     "len",
		Kind:     symbol.KindFunction,
		Exec:     symbol.ExecWASM,
		Internal: true,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "handle", Type: types.I32()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		}),
	},
}

var SymbolResolver = symbol.CompoundResolver{
	userSymbols,
	&symbol.ModuleResolver{Name: "series", Members: moduleMembers},
	&symbol.ModuleResolver{Name: "series", Members: hostSymbols},
}

type Module struct {
	series *ProgramState
}

func NewModule(
	ctx context.Context,
	s *ProgramState,
	rat wazero.Runtime,
) (*Module, error) {
	if rat == nil {
		return &Module{series: s}, nil
	}
	builder := rat.NewHostModuleBuilder("series")
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
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32) uint64 {
			if ser, ok := s.Get(handle); ok {
				return uint64(ser.Len())
			}
			return 0
		}).Export("len")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32, start uint32, end uint32) uint32 {
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
			result := telem.Series{DataType: telem.Uint8T}
			op.NotU8(ser, &result)
			return s.Store(result)
		}).Export("not_u8")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return &Module{series: s}, nil
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
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	dt := ops.dt
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, length uint32) uint32 {
			return s.Store(telem.MakeSeries(dt, int(length)))
		}).Export("create_empty_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32, index uint32, value uint32) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), T(value))
				}
			}
			return handle
		}).Export("set_element_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32, index uint32) uint32 {
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
			WithFunc(func(_ context.Context, handle uint32, scalar uint32) uint32 {
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
			WithFunc(func(_ context.Context, scalar uint32, handle uint32) uint32 {
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
		WithFunc(func(_ context.Context, scalar uint32, handle uint32) uint32 {
			ser, ok := s.Get(handle)
			if !ok {
				return 0
			}
			result := telem.Series{DataType: ser.DataType}
			ops.addScalar(ser, T(scalar), &result)
			return s.Store(result)
		}).Export("element_radd_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, scalar uint32, handle uint32) uint32 {
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
			WithFunc(func(_ context.Context, handle uint32, scalar uint32) uint32 {
				ser, ok := s.Get(handle)
				if !ok {
					return 0
				}
				result := telem.Series{DataType: telem.Uint8T}
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
			WithFunc(func(_ context.Context, h1 uint32, h2 uint32) uint32 {
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
			WithFunc(func(_ context.Context, h1 uint32, h2 uint32) uint32 {
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
) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, length uint32) uint32 {
			return s.Store(telem.MakeSeries(dt, int(length)))
		}).Export("create_empty_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32, index uint32, value uint64) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), T(value))
				}
			}
			return handle
		}).Export("set_element_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32, index uint32) uint64 {
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
				result := telem.Series{DataType: telem.Uint8T}
				fn(ser, T(scalar), &result)
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

// bindFloatType handles f32 or f64.
func bindFloatType[T float32 | float64](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	suffix string,
	dt telem.DataType,
	ops seriesOps[T],
) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, length uint32) uint32 {
			return s.Store(telem.MakeSeries(dt, int(length)))
		}).Export("create_empty_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32, index uint32, value T) uint32 {
			if ser, ok := s.Get(handle); ok {
				if int64(index) < ser.Len() {
					telem.SetValueAt[T](ser, int(index), value)
				}
			}
			return handle
		}).Export("set_element_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32, index uint32) T {
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
				result := telem.Series{DataType: telem.Uint8T}
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

func bindU8(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
	})
}

func bindU16(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
	})
}

func bindU32(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
	})
}

func bindI8(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
	})
}

func bindI16(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
	})
}

func bindI32(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
	})
}

func bindU64(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
	})
}

func bindI64(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
	})
}

func bindF32(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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

func bindF64(builder wazero.HostModuleBuilder, s *ProgramState) wazero.HostModuleBuilder {
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
