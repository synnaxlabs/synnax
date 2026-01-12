// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:generate go run gen/main.go

package bindings

import (
	"fmt"

	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
)

// ImportIndex tracks the indices of all host functions that the runtime must provide.
// This defines the contract between compiled arc WASM modules and the host runtime.
type ImportIndex struct {
	// Channel operations - per-type functions for type safety
	ChannelRead  map[string]uint32 // type suffix -> function index
	ChannelWrite map[string]uint32

	// Series operations - handle-based for memory isolation
	SeriesCreateEmpty map[string]uint32
	SeriesSetElement  map[string]uint32
	SeriesIndex       map[string]uint32
	SeriesLen         uint32
	SeriesSlice       uint32

	// Series arithmetic - per-type for performance
	SeriesElementAdd  map[string]uint32
	SeriesElementMul  map[string]uint32
	SeriesElementSub  map[string]uint32
	SeriesElementDiv  map[string]uint32
	SeriesElementMod  map[string]uint32
	SeriesElementRSub map[string]uint32 // reverse subtract: scalar - series
	SeriesElementRDiv map[string]uint32 // reverse divide: scalar / series
	SeriesSeriesAdd   map[string]uint32
	SeriesSeriesMul   map[string]uint32
	SeriesSeriesSub   map[string]uint32
	SeriesSeriesDiv   map[string]uint32
	SeriesSeriesMod   map[string]uint32

	// Series comparison - returns series u8
	SeriesCompareGT map[string]uint32
	SeriesCompareLT map[string]uint32
	SeriesCompareGE map[string]uint32
	SeriesCompareLE map[string]uint32
	SeriesCompareEQ map[string]uint32
	SeriesCompareNE map[string]uint32

	// Series scalar comparison - returns series u8
	SeriesCompareGTScalar map[string]uint32
	SeriesCompareLTScalar map[string]uint32
	SeriesCompareGEScalar map[string]uint32
	SeriesCompareLEScalar map[string]uint32
	SeriesCompareEQScalar map[string]uint32
	SeriesCompareNEScalar map[string]uint32

	// Series unary operations
	SeriesNegate map[string]uint32 // For signed types (f64, f32, i64, i32, i16, i8)
	SeriesNotU8  uint32            // Logical NOT for boolean series

	// State persistence - for stateful variables
	StateLoad  map[string]uint32
	StateStore map[string]uint32

	// State persistence for series - uses handles (i32) instead of actual values
	StateLoadSeries  map[string]uint32
	StateStoreSeries map[string]uint32

	// String operations
	StringFromLiteral uint32
	StringConcat      uint32
	StringEqual       uint32
	StringLen         uint32

	// Built-in functions
	Now uint32
	Len uint32 // For series length

	// Math operations (for exponentiation)
	MathPowF32 uint32
	MathPowF64 uint32
	// Integer power operations
	MathIntPowU8  uint32
	MathIntPowU16 uint32
	MathIntPowU32 uint32
	MathIntPowU64 uint32
	MathIntPowI8  uint32
	MathIntPowI16 uint32
	MathIntPowI32 uint32
	MathIntPowI64 uint32

	// Error handling
	Panic uint32
}

// NewImportIndex creates a new import index with initialized maps
func NewImportIndex() *ImportIndex {
	return &ImportIndex{
		ChannelRead:           make(map[string]uint32),
		ChannelWrite:          make(map[string]uint32),
		SeriesCreateEmpty:     make(map[string]uint32),
		SeriesSetElement:      make(map[string]uint32),
		SeriesIndex:           make(map[string]uint32),
		SeriesElementAdd:      make(map[string]uint32),
		SeriesElementMul:      make(map[string]uint32),
		SeriesElementSub:      make(map[string]uint32),
		SeriesElementDiv:      make(map[string]uint32),
		SeriesElementMod:      make(map[string]uint32),
		SeriesElementRSub:     make(map[string]uint32),
		SeriesElementRDiv:     make(map[string]uint32),
		SeriesSeriesAdd:       make(map[string]uint32),
		SeriesSeriesMul:       make(map[string]uint32),
		SeriesSeriesSub:       make(map[string]uint32),
		SeriesSeriesDiv:       make(map[string]uint32),
		SeriesSeriesMod:       make(map[string]uint32),
		SeriesCompareGT:       make(map[string]uint32),
		SeriesCompareLT:       make(map[string]uint32),
		SeriesCompareGE:       make(map[string]uint32),
		SeriesCompareLE:       make(map[string]uint32),
		SeriesCompareEQ:       make(map[string]uint32),
		SeriesCompareNE:       make(map[string]uint32),
		SeriesCompareGTScalar: make(map[string]uint32),
		SeriesCompareLTScalar: make(map[string]uint32),
		SeriesCompareGEScalar: make(map[string]uint32),
		SeriesCompareLEScalar: make(map[string]uint32),
		SeriesCompareEQScalar: make(map[string]uint32),
		SeriesCompareNEScalar: make(map[string]uint32),
		SeriesNegate:          make(map[string]uint32),
		StateLoad:             make(map[string]uint32),
		StateStore:            make(map[string]uint32),
		StateLoadSeries:       make(map[string]uint32),
		StateStoreSeries:      make(map[string]uint32),
	}
}

