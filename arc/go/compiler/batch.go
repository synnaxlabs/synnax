// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
)

// BatchSuffix names the vectorized companion the compiler exports beside every
// element-wise function. The wrapper loops over a whole series inside the guest, so a
// runtime crosses the host-guest boundary once per series instead of once per sample.
const BatchSuffix = "$batch"

// Parameter slots of a BatchSuffix wrapper: the sample count, the output block's base
// pointer, then one (base pointer, byte stride) pair per input. A stride of zero
// repeats a single sample across the whole series.
const (
	BatchCountParam = 0
	BatchOutParam   = 1
	BatchInputParam = 2
)

type batchTarget struct {
	key    string
	inputs types.Params
	output types.Type
}

// batchTargetFor qualifies fn when every input and its single default output is a
// fixed-width numeric. Strings, channels and series cross the boundary as host-owned
// handles, so a guest-side loop cannot index them out of linear memory.
func batchTargetFor(fn ir.Function) (batchTarget, bool) {
	out, ok := fn.Outputs.Get(ir.DefaultOutputParam)
	if !ok || len(fn.Outputs) != 1 || !out.Type.IsFixedWidth() {
		return batchTarget{}, false
	}
	for _, in := range fn.Inputs {
		if !in.Type.IsFixedWidth() {
			return batchTarget{}, false
		}
	}
	return batchTarget{key: fn.Key, inputs: fn.Inputs, output: out.Type}, true
}

// compileBatchWrapper emits the wrapper exported under t.key + BatchSuffix.
func compileBatchWrapper(
	rootCtx context.Context[antlr.ParserRuleContext],
	t batchTarget,
) compiledFunction {
	ctx := rootCtx.WithNewWriter()
	params := make([]wasm.ValueType, 0, BatchInputParam+2*len(t.inputs))
	params = append(params, wasm.I32, wasm.I32)
	for range t.inputs {
		params = append(params, wasm.I32, wasm.I32)
	}
	typeIdx := ctx.Module.AddType(wasm.FunctionType{Params: params})
	var (
		w = ctx.Writer
		i = len(params)
	)
	w.WriteLocalGet(BatchCountParam)
	w.WriteI32Const(0)
	w.WriteBinaryOp(wasm.OpI32GtS)
	w.WriteIf(wasm.BlockTypeEmpty)
	w.WriteLoop(wasm.BlockTypeEmpty)
	w.WriteLocalGet(BatchOutParam)
	w.WriteLocalGet(i)
	w.WriteI32Const(int32(t.output.Density()))
	w.WriteBinaryOp(wasm.OpI32Mul)
	w.WriteBinaryOp(wasm.OpI32Add)
	for k, in := range t.inputs {
		w.WriteLocalGet(BatchInputParam + 2*k)
		w.WriteLocalGet(i)
		w.WriteLocalGet(BatchInputParam + 2*k + 1)
		w.WriteBinaryOp(wasm.OpI32Mul)
		w.WriteBinaryOp(wasm.OpI32Add)
		op, align := batchLoadOp(in.Type)
		w.WriteMemoryOp(op, align, 0)
	}
	ctx.Resolver.EmitLocalCall(w, ctx.WriterID, t.key, types.Type{})
	op, align := batchStoreOp(t.output)
	w.WriteMemoryOp(op, align, 0)
	w.WriteLocalGet(i)
	w.WriteI32Const(1)
	w.WriteBinaryOp(wasm.OpI32Add)
	w.WriteLocalTee(i)
	w.WriteLocalGet(BatchCountParam)
	w.WriteBinaryOp(wasm.OpI32LtS)
	w.WriteBrIf(0)
	w.WriteEnd()
	w.WriteEnd()
	return compiledFunction{
		scopeName: t.key + BatchSuffix,
		typeIdx:   typeIdx,
		locals:    []wasm.ValueType{wasm.I32},
		writer:    w,
	}
}

// batchLoadOp sign-extends narrow signed kinds: the body operates on them as signed
// i32.
func batchLoadOp(t types.Type) (wasm.Opcode, uint32) {
	switch t.Density() {
	case 1:
		if t.Kind == types.KindI8 {
			return wasm.OpI32Load8S, 0
		}
		return wasm.OpI32Load8U, 0
	case 2:
		if t.Kind == types.KindI16 {
			return wasm.OpI32Load16S, 1
		}
		return wasm.OpI32Load16U, 1
	case 4:
		if t.Kind == types.KindF32 {
			return wasm.OpF32Load, 2
		}
		return wasm.OpI32Load, 2
	default:
		if t.Kind == types.KindF64 {
			return wasm.OpF64Load, 3
		}
		return wasm.OpI64Load, 3
	}
}

func batchStoreOp(t types.Type) (wasm.Opcode, uint32) {
	switch t.Density() {
	case 1:
		return wasm.OpI32Store8, 0
	case 2:
		return wasm.OpI32Store16, 1
	case 4:
		if t.Kind == types.KindF32 {
			return wasm.OpF32Store, 2
		}
		return wasm.OpI32Store, 2
	default:
		if t.Kind == types.KindF64 {
			return wasm.OpF64Store, 3
		}
		return wasm.OpI64Store, 3
	}
}
