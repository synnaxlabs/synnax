// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"fmt"

	"github.com/synnaxlabs/slate/compiler/wasm"
)

// ImportIndex tracks the indices of all host functions that the runtime must provide.
// This defines the contract between compiled Slate WASM modules and the host runtime.
type ImportIndex struct {
	// Channel operations - per-type functions for type safety
	ChannelRead         map[string]uint32 // type suffix -> function index
	ChannelWrite        map[string]uint32
	ChannelBlockingRead map[string]uint32

	// Series operations - handle-based for memory isolation
	SeriesCreateEmpty map[string]uint32
	SeriesSetElement  map[string]uint32
	SeriesIndex       map[string]uint32
	SeriesLen         uint32
	SeriesSlice       uint32

	// Series arithmetic - per-type for performance
	SeriesElementAdd map[string]uint32
	SeriesElementMul map[string]uint32
	SeriesElementSub map[string]uint32
	SeriesElementDiv map[string]uint32
	SeriesSeriesAdd  map[string]uint32
	SeriesSeriesMul  map[string]uint32
	SeriesSeriesSub  map[string]uint32
	SeriesSeriesDiv  map[string]uint32

	// Series comparison - returns series u8
	SeriesCompareGT map[string]uint32
	SeriesCompareLT map[string]uint32
	SeriesCompareGE map[string]uint32
	SeriesCompareLE map[string]uint32
	SeriesCompareEQ map[string]uint32
	SeriesCompareNE map[string]uint32

	// State persistence - for stateful variables
	StateLoad  map[string]uint32
	StateStore map[string]uint32

	// String operations
	StringFromLiteral uint32
	StringLen         uint32
	StringEqual       uint32

	// Built-in functions
	Now uint32
	Len uint32 // For series length

	// Math operations (for exponentiation)
	MathPowF32 uint32
	MathPowF64 uint32

	// Error handling
	Panic uint32
}

// NewImportIndex creates a new import index with initialized maps
func NewImportIndex() *ImportIndex {
	return &ImportIndex{
		ChannelRead:         make(map[string]uint32),
		ChannelWrite:        make(map[string]uint32),
		ChannelBlockingRead: make(map[string]uint32),
		SeriesCreateEmpty:   make(map[string]uint32),
		SeriesSetElement:    make(map[string]uint32),
		SeriesIndex:         make(map[string]uint32),
		SeriesElementAdd:    make(map[string]uint32),
		SeriesElementMul:    make(map[string]uint32),
		SeriesElementSub:    make(map[string]uint32),
		SeriesElementDiv:    make(map[string]uint32),
		SeriesSeriesAdd:     make(map[string]uint32),
		SeriesSeriesMul:     make(map[string]uint32),
		SeriesSeriesSub:     make(map[string]uint32),
		SeriesSeriesDiv:     make(map[string]uint32),
		SeriesCompareGT:     make(map[string]uint32),
		SeriesCompareLT:     make(map[string]uint32),
		SeriesCompareGE:     make(map[string]uint32),
		SeriesCompareLE:     make(map[string]uint32),
		SeriesCompareEQ:     make(map[string]uint32),
		SeriesCompareNE:     make(map[string]uint32),
		StateLoad:           make(map[string]uint32),
		StateStore:          make(map[string]uint32),
	}
}

// SetupImports registers all host function imports with the WASM module.
// This defines the complete host interface that runtimes must implement.
func SetupImports(m *wasm.Module) *ImportIndex {
	idx := NewImportIndex()

	// Define the types we support
	intTypes := []string{"i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64"}
	floatTypes := []string{"f32", "f64"}
	allNumericTypes := append(intTypes, floatTypes...)

	// Register channel operations for each type
	for _, typ := range allNumericTypes {
		setupChannelOps(m, idx, typ)
	}
	setupChannelOps(m, idx, "string")

	// Register series operations for each type
	for _, typ := range allNumericTypes {
		setupSeriesOps(m, idx, typ)
	}

	// Register state operations for each type
	for _, typ := range allNumericTypes {
		setupStateOps(m, idx, typ)
	}
	setupStateOps(m, idx, "string")

	// Register type-agnostic operations
	setupGenericOps(m, idx)

	return idx
}

// setupChannelOps registers channel operations for a specific type
func setupChannelOps(m *wasm.Module, idx *ImportIndex, typ string) {
	wasmType := typeToWASM(typ)

	// Non-blocking read
	funcName := fmt.Sprintf("channel_read_%s", typ)
	idx.ChannelRead[typ] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // channel ID
		Results: []wasm.ValueType{wasmType}, // value or handle
	})

	// Write
	funcName = fmt.Sprintf("channel_write_%s", typ)
	idx.ChannelWrite[typ] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasmType}, // channel ID, value
		Results: []wasm.ValueType{},
	})

	// Blocking read
	funcName = fmt.Sprintf("channel_blocking_read_%s", typ)
	idx.ChannelBlockingRead[typ] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // channel ID
		Results: []wasm.ValueType{wasmType}, // value or handle
	})
}