// SetupImports registers all host function imports with the WASM module.
// This defines the complete host interface that runtimes must implement.
func SetupImports(m *wasm.Module) *ImportIndex {
	idx := NewImportIndex()
	// register channel operations for each type
	for _, typ := range types.Numerics {
		setupChannelOps(m, idx, typ)
	}
	setupChannelOps(m, idx, types.String())
	for _, typ := range types.Numerics {
		setupSeriesOps(m, idx, typ)
	}
	setupSeriesUnaryOps(m, idx)
	for _, typ := range types.Numerics {
		setupStateOps(m, idx, typ)
	}
	setupStateOps(m, idx, types.String())
	setupGenericOps(m, idx)
	return idx
}

// setupChannelOps registers channel operations for a specific type
func setupChannelOps(m *wasm.Module, idx *ImportIndex, t types.Type) {
	wasmType := wasm.ConvertType(t)
	// Non-blocking read
	funcName := fmt.Sprintf("channel_read_%s", t)
	idx.ChannelRead[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // channel ID
		Results: []wasm.ValueType{wasmType}, // value or handle
	})

	funcName = fmt.Sprintf("channel_write_%s", t)
	idx.ChannelWrite[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasmType}, // channel ID, value
		Results: []wasm.ValueType{},
	})
}

// setupSeriesOps registers series operations for a specific type
func setupSeriesOps(m *wasm.Module, idx *ImportIndex, t types.Type) {
	wasmType := wasm.ConvertType(t)

	// Create empty series
	funcName := fmt.Sprintf("series_create_empty_%s", t)
	idx.SeriesCreateEmpty[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // length
		Results: []wasm.ValueType{wasm.I32}, // series handle
	})

	// Set element - returns handle for stack-based chaining
	funcName = fmt.Sprintf("series_set_element_%s", t)
	idx.SeriesSetElement[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType}, // series, index, value
		Results: []wasm.ValueType{wasm.I32},                     // returns series handle
	})

	// Resolve element (indexing)
	funcName = fmt.Sprintf("series_index_%s", t)
	idx.SeriesIndex[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // series, index
		Results: []wasm.ValueType{wasmType},
	})

	// Arithmetic operations
	setupSeriesArithmetic(m, idx, t, wasmType)

	// Comparison operations (only for numeric types)
	setupSeriesComparison(m, idx, t)

	// Series state operations (must come after comparison for index alignment)
	setupSeriesStateOps(m, idx, t)
}

