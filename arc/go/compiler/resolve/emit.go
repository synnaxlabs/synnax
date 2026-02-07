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
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// EmitCall resolves the qualified name with the given concrete type, writes a
// call placeholder to the writer, and records the patch entry. The concreteType
// determines both the WASM function signature and the type suffix.
func (r *Resolver) EmitCall(
	w *wasm.Writer,
	writerID int,
	name string,
	concreteType types.Type,
) error {
	handle, err := r.Resolve(name, concreteType)
	if err != nil {
		return err
	}
	offset := w.WriteCallPlaceholder(handle)
	r.RecordPlaceholder(writerID, handle, offset)
	return nil
}

// emitCallWithSuffix resolves the qualified name with a WASM function type and
// an explicit type suffix. Use this for functions whose WASM params are all
// handles (i32) but whose import name still needs a type suffix.
func (r *Resolver) emitCallWithSuffix(
	w *wasm.Writer,
	writerID int,
	name string,
	wasmType types.Type,
	suffix string,
) error {
	handle, err := r.ResolveWithSuffix(name, wasmType, suffix)
	if err != nil {
		return err
	}
	offset := w.WriteCallPlaceholder(handle)
	r.RecordPlaceholder(writerID, handle, offset)
	return nil
}

// EmitChannelRead emits a call to channel.read for the given channel type.
func (r *Resolver) EmitChannelRead(w *wasm.Writer, wID int, chanType types.Type) error {
	elemType := chanType.UnwrapChan()
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: elemType}},
	})
	return r.EmitCall(w, wID, "channel.read", ct)
}

// EmitChannelWrite emits a call to channel.write for the given element type.
func (r *Resolver) EmitChannelWrite(w *wasm.Writer, wID int, elemType types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs: types.Params{{Type: types.I32()}, {Type: elemType}},
	})
	return r.EmitCall(w, wID, "channel.write", ct)
}

// EmitStateLoad emits a call to state.load for the given type.
func (r *Resolver) EmitStateLoad(w *wasm.Writer, wID int, t types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: t}},
		Outputs: types.Params{{Type: t}},
	})
	return r.EmitCall(w, wID, "state.load", ct)
}

// EmitStateStore emits a call to state.store for the given type.
func (r *Resolver) EmitStateStore(w *wasm.Writer, wID int, t types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs: types.Params{{Type: types.I32()}, {Type: t}},
	})
	return r.EmitCall(w, wID, "state.store", ct)
}

// EmitStateLoadSeries emits a call to state.load_series. The elemType is used
// to derive the type suffix (e.g., "f64") even though the WASM params are all i32.
func (r *Resolver) EmitStateLoadSeries(w *wasm.Writer, wID int, elemType types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.emitCallWithSuffix(w, wID, "state.load_series", ct, elemType.String())
}

// EmitStateStoreSeries emits a call to state.store_series. The elemType is used
// to derive the type suffix.
func (r *Resolver) EmitStateStoreSeries(w *wasm.Writer, wID int, elemType types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs: types.Params{{Type: types.I32()}, {Type: types.I32()}},
	})
	return r.emitCallWithSuffix(w, wID, "state.store_series", ct, elemType.String())
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
		return r.EmitCall(w, wID, "series.element_"+name, ct)
	}
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.emitCallWithSuffix(w, wID, "series.series_"+name, ct, elemType.String())
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
	return r.EmitCall(w, wID, "series.element_r"+name, ct)
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
	return r.emitCallWithSuffix(w, wID, "series.compare_"+name, ct, elemType.String())
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
	return r.EmitCall(w, wID, "series.compare_"+name+"_scalar", ct)
}

// EmitSeriesCreateEmpty emits a call to series.create_empty for the given element type.
func (r *Resolver) EmitSeriesCreateEmpty(w *wasm.Writer, wID int, elemType types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.emitCallWithSuffix(w, wID, "series.create_empty", ct, elemType.String())
}

// EmitSeriesSetElement emits a call to series.set_element for the given element type.
func (r *Resolver) EmitSeriesSetElement(w *wasm.Writer, wID int, elemType types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}, {Type: elemType}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.EmitCall(w, wID, "series.set_element", ct)
}

// EmitSeriesIndex emits a call to series.index for the given element type.
func (r *Resolver) EmitSeriesIndex(w *wasm.Writer, wID int, elemType types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: elemType}},
	})
	return r.EmitCall(w, wID, "series.index", ct)
}

// EmitSeriesNegate emits a call to series.negate for the given element type.
func (r *Resolver) EmitSeriesNegate(w *wasm.Writer, wID int, elemType types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.emitCallWithSuffix(w, wID, "series.negate", ct, elemType.String())
}

// EmitSeriesNotU8 emits a call to series.not_u8.
func (r *Resolver) EmitSeriesNotU8(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.EmitCall(w, wID, "series.not_u8", ct)
}

// EmitSeriesLen emits a call to series.len.
func (r *Resolver) EmitSeriesLen(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I64()}},
	})
	return r.EmitCall(w, wID, "series.len", ct)
}

// EmitSeriesSlice emits a call to series.slice.
func (r *Resolver) EmitSeriesSlice(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.EmitCall(w, wID, "series.slice", ct)
}

// EmitStringFromLiteral emits a call to string.from_literal.
func (r *Resolver) EmitStringFromLiteral(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.EmitCall(w, wID, "string.from_literal", ct)
}

// EmitStringConcat emits a call to string.concat.
func (r *Resolver) EmitStringConcat(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.EmitCall(w, wID, "string.concat", ct)
}

// EmitStringEqual emits a call to string.equal.
func (r *Resolver) EmitStringEqual(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}, {Type: types.I32()}},
		Outputs: types.Params{{Type: types.I32()}},
	})
	return r.EmitCall(w, wID, "string.equal", ct)
}

// EmitStringLen emits a call to string.len.
func (r *Resolver) EmitStringLen(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: types.I32()}},
		Outputs: types.Params{{Type: types.I64()}},
	})
	return r.EmitCall(w, wID, "string.len", ct)
}

// EmitNow emits a call to time.now.
func (r *Resolver) EmitNow(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Outputs: types.Params{{Type: types.I64()}},
	})
	return r.EmitCall(w, wID, "time.now", ct)
}

// EmitMathPow emits a call to math.pow for the given type.
func (r *Resolver) EmitMathPow(w *wasm.Writer, wID int, t types.Type) error {
	ct := types.Function(types.FunctionProperties{
		Inputs:  types.Params{{Type: t}, {Type: t}},
		Outputs: types.Params{{Type: t}},
	})
	return r.EmitCall(w, wID, "math.pow", ct)
}

// EmitPanic emits a call to error.panic.
func (r *Resolver) EmitPanic(w *wasm.Writer, wID int) error {
	ct := types.Function(types.FunctionProperties{
		Inputs: types.Params{{Type: types.I32()}, {Type: types.I32()}},
	})
	return r.EmitCall(w, wID, "error.panic", ct)
}
