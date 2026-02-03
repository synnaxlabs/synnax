// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bindings

import (
	"fmt"

	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
)

// Module names for the Arc module system.
// These define the WASM import module names that the runtime must provide.
const (
	ModuleCore   = "arc.core"   // Channel I/O, panic
	ModuleSeries = "arc.series" // Series operations
	ModuleState  = "arc.state"  // Stateful variable persistence
	ModuleString = "arc.string" // String operations
	ModuleMath   = "math"       // Math stdlib
	ModuleTime   = "time"       // Time stdlib
)

// SetupImports registers all host function imports with the WASM module.
// Returns the ImportRegistry which can be used for name-based lookups during compilation.
func SetupImports(m *wasm.Module) *ImportRegistry {
	r := NewImportRegistry()

	// Register all modules
	RegisterCoreImports(r)
	RegisterSeriesImports(r)
	RegisterStateImports(r)
	RegisterStringImports(r)
	RegisterMathImports(r)
	RegisterTimeImports(r)

	// Write imports to module
	r.WriteToModule(m)

	return r
}

// RegisterCoreImports registers arc.core module functions (channel operations, panic).
func RegisterCoreImports(r *ImportRegistry) {
	// Channel operations for each type
	for _, typ := range types.Numerics {
		registerChannelOps(r, typ)
	}
	registerChannelOps(r, types.String())

	// Panic
	r.Register(ModuleCore, "panic", wasm.FunctionType{
		Params: []wasm.ValueType{wasm.I32, wasm.I32}, // ptr, len
	})
}

func registerChannelOps(r *ImportRegistry, t types.Type) {
	wasmType := wasm.ConvertType(t)

	r.Register(ModuleCore, fmt.Sprintf("channel_read_%s", t), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32},
		Results: []wasm.ValueType{wasmType},
	})

	r.Register(ModuleCore, fmt.Sprintf("channel_write_%s", t), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasmType},
		Results: []wasm.ValueType{},
	})
}

// RegisterSeriesImports registers arc.series module functions.
func RegisterSeriesImports(r *ImportRegistry) {
	for _, typ := range types.Numerics {
		registerSeriesOps(r, typ)
	}
	registerSeriesUnaryOps(r)
}

func registerSeriesOps(r *ImportRegistry, t types.Type) {
	wasmType := wasm.ConvertType(t)
	typeName := t.String()

	// Create empty series
	r.Register(ModuleSeries, fmt.Sprintf("create_empty_%s", typeName), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32},
		Results: []wasm.ValueType{wasm.I32},
	})

	// Set element
	r.Register(ModuleSeries, fmt.Sprintf("set_element_%s", typeName), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType},
		Results: []wasm.ValueType{wasm.I32},
	})

	// Index element
	r.Register(ModuleSeries, fmt.Sprintf("index_%s", typeName), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32},
		Results: []wasm.ValueType{wasmType},
	})

	// Arithmetic operations
	registerSeriesArithmetic(r, t, wasmType)

	// Comparison operations
	registerSeriesComparison(r, t, wasmType)

	// Series state operations
	registerSeriesStateOps(r, t)
}

func registerSeriesArithmetic(r *ImportRegistry, typ types.Type, wasmType wasm.ValueType) {
	typeName := typ.String()

	// Scalar operations: series op scalar
	for _, op := range []string{"add", "mul", "sub", "div", "mod"} {
		r.Register(ModuleSeries, fmt.Sprintf("element_%s_%s", op, typeName), wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasmType},
			Results: []wasm.ValueType{wasm.I32},
		})
	}

	// Reverse operations: scalar op series
	for _, op := range []string{"rsub", "rdiv", "radd", "rmul", "rmod"} {
		r.Register(ModuleSeries, fmt.Sprintf("element_%s_%s", op, typeName), wasm.FunctionType{
			Params:  []wasm.ValueType{wasmType, wasm.I32},
			Results: []wasm.ValueType{wasm.I32},
		})
	}

	// Series-to-series operations
	for _, op := range []string{"add", "mul", "sub", "div", "mod"} {
		r.Register(ModuleSeries, fmt.Sprintf("series_%s_%s", op, typeName), wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasm.I32},
			Results: []wasm.ValueType{wasm.I32},
		})
	}
}

