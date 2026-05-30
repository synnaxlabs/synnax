// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolve

import (
	"context"

	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// EmitFixedImportCall looks up the host import's type signature in scope
// and emits a call with that exact signature. Use for monomorphic host
// functions whose signatures the compiler does not want to redeclare
// inline (e.g., string.from_i32 dispatched from EmitNumericToString).
func (r *Resolver) EmitFixedImportCall(
	ctx context.Context,
	w *wasm.Writer,
	writerID int,
	scope *symbol.Symbol,
	module, name string,
) error {
	if scope == nil {
		return errors.Newf("cannot resolve %s.%s: no scope", module, name)
	}
	modSym, err := scope.Resolve(ctx, module, symbol.IncludeInternal)
	if err != nil {
		return errors.Wrapf(err, "resolve module %s", module)
	}
	sym, err := modSym.Resolve(ctx, name, symbol.IncludeInternal)
	if err != nil {
		return errors.Wrapf(err, "resolve %s.%s", module, name)
	}
	if sym.Type.Kind != types.KindFunction {
		return errors.Newf("symbol %s.%s is not a function", module, name)
	}
	r.EmitCall(w, writerID, sym, sym.Type)
	return nil
}

// EmitChannelRead emits a call to channels.read for the given channel type.
// The host function is polymorphic over the element type; the emitted
// import name carries a per-type suffix (channels.read_f64, ...).
func (r *Resolver) EmitChannelRead(w *wasm.Writer, wID int, chanType types.Type) {
	elemType := chanType.UnwrapChan()
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: elemType}},
	})
	r.EmitImportCallWithSuffix(w, wID, "channels", "read", ct, suffixForType(elemType))
}

// EmitChannelWrite emits a call to channels.write for the given element
// type. The host function is polymorphic; suffix derives from elemType.
func (r *Resolver) EmitChannelWrite(w *wasm.Writer, wID int, elemType types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs: types.Params{{Type: types.I32()}, {Type: elemType}},
	})
	r.EmitImportCallWithSuffix(w, wID, "channels", "write", ct, suffixForType(elemType))
}

// EmitStateLoad emits a call to stateful.load for the given type. The host
// function is polymorphic; suffix derives from t.
func (r *Resolver) EmitStateLoad(w *wasm.Writer, wID int, t types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: t}},
		Outputs: types.Params{{Type: t}},
	})
	r.EmitImportCallWithSuffix(w, wID, "stateful", "load", ct, suffixForType(t))
}

// EmitStateStore emits a call to stateful.store for the given type. The
// host function is polymorphic; suffix derives from t.
func (r *Resolver) EmitStateStore(w *wasm.Writer, wID int, t types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs: types.Params{{Type: types.I32()}, {Type: t}},
	})
	r.EmitImportCallWithSuffix(w, wID, "stateful", "store", ct, suffixForType(t))
}

// EmitStateLoadSeries emits a call to stateful.load_series. The elemType
// is used to derive the type suffix (e.g., "f64") even though the WASM
// params are all i32.
func (r *Resolver) EmitStateLoadSeries(w *wasm.Writer, wID int, elemType types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "stateful", "load_series", ct, elemType.String())
}

// EmitStateStoreSeries emits a call to stateful.store_series. The
// elemType is used to derive the type suffix.
func (r *Resolver) EmitStateStoreSeries(w *wasm.Writer, wID int, elemType types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs: types.Params{{Type: types.I32()}, {Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "stateful", "store_series", ct, elemType.String())
}

var opToArithName = map[string]string{
	"+": "add",
	"-": "sub",
	"*": "mul",
	"/": "div",
	"%": "mod",
}

// EmitSeriesArithmetic emits a call to a series arithmetic function.
// If isScalar is true, emits series_element_<op>; otherwise series_series_<op>.
func (r *Resolver) EmitSeriesArithmetic(
	w *wasm.Writer,
	wID int,
	op string,
	elemType types.Type,
	isScalar bool,
) error {
	name, ok := opToArithName[op]
	if !ok {
		return errors.Newf("unknown arithmetic operator: %s", op)
	}
	if isScalar {
		ct := types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Type: types.I32()}, {Type: elemType}},
			Outputs: types.Params{{Type: types.I32()}},
		})
		r.EmitImportCallWithSuffix(w, wID, "series", "element_"+name, ct, suffixForType(elemType))
		return nil
	}
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "series", "series_"+name, ct, suffixForType(elemType))
	return nil
}

// EmitSeriesReverseArithmetic emits a call to a reverse series arithmetic function
// (scalar op series).
func (r *Resolver) EmitSeriesReverseArithmetic(
	w *wasm.Writer,
	wID int,
	op string,
	elemType types.Type,
) error {
	name, ok := opToArithName[op]
	if !ok {
		return errors.Newf("unknown arithmetic operator: %s", op)
	}
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: elemType}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "series", "element_r"+name, ct, suffixForType(elemType))
	return nil
}

var opToCompareName = map[string]string{
	">":  "gt",
	"<":  "lt",
	">=": "ge",
	"<=": "le",
	"==": "eq",
	"!=": "ne",
}

// EmitSeriesComparison emits a call to a series-to-series comparison function.
func (r *Resolver) EmitSeriesComparison(
	w *wasm.Writer,
	wID int,
	op string,
	elemType types.Type,
) error {
	name, ok := opToCompareName[op]
	if !ok {
		return errors.Newf("unknown comparison operator: %s", op)
	}
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "series", "compare_"+name, ct, suffixForType(elemType))
	return nil
}