// setupSeriesOps registers series operations for a specific type
func setupSeriesOps(m *wasm.Module, idx *ImportIndex, typ string) {
	wasmType := typeToWASM(typ)

	// Create empty series
	funcName := fmt.Sprintf("series_create_empty_%s", typ)
	idx.SeriesCreateEmpty[typ] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // length
		Results: []wasm.ValueType{wasm.I32}, // series handle
	})

	// Set element
	funcName = fmt.Sprintf("series_set_element_%s", typ)
	idx.SeriesSetElement[typ] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType}, // series, index, value
		Results: []wasm.ValueType{},
	})

	// Resolve element (indexing)
	funcName = fmt.Sprintf("series_index_%s", typ)
	idx.SeriesIndex[typ] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // series, index
		Results: []wasm.ValueType{wasmType},
	})

	// Arithmetic operations
	setupSeriesArithmetic(m, idx, typ, wasmType)

	// Comparison operations (only for numeric types)
	setupSeriesComparison(m, idx, typ)
}

// setupSeriesArithmetic registers arithmetic operations for series
func setupSeriesArithmetic(m *wasm.Module, idx *ImportIndex, typ string, wasmType wasm.ValueType) {
	// Scalar operations
	ops := []struct {
		name string
		idx  *map[string]uint32
	}{
		{"add", &idx.SeriesElementAdd},
		{"mul", &idx.SeriesElementMul},
		{"sub", &idx.SeriesElementSub},
		{"div", &idx.SeriesElementDiv},
	}

	for _, op := range ops {
		funcName := fmt.Sprintf("series_element_%s_%s", op.name, typ)
		(*op.idx)[typ] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasmType}, // series, scalar
			Results: []wasm.ValueType{wasm.I32},           // new series
		})
	}

	// Series-to-series operations
	seriesOps := []struct {
		name string
		idx  *map[string]uint32
	}{
		{"add", &idx.SeriesSeriesAdd},
		{"mul", &idx.SeriesSeriesMul},
		{"sub", &idx.SeriesSeriesSub},
		{"div", &idx.SeriesSeriesDiv},
	}

	for _, op := range seriesOps {
		funcName := fmt.Sprintf("series_series_%s_%s", op.name, typ)
		(*op.idx)[typ] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // series1, series2
			Results: []wasm.ValueType{wasm.I32},           // new series
		})
	}
}

// setupSeriesComparison registers comparison operations for series
func setupSeriesComparison(m *wasm.Module, idx *ImportIndex, typ string) {
	ops := []struct {
		name string
		idx  *map[string]uint32
	}{
		{"gt", &idx.SeriesCompareGT},
		{"lt", &idx.SeriesCompareLT},
		{"ge", &idx.SeriesCompareGE},
		{"le", &idx.SeriesCompareLE},
		{"eq", &idx.SeriesCompareEQ},
		{"ne", &idx.SeriesCompareNE},
	}

	for _, op := range ops {
		funcName := fmt.Sprintf("series_compare_%s_%s", op.name, typ)
		(*op.idx)[typ] = m.AddImport("env", funcName, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // series1, series2 or series, scalar
			Results: []wasm.ValueType{wasm.I32},           // series u8 (boolean mask)
		})
	}
}

// setupStateOps registers state persistence operations
func setupStateOps(m *wasm.Module, idx *ImportIndex, typ string) {
	wasmType := typeToWASM(typ)

	// Load state
	funcName := fmt.Sprintf("state_load_%s", typ)
	idx.StateLoad[typ] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // task ID, var ID
		Results: []wasm.ValueType{wasmType},
	})

	// Store state
	funcName = fmt.Sprintf("state_store_%s", typ)
	idx.StateStore[typ] = m.AddImport("env", funcName, wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType}, // task ID, var ID, value
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

	idx.StringLen = m.AddImport("env", "string_len", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // string handle
		Results: []wasm.ValueType{wasm.I32}, // length
	})

	idx.StringEqual = m.AddImport("env", "string_equal", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // string1, string2
		Results: []wasm.ValueType{wasm.I32},           // u8 result (0 or 1)
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

	// Error handling
	idx.Panic = m.AddImport("env", "panic", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32}, // ptr, len
		Results: []wasm.ValueType{},
	})
}

// typeToWASM converts a Slate type suffix to a WASM value type
func typeToWASM(typeSuffix string) wasm.ValueType {
	switch typeSuffix {
	case "i8", "i16", "i32", "u8", "u16", "u32":
		return wasm.I32 // WASM uses i32 for all <32-bit integers
	case "i64", "u64":
		return wasm.I64
	case "f32":
		return wasm.F32
	case "f64":
		return wasm.F64
	case "string":
		return wasm.I32 // String handle
	default:
		return wasm.I32 // Default for handles
	}
}