func registerSeriesComparison(r *ImportRegistry, typ types.Type, wasmType wasm.ValueType) {
	typeName := typ.String()

	// Series-to-series comparison
	for _, op := range []string{"gt", "lt", "ge", "le", "eq", "ne"} {
		r.Register(ModuleSeries, fmt.Sprintf("compare_%s_%s", op, typeName), wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasm.I32},
			Results: []wasm.ValueType{wasm.I32},
		})
	}

	// Series-to-scalar comparison
	for _, op := range []string{"gt", "lt", "ge", "le", "eq", "ne"} {
		r.Register(ModuleSeries, fmt.Sprintf("compare_%s_scalar_%s", op, typeName), wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32, wasmType},
			Results: []wasm.ValueType{wasm.I32},
		})
	}
}

func registerSeriesUnaryOps(r *ImportRegistry) {
	// Negate for signed types
	signedTypes := []types.Type{
		types.F64(), types.F32(),
		types.I64(), types.I32(), types.I16(), types.I8(),
	}
	for _, typ := range signedTypes {
		r.Register(ModuleSeries, fmt.Sprintf("negate_%s", typ), wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I32},
			Results: []wasm.ValueType{wasm.I32},
		})
	}

	// Logical NOT for u8 (boolean)
	r.Register(ModuleSeries, "not_u8", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32},
		Results: []wasm.ValueType{wasm.I32},
	})

	// Series length
	r.Register(ModuleSeries, "len", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32},
		Results: []wasm.ValueType{wasm.I64},
	})

	// Series slice
	r.Register(ModuleSeries, "slice", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasm.I32},
		Results: []wasm.ValueType{wasm.I32},
	})
}

func registerSeriesStateOps(r *ImportRegistry, t types.Type) {
	typeName := t.String()

	// Load series state
	r.Register(ModuleState, fmt.Sprintf("load_series_%s", typeName), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasm.I32},
		Results: []wasm.ValueType{wasm.I32},
	})

	// Store series state
	r.Register(ModuleState, fmt.Sprintf("store_series_%s", typeName), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasm.I32},
		Results: []wasm.ValueType{},
	})
}

// RegisterStateImports registers arc.state module functions.
func RegisterStateImports(r *ImportRegistry) {
	for _, typ := range types.Numerics {
		registerStateOps(r, typ)
	}
	registerStateOps(r, types.String())
}

func registerStateOps(r *ImportRegistry, t types.Type) {
	wasmType := wasm.ConvertType(t)
	typeName := t.String()

	// Load state
	r.Register(ModuleState, fmt.Sprintf("load_%s", typeName), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType},
		Results: []wasm.ValueType{wasmType},
	})

	// Store state
	r.Register(ModuleState, fmt.Sprintf("store_%s", typeName), wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType},
		Results: []wasm.ValueType{},
	})
}

// RegisterStringImports registers arc.string module functions.
func RegisterStringImports(r *ImportRegistry) {
	r.Register(ModuleString, "from_literal", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32},
		Results: []wasm.ValueType{wasm.I32},
	})

	r.Register(ModuleString, "concat", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32},
		Results: []wasm.ValueType{wasm.I32},
	})

	r.Register(ModuleString, "equal", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.I32},
		Results: []wasm.ValueType{wasm.I32},
	})

	r.Register(ModuleString, "len", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32},
		Results: []wasm.ValueType{wasm.I32},
	})
}

// RegisterMathImports registers math stdlib module functions.
func RegisterMathImports(r *ImportRegistry) {
	// f64 -> f64 functions
	for _, name := range []string{"sqrt", "sin", "cos", "tan", "asin", "acos", "atan",
		"abs", "floor", "ceil", "round", "exp", "log", "log10"} {
		r.Register(ModuleMath, name, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.F64},
			Results: []wasm.ValueType{wasm.F64},
		})
	}

	// (f64, f64) -> f64 functions
	for _, name := range []string{"pow", "min", "max", "atan2"} {
		r.Register(ModuleMath, name, wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.F64, wasm.F64},
			Results: []wasm.ValueType{wasm.F64},
		})
	}

	// Integer power operations (for the ^ operator)
	r.Register(ModuleMath, "pow_f32", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.F32, wasm.F32},
		Results: []wasm.ValueType{wasm.F32},
	})

	// Integer power for all integer types
	for _, typ := range []struct {
		name    string
		wasmTyp wasm.ValueType
	}{
		{"u8", wasm.I32}, {"u16", wasm.I32}, {"u32", wasm.I32}, {"u64", wasm.I64},
		{"i8", wasm.I32}, {"i16", wasm.I32}, {"i32", wasm.I32}, {"i64", wasm.I64},
	} {
		r.Register(ModuleMath, fmt.Sprintf("pow_%s", typ.name), wasm.FunctionType{
			Params:  []wasm.ValueType{typ.wasmTyp, typ.wasmTyp},
			Results: []wasm.ValueType{typ.wasmTyp},
		})
	}

	// Constants
	r.Register(ModuleMath, "pi", wasm.FunctionType{
		Results: []wasm.ValueType{wasm.F64},
	})
	r.Register(ModuleMath, "e", wasm.FunctionType{
		Results: []wasm.ValueType{wasm.F64},
	})
}