// setupSeriesArithmetic registers arithmetic operations for series
func setupSeriesArithmetic(m *wasm.Module, idx *ImportIndex, typ types.Type, wasmType wasm.ValueType) {
	// Scalar operations: series op scalar -> (handle, scalar)
	ops := []struct {
		idx  *map[string]uint32
		name string
	}{
		{name: "add", idx: &idx.SeriesElementAdd},
		{name: "mul", idx: &idx.SeriesElementMul},
		{name: "sub", idx: &idx.SeriesElementSub},
		{name: "div", idx: &idx.SeriesElementDiv},
		{name: "mod", idx: &idx.SeriesElementMod},
	}

	for _, op := range ops {
		funcName := fmt.Sprintf("series_element_%s_%s", op.name, typ)
		(*op.idx)[typ.String()] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasmType}, // series, scalar
			Results: []wasm.ValueType{wasm.I32},           // new series
		})
	}

	// Reverse operations: scalar op series -> (scalar, handle)
	// For `scalar - series`, stack is [scalar, handle], so signature is (scalar, handle)
	reverseOps := []struct {
		name string
		idx  *map[string]uint32
	}{
		{"rsub", &idx.SeriesElementRSub},
		{"rdiv", &idx.SeriesElementRDiv},
	}

	for _, op := range reverseOps {
		funcName := fmt.Sprintf("series_element_%s_%s", op.name, typ)
		(*op.idx)[typ.String()] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasmType, wasm.I32}, // scalar, series
			Results: []wasm.ValueType{wasm.I32},           // new series
		})
	}

	// Series-to-series operations
	seriesOps := []struct {
		idx  *map[string]uint32
		name string
	}{
		{name: "add", idx: &idx.SeriesSeriesAdd},
		{name: "mul", idx: &idx.SeriesSeriesMul},
		{name: "sub", idx: &idx.SeriesSeriesSub},
		{name: "div", idx: &idx.SeriesSeriesDiv},
		{name: "mod", idx: &idx.SeriesSeriesMod},
	}

	for _, op := range seriesOps {
		funcName := fmt.Sprintf("series_series_%s_%s", op.name, typ)
		(*op.idx)[typ.String()] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // series1, series2
			Results: []wasm.ValueType{wasm.I32},           // new series
		})
	}
}

// setupSeriesComparison registers comparison operations for series
func setupSeriesComparison(m *wasm.Module, idx *ImportIndex, typ types.Type) {
	wasmType := wasm.ConvertType(typ)

	// Series-to-series comparison operations
	ops := []struct {
		idx  *map[string]uint32
		name string
	}{
		{name: "gt", idx: &idx.SeriesCompareGT},
		{name: "lt", idx: &idx.SeriesCompareLT},
		{name: "ge", idx: &idx.SeriesCompareGE},
		{name: "le", idx: &idx.SeriesCompareLE},
		{name: "eq", idx: &idx.SeriesCompareEQ},
		{name: "ne", idx: &idx.SeriesCompareNE},
	}

	for _, op := range ops {
		funcName := fmt.Sprintf("series_compare_%s_%s", op.name, typ)
		(*op.idx)[typ.String()] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // series1, series2
			Results: []wasm.ValueType{wasm.I32},           // series u8 (boolean mask)
		})
	}

	// Series-to-scalar comparison operations
	scalarOps := []struct {
		name string
		idx  *map[string]uint32
	}{
		{"gt_scalar", &idx.SeriesCompareGTScalar},
		{"lt_scalar", &idx.SeriesCompareLTScalar},
		{"ge_scalar", &idx.SeriesCompareGEScalar},
		{"le_scalar", &idx.SeriesCompareLEScalar},
		{"eq_scalar", &idx.SeriesCompareEQScalar},
		{"ne_scalar", &idx.SeriesCompareNEScalar},
	}

	for _, op := range scalarOps {
		funcName := fmt.Sprintf("series_compare_%s_%s", op.name, typ)
		(*op.idx)[typ.String()] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasmType}, // series, scalar
			Results: []wasm.ValueType{wasm.I32},           // series u8 (boolean mask)
		})
	}
}

// setupSeriesUnaryOps registers unary operations for series (negate, not)
func setupSeriesUnaryOps(m *wasm.Module, idx *ImportIndex) {
	// Negate for signed types (floats and signed integers)
	signedTypes := []types.Type{
		types.F64(), types.F32(),
		types.I64(), types.I32(), types.I16(), types.I8(),
	}
	for _, typ := range signedTypes {
		funcName := fmt.Sprintf("series_negate_%s", typ)
		idx.SeriesNegate[typ.String()] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32}, // series handle
			Results: []wasm.ValueType{wasm.I32}, // new series handle
		})
	}

	// Logical NOT for boolean series (u8)
	idx.SeriesNotU8 = m.AddImport("env", "series_not_u8", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // series handle
		Results: []wasm.ValueType{wasm.I32}, // new series handle
	})
}

// setupStateOps registers state persistence operations
func setupStateOps(m *wasm.Module, idx *ImportIndex, t types.Type) {
	wasmType := wasm.ConvertType(t)

	// Load state (with initialization value)
	funcName := fmt.Sprintf("state_load_%s", t)
	idx.StateLoad[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType}, // func ID, var ID, init value
		Results: []wasm.ValueType{wasmType},
	})

	// Store state
	funcName = fmt.Sprintf("state_store_%s", t)
	idx.StateStore[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType}, // func ID, var ID, value
		Results: []wasm.ValueType{},
	})
}

