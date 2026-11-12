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
	"math"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
)

// compileIndexFunctions compiles the function that narrows a 64-bit series index to
// i32, for each index type the program uses. The module holds them after its first
// bodies functions, with no export.
func compileIndexFunctions(
	rootCtx context.Context[antlr.ParserRuleContext],
	bodies int,
) []compiledFunction {
	var compiled []compiledFunction
	for _, t := range []types.Type{types.I64(), types.U64()} {
		name := resolve.IndexFromName(t)
		if !rootCtx.Resolver.References(name) {
			continue
		}
		rootCtx.Resolver.RegisterLocal(name, uint32(bodies+len(compiled)))
		compiled = append(compiled, compileIndexFunction(rootCtx, name, t))
	}
	return compiled
}

// compileIndexFunction emits a function that narrows a 64-bit index of type t to i32.
// It calls the host only for an index that does not fit.
func compileIndexFunction(
	rootCtx context.Context[antlr.ParserRuleContext],
	name string,
	t types.Type,
) compiledFunction {
	const index = 0
	var (
		ctx     = rootCtx.WithNewWriter()
		w       = ctx.Writer
		typeIdx = ctx.Module.AddType(wasm.FunctionType{
			Params:  []wasm.ValueType{wasm.I64},
			Results: []wasm.ValueType{wasm.I32},
		})
	)
	w.WriteLocalGet(index)
	w.WriteI64Const(math.MaxInt32)
	if t.Kind == types.KindU64 {
		w.WriteOpcode(wasm.OpI64GtU)
	} else {
		w.WriteOpcode(wasm.OpI64GtS)
		w.WriteLocalGet(index)
		w.WriteI64Const(math.MinInt32)
		w.WriteOpcode(wasm.OpI64LtS)
		w.WriteOpcode(wasm.OpI32Or)
	}
	w.WriteIf(wasm.BlockTypeEmpty)
	w.WriteLocalGet(index)
	ctx.Resolver.EmitSeriesIndexFrom(w, ctx.WriterID, t)
	w.WriteOpcode(wasm.OpDrop)
	w.WriteEnd()
	w.WriteLocalGet(index)
	w.WriteOpcode(wasm.OpI32WrapI64)
	return compiledFunction{
		scopeName: name,
		typeIdx:   typeIdx,
		writer:    w,
	}
}
