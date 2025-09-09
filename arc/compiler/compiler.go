// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import (
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/statement"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
)

// Compile generates a compiled WASM module from the provided IR.
func Compile(ir *ir.IR, opts ...Option) ([]byte, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	ctx := context.CreateRoot(ir.Symbols, o.disableHostImports)
	for _, i := range ir.Stages {
		params := slices.Concat(i.Config.Values, i.Params.Values)
		if err := compileItem(ctx, i.Key, i.Body.AST, params, i.Return); err != nil {
			return nil, err
		}
	}
	for _, i := range ir.Functions {
		if err := compileItem(ctx, i.Key, i.Body.AST, i.Params.Values, i.Return); err != nil {
			return nil, err
		}
	}
	return ctx.Module.Generate(), nil
}

func compileItem(
	rootCtx context.Context[antlr.ParserRuleContext],
	key string,
	body parser.IBlockContext,
	params []ir.Type,
	results ir.Type,
) error {
	scope, err := rootCtx.Scope.Resolve(key)
	if err != nil {
		return err
	}
	wasmParams := make([]wasm.ValueType, 0, len(params))
	for _, paramType := range params {
		wasmParams = append(wasmParams, wasm.ConvertType(paramType))
	}
	var wasmResults []wasm.ValueType
	if results != nil {
		wasmResults = append(wasmResults, wasm.ConvertType(results))
	}
	ctx := context.Child(rootCtx, body).WithScope(scope).WithNewWriter()
	funcT := wasm.FunctionType{Params: wasmParams, Results: wasmResults}
	typeIdx := ctx.Module.AddType(funcT)
	if err = statement.CompileBlock(ctx); err != nil {
		return errors.Wrapf(err, "failed to compile function '%s' body", ctx.Scope.Name)
	}
	funcIdx := ctx.Module.AddFunction(typeIdx, collectLocals(ctx.Scope), ctx.Writer.Bytes())
	ctx.Module.AddExport(ctx.Scope.Name, wasm.ExportFunc, funcIdx)
	return nil
}

func collectLocals(scope *ir.Scope) []wasm.ValueType {
	var locals []wasm.ValueType
	for _, child := range scope.Children {
		if child.Kind == ir.KindVariable {
			locals = append(locals, wasm.ConvertType(child.Type))
		} else if child.Kind == ir.KindBlock {
			locals = append(locals, collectLocals(child)...)
		}
	}
	return locals
}