// EmitSeriesScalarComparison emits a call to a series-to-scalar comparison function.
func (r *Resolver) EmitSeriesScalarComparison(
	w *wasm.Writer,
	wID int,
	op string,
	elemType types.Type,
) error {
	name, ok := opToCompareName[op]
	if !ok {
		return errors.Newf("unknown comparison operator: %s", op)
	}
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: elemType}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "series", "compare_"+name+"_scalar", ct, suffixForType(elemType))
	return nil
}

// EmitSeriesCreateEmpty emits a call to series.create_empty for the given element type.
func (r *Resolver) EmitSeriesCreateEmpty(w *wasm.Writer, wID int, elemType types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "series", "create_empty", ct, suffixForType(elemType))
}

// EmitSeriesSetElement emits a call to series.set_element for the given element type.
func (r *Resolver) EmitSeriesSetElement(w *wasm.Writer, wID int, elemType types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}, {Type: elemType}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "series", "set_element", ct, suffixForType(elemType))
}

// EmitSeriesIndex emits a call to series.index for the given element type.
func (r *Resolver) EmitSeriesIndex(w *wasm.Writer, wID int, elemType types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: elemType}},
	})
	r.EmitImportCallWithSuffix(w, wID, "series", "index", ct, suffixForType(elemType))
}

// EmitSeriesNegate emits a call to series.negate for the given element type.
func (r *Resolver) EmitSeriesNegate(w *wasm.Writer, wID int, elemType types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCallWithSuffix(w, wID, "series", "negate", ct, suffixForType(elemType))
}

// EmitSeriesNotU8 emits a call to series.not_u8.
func (r *Resolver) EmitSeriesNotU8(w *wasm.Writer, wID int) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCall(w, wID, "series", "not_u8", ct)
}

// EmitSeriesLen emits a call to series.len.
func (r *Resolver) EmitSeriesLen(w *wasm.Writer, wID int) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I64()}},
	})
	r.EmitImportCall(w, wID, "series", "len", ct)
}

// EmitSeriesSlice emits a call to series.slice.
func (r *Resolver) EmitSeriesSlice(w *wasm.Writer, wID int) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCall(w, wID, "series", "slice", ct)
}

// EmitStringFromLiteral emits a call to string.from_literal.
func (r *Resolver) EmitStringFromLiteral(w *wasm.Writer, wID int) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCall(w, wID, "strings", "from_literal", ct)
}

// EmitStringConcat emits a call to string.concat.
func (r *Resolver) EmitStringConcat(w *wasm.Writer, wID int) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCall(w, wID, "strings", "concat", ct)
}

// EmitStringEqual emits a call to string.equal.
func (r *Resolver) EmitStringEqual(w *wasm.Writer, wID int) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	r.EmitImportCall(w, wID, "strings", "equal", ct)
}

// EmitStringLen emits a call to string.len.
func (r *Resolver) EmitStringLen(w *wasm.Writer, wID int) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I64()}},
	})
	r.EmitImportCall(w, wID, "strings", "len", ct)
}

// EmitNumericToString emits a call to the string.from_* host fn matching
// the source numeric type. Shared by the str() typecast and f-strings.
// The scope is consulted to look up the host function's signature.
func (r *Resolver) EmitNumericToString(
	ctx context.Context,
	w *wasm.Writer,
	wID int,
	scope *symbol.Symbol,
	from types.Type,
) error {
	suffix, err := numericSuffix(from)
	if err != nil {
		return err
	}
	return r.EmitFixedImportCall(ctx, w, wID, scope, "strings", "from_"+suffix)
}

// EmitNumericFormat emits a call to string.format_<suffix> for the given
// source numeric type. Used by backtick f-strings with format specs.
func (r *Resolver) EmitNumericFormat(
	ctx context.Context,
	w *wasm.Writer,
	wID int,
	scope *symbol.Symbol,
	from types.Type,
) error {
	suffix, err := numericSuffix(from)
	if err != nil {
		return err
	}
	return r.EmitFixedImportCall(ctx, w, wID, scope, "strings", "format_"+suffix)
}

// EmitStringFormat emits a call to string.format_str. Used by backtick
// f-strings to apply a format spec to an already-string value.
func (r *Resolver) EmitStringFormat(
	ctx context.Context,
	w *wasm.Writer,
	wID int,
	scope *symbol.Symbol,
) error {
	return r.EmitFixedImportCall(ctx, w, wID, scope, "strings", "format_str")
}

func numericSuffix(t types.Type) (string, error) {
	switch t.Kind {
	case types.KindI8, types.KindI16, types.KindI32:
		return "i32", nil
	case types.KindU8, types.KindU16, types.KindU32:
		return "u32", nil
	case types.KindI64, types.KindIntegerConstant:
		return "i64", nil
	case types.KindU64:
		return "u64", nil
	case types.KindF32:
		return "f32", nil
	case types.KindF64,
		types.KindFloatConstant, types.KindNumericConstant,
		types.KindExactIntegerFloatConstant:
		return "f64", nil
	}
	return "", errors.Newf("cannot convert %s to str", t)
}

// EmitMathPow emits a call to math.pow for the given type. The host
// function is polymorphic; suffix derives from t.
func (r *Resolver) EmitMathPow(w *wasm.Writer, wID int, t types.Type) {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: t}, {Type: t}},
		Outputs: types.Params{{Type: t}},
	})
	r.EmitImportCallWithSuffix(w, wID, "math", "pow", ct, suffixForType(t))
}

// EmitPanic emits a call to error.panic.
func (r *Resolver) EmitPanic(w *wasm.Writer, wID int) {
	ct := types.Function(types.FunctionProperties{
		Inputs: types.Params{{Type: types.I32()}, {Type: types.I32()}},
	})
	r.EmitImportCall(w, wID, "error", "panic", ct)
}