// RegisterTimeImports registers time stdlib module functions.
func RegisterTimeImports(r *ImportRegistry) {
	r.Register(ModuleTime, "now", wasm.FunctionType{
		Results: []wasm.ValueType{wasm.I64},
	})

	r.Register(ModuleTime, "elapsed", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I64},
		Results: []wasm.ValueType{wasm.I64},
	})
}

// Convenience lookup functions for common operations.
// These provide type-safe access to import indices.

// ChannelRead returns the index for channel_read_<type>.
func (r *ImportRegistry) ChannelRead(t types.Type) uint32 {
	return r.MustLookup(ModuleCore, fmt.Sprintf("channel_read_%s", t))
}

// ChannelWrite returns the index for channel_write_<type>.
func (r *ImportRegistry) ChannelWrite(t types.Type) uint32 {
	return r.MustLookup(ModuleCore, fmt.Sprintf("channel_write_%s", t))
}

// SeriesCreateEmpty returns the index for series_create_empty_<type>.
func (r *ImportRegistry) SeriesCreateEmpty(t types.Type) uint32 {
	return r.MustLookup(ModuleSeries, fmt.Sprintf("create_empty_%s", t))
}

// SeriesSetElement returns the index for series_set_element_<type>.
func (r *ImportRegistry) SeriesSetElement(t types.Type) uint32 {
	return r.MustLookup(ModuleSeries, fmt.Sprintf("set_element_%s", t))
}

// SeriesIndex returns the index for series_index_<type>.
func (r *ImportRegistry) SeriesIndex(t types.Type) uint32 {
	return r.MustLookup(ModuleSeries, fmt.Sprintf("index_%s", t))
}

// SeriesElementOp returns the index for series_element_<op>_<type>.
func (r *ImportRegistry) SeriesElementOp(op string, t types.Type) uint32 {
	return r.MustLookup(ModuleSeries, fmt.Sprintf("element_%s_%s", op, t))
}

// SeriesSeriesOp returns the index for series_series_<op>_<type>.
func (r *ImportRegistry) SeriesSeriesOp(op string, t types.Type) uint32 {
	return r.MustLookup(ModuleSeries, fmt.Sprintf("series_%s_%s", op, t))
}

// SeriesCompare returns the index for series_compare_<op>_<type>.
func (r *ImportRegistry) SeriesCompare(op string, t types.Type) uint32 {
	return r.MustLookup(ModuleSeries, fmt.Sprintf("compare_%s_%s", op, t))
}

// SeriesCompareScalar returns the index for series_compare_<op>_scalar_<type>.
func (r *ImportRegistry) SeriesCompareScalar(op string, t types.Type) uint32 {
	return r.MustLookup(ModuleSeries, fmt.Sprintf("compare_%s_scalar_%s", op, t))
}

// SeriesNegate returns the index for series_negate_<type>.
func (r *ImportRegistry) SeriesNegate(t types.Type) uint32 {
	return r.MustLookup(ModuleSeries, fmt.Sprintf("negate_%s", t))
}

// SeriesNotU8 returns the index for series_not_u8.
func (r *ImportRegistry) SeriesNotU8() uint32 {
	return r.MustLookup(ModuleSeries, "not_u8")
}

// SeriesLen returns the index for series_len.
func (r *ImportRegistry) SeriesLen() uint32 {
	return r.MustLookup(ModuleSeries, "len")
}

// SeriesSlice returns the index for series_slice.
func (r *ImportRegistry) SeriesSlice() uint32 {
	return r.MustLookup(ModuleSeries, "slice")
}

// StateLoad returns the index for state_load_<type>.
func (r *ImportRegistry) StateLoad(t types.Type) uint32 {
	return r.MustLookup(ModuleState, fmt.Sprintf("load_%s", t))
}

// StateStore returns the index for state_store_<type>.
func (r *ImportRegistry) StateStore(t types.Type) uint32 {
	return r.MustLookup(ModuleState, fmt.Sprintf("store_%s", t))
}

// StateLoadSeries returns the index for state_load_series_<type>.
func (r *ImportRegistry) StateLoadSeries(t types.Type) uint32 {
	return r.MustLookup(ModuleState, fmt.Sprintf("load_series_%s", t))
}