// setupSeriesStateOps registers state persistence operations for series types.
// Series state uses handles (i32) for both input and output, not actual values.
func setupSeriesStateOps(m *wasm.Module, idx *ImportIndex, t types.Type) {
	// Load series state (with initialization handle)
	funcName := fmt.Sprintf("state_load_series_%s", t)
	idx.StateLoadSeries[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasm.I32}, // func ID, var ID, init handle
		Results: []wasm.ValueType{wasm.I32},                     // result handle
	})

	// Store series state
	funcName = fmt.Sprintf("state_store_series_%s", t)
	idx.StateStoreSeries[t.String()] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasm.I32}, // func ID, var ID, handle
		Results: []wasm.ValueType{},
	})
}

// setupGenericOps registers type-agnostic operations
func setupGenericOps(m *wasm.Module, idx *ImportIndex) {
	// Series operations
	idx.SeriesLen = m.AddImport("env", "series_len", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // series handle
		Results: []wasm.ValueType{wasm.I64}, // length
	})

	idx.SeriesSlice = m.AddImport("env", "series_slice", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasm.I32}, // series, start, end
		Results: []wasm.ValueType{wasm.I32},                     // new series handle
	})

	// String operations
	idx.StringFromLiteral = m.AddImport("env", "string_from_literal", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // ptr, len
		Results: []wasm.ValueType{wasm.I32},           // string handle
	})

	idx.StringConcat = m.AddImport("env", "string_concat", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // string1, string2
		Results: []wasm.ValueType{wasm.I32},           // new string handle
	})

	idx.StringEqual = m.AddImport("env", "string_equal", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // string1, string2
		Results: []wasm.ValueType{wasm.I32},           // u8 result (0 or 1)
	})

	idx.StringLen = m.AddImport("env", "string_len", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // string handle
		Results: []wasm.ValueType{wasm.I32}, // length
	})

	// Built-in functions
	idx.Now = m.AddImport("env", "now", wasm.FunctionType{
		Params:  []wasm.ValueType{},
		Results: []wasm.ValueType{wasm.I64}, // timestamp
	})

	idx.Len = m.AddImport("env", "len", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // series handle
		Results: []wasm.ValueType{wasm.I64}, // length
	})

	// Math operations
	idx.MathPowF32 = m.AddImport("env", "math_pow_f32", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.F32, wasm.F32}, // base, exponent
		Results: []wasm.ValueType{wasm.F32},
	})

	idx.MathPowF64 = m.AddImport("env", "math_pow_f64", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.F64, wasm.F64}, // base, exponent
		Results: []wasm.ValueType{wasm.F64},
	})

	// Integer power operations
	idx.MathIntPowU8 = m.AddImport("env", "math_pow_u8", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // base, exponent (u8 represented as i32)
		Results: []wasm.ValueType{wasm.I32},
	})

	idx.MathIntPowU16 = m.AddImport("env", "math_pow_u16", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // base, exponent (u16 represented as i32)
		Results: []wasm.ValueType{wasm.I32},
	})

	idx.MathIntPowU32 = m.AddImport("env", "math_pow_u32", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // base, exponent
		Results: []wasm.ValueType{wasm.I32},
	})

	idx.MathIntPowU64 = m.AddImport("env", "math_pow_u64", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I64, wasm.I64}, // base, exponent
		Results: []wasm.ValueType{wasm.I64},
	})

	idx.MathIntPowI8 = m.AddImport("env", "math_pow_i8", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // base, exponent (i8 represented as i32)
		Results: []wasm.ValueType{wasm.I32},
	})

	idx.MathIntPowI16 = m.AddImport("env", "math_pow_i16", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // base, exponent (i16 represented as i32)
		Results: []wasm.ValueType{wasm.I32},
	})

	idx.MathIntPowI32 = m.AddImport("env", "math_pow_i32", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // base, exponent
		Results: []wasm.ValueType{wasm.I32},
	})

	idx.MathIntPowI64 = m.AddImport("env", "math_pow_i64", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I64, wasm.I64}, // base, exponent
		Results: []wasm.ValueType{wasm.I64},
	})

	// Error handling
	idx.Panic = m.AddImport("env", "panic", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // ptr, len
		Results: []wasm.ValueType{},
	})
}