// StateStoreSeries returns the index for state_store_series_<type>.
func (r *ImportRegistry) StateStoreSeries(t types.Type) uint32 {
	return r.MustLookup(ModuleState, fmt.Sprintf("store_series_%s", t))
}

// StringFromLiteral returns the index for string_from_literal.
func (r *ImportRegistry) StringFromLiteral() uint32 {
	return r.MustLookup(ModuleString, "from_literal")
}

// StringConcat returns the index for string_concat.
func (r *ImportRegistry) StringConcat() uint32 {
	return r.MustLookup(ModuleString, "concat")
}

// StringEqual returns the index for string_equal.
func (r *ImportRegistry) StringEqual() uint32 {
	return r.MustLookup(ModuleString, "equal")
}

// StringLen returns the index for string_len.
func (r *ImportRegistry) StringLen() uint32 {
	return r.MustLookup(ModuleString, "len")
}

// MathPow returns the index for math.pow (f64 version).
func (r *ImportRegistry) MathPow() uint32 {
	return r.MustLookup(ModuleMath, "pow")
}

// MathPowF32 returns the index for math.pow_f32.
func (r *ImportRegistry) MathPowF32() uint32 {
	return r.MustLookup(ModuleMath, "pow_f32")
}

// MathPowInt returns the index for math.pow_<type> for integer types.
func (r *ImportRegistry) MathPowInt(t types.Type) uint32 {
	return r.MustLookup(ModuleMath, fmt.Sprintf("pow_%s", t))
}

// TimeNow returns the index for time.now.
func (r *ImportRegistry) TimeNow() uint32 {
	return r.MustLookup(ModuleTime, "now")
}

// Panic returns the index for panic.
func (r *ImportRegistry) Panic() uint32 {
	return r.MustLookup(ModuleCore, "panic")
}

// Len returns the index for the builtin len function (series length).
func (r *ImportRegistry) Len() uint32 {
	return r.MustLookup(ModuleSeries, "len")
}

// operatorToOpName converts an operator symbol to its import name.
func operatorToOpName(op string) string {
	switch op {
	case "+":
		return "add"
	case "-":
		return "sub"
	case "*":
		return "mul"
	case "/":
		return "div"
	case "%":
		return "mod"
	default:
		panic(fmt.Sprintf("unknown arithmetic operator: %s", op))
	}
}

// comparisonOpName converts a comparison operator to its import name.
func comparisonOpName(op string) string {
	switch op {
	case ">":
		return "gt"
	case "<":
		return "lt"
	case ">=":
		return "ge"
	case "<=":
		return "le"
	case "==":
		return "eq"
	case "!=":
		return "ne"
	default:
		panic(fmt.Sprintf("unknown comparison operator: %s", op))
	}
}

// GetSeriesArithmetic returns the import index for series arithmetic operations.
// For scalar operations (series op scalar), uses element_<op>_<type>.
// For series-to-series operations, uses series_<op>_<type>.
func (r *ImportRegistry) GetSeriesArithmetic(op string, t types.Type, isScalar bool) uint32 {
	opName := operatorToOpName(op)
	typeName := t.Unwrap().String()
	if isScalar {
		return r.MustLookup(ModuleSeries, fmt.Sprintf("element_%s_%s", opName, typeName))
	}
	return r.MustLookup(ModuleSeries, fmt.Sprintf("series_%s_%s", opName, typeName))
}

// GetSeriesReverseArithmetic returns the import index for reverse scalar arithmetic
// operations (scalar op series instead of series op scalar).
func (r *ImportRegistry) GetSeriesReverseArithmetic(op string, t types.Type) uint32 {
	opName := "r" + operatorToOpName(op) // radd, rsub, etc.
	typeName := t.Unwrap().String()
	return r.MustLookup(ModuleSeries, fmt.Sprintf("element_%s_%s", opName, typeName))
}

// GetSeriesComparison returns the import index for series-to-series comparison.
func (r *ImportRegistry) GetSeriesComparison(op string, t types.Type) uint32 {
	opName := comparisonOpName(op)
	typeName := t.Unwrap().String()
	return r.MustLookup(ModuleSeries, fmt.Sprintf("compare_%s_%s", opName, typeName))
}

// GetSeriesScalarComparison returns the import index for series-to-scalar comparison.
func (r *ImportRegistry) GetSeriesScalarComparison(op string, t types.Type) uint32 {
	opName := comparisonOpName(op)
	typeName := t.Unwrap().String()
	return r.MustLookup(ModuleSeries, fmt.Sprintf("compare_%s_scalar_%s", opName, typeName))
}
