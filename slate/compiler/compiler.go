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
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/compiler/core"
	"github.com/synnaxlabs/slate/compiler/statement"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

type Config struct {
	Program            parser.IProgramContext
	Analysis           *analyzer.Result
	DisableHostImports bool
}

func paramsToWasm(params types.OrderedMap[string, types.Type]) []wasm.ValueType {
	wasmParams := make([]wasm.ValueType, 0, params.Count())
	for _, paramType := range params.Iter() {
		wasmParams = append(wasmParams, typeToWASM(paramType))
	}
	return wasmParams
}

// Compile generates WASM module from the analyzed program and graph
func Compile(cfg Config) ([]byte, error) {
	ctx := core.NewContext(cfg.Analysis.Symbols, cfg.DisableHostImports)
	for _, item := range cfg.Program.AllTopLevelItem() {
		if err := compileTopLevelItem(ctx, item); err != nil {
			return nil, err
		}
	}
	return ctx.Module.Generate(), nil
}

// compileTopLevelItem dispatches compilation based on item type
func compileTopLevelItem(ctx *core.Context, item parser.ITopLevelItemContext) error {
	if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
		return compileFunctionDeclaration(ctx, funcDecl)
	}
	if taskDecl := item.TaskDeclaration(); taskDecl != nil {
		return compileTaskDeclaration(ctx, taskDecl)
	}
	if flowStmt := item.FlowStatement(); flowStmt != nil {
		return nil
	}
	return errors.New("unknown top-level item")
}

// typeToWASM converts a Slate type to a WASM value type
func typeToWASM(t types.Type) wasm.ValueType {
	switch t.(type) {
	case types.I32:
		return wasm.I32
	case types.I64, types.TimeStamp, types.TimeSpan:
		return wasm.I64
	case types.F32:
		return wasm.F32
	case types.F64:
		return wasm.F64
	case types.U8, types.U16, types.U32, types.I8, types.I16:
		// All smaller integers are represented as i32 in WASM
		return wasm.I32
	case types.U64:
		return wasm.I64
	default:
		// Default to i32 for unknown types (bool, etc)
		return wasm.I32
	}
}

// compileFunctionDeclaration compiles a function definition
func compileFunctionDeclaration(
	ctx *core.Context,
	funcDecl parser.IFunctionDeclarationContext,
) error {
	name := funcDecl.IDENTIFIER().GetText()
	funcScope, err := ctx.Scope.Resolve(name)
	if err != nil {
		return errors.Wrapf(err, "function '%s' not found in symbols", name)
	}
	funcType, ok := funcScope.Type.(types.Function)
	if !ok {
		return errors.Newf("expected function type for '%s'", name)
	}

	params := make([]wasm.ValueType, 0, funcType.Params.Count())
	for _, paramType := range funcType.Params.Iter() {
		wasmType := typeToWASM(paramType)
		params = append(params, wasmType)
	}
	var results []wasm.ValueType
	if funcType.Return != nil {
		results = append(results, typeToWASM(funcType.Return))
	}
	typeIdx := ctx.Module.AddType(wasm.FunctionType{Params: params, Results: results})
	blockScope, _ := funcScope.FirstChildOfKind(symbol.KindBlock)
	blockCtx := ctx.WithScope(funcScope).WithNewWriter()
	if err = statement.CompileBlock(blockCtx, funcDecl.Block()); err != nil {
		return errors.Wrapf(err, "failed to compile function '%s' body", name)
	}
	funcIdx := ctx.Module.AddFunction(typeIdx, collectLocals(blockScope), blockCtx.Writer.Bytes())
	ctx.Module.AddExport(name, wasm.ExportFunc, funcIdx)
	return nil
}

func collectLocals(scope *symbol.Scope) []wasm.ValueType {
	var locals []wasm.ValueType
	for _, child := range scope.Children {
		if child.Symbol != nil && child.Kind == symbol.KindVariable {
			locals = append(locals, typeToWASM(child.Type))
		} else if child.Kind == symbol.KindBlock {
			locals = append(locals, collectLocals(child)...)
		}
	}
	return locals
}

// compileTaskDeclaration compiles a task definition
func compileTaskDeclaration(
	ctx *core.Context,
	taskDecl parser.ITaskDeclarationContext,
) error {
	name := taskDecl.IDENTIFIER().GetText()
	taskScope, err := ctx.Scope.Resolve(name)
	if err != nil {
		return errors.Wrapf(err, "task '%s' not found in symbols", name)
	}
	taskType, ok := taskScope.Type.(types.Task)
	if !ok {
		return errors.Newf("expected task type for '%s'", name)
	}
	wasmParams := append(paramsToWasm(taskType.Config), paramsToWasm(taskType.Params)...)
	var results []wasm.ValueType
	if taskType.Return != nil {
		results = append(results, typeToWASM(taskType.Return))
	}
	typeIdx := ctx.Module.AddType(wasm.FunctionType{Params: wasmParams, Results: results})
	blockScope, _ := taskScope.FirstChildOfKind(symbol.KindBlock)
	blockCtx := ctx.WithScope(blockScope).WithNewWriter()
	if err = statement.CompileBlock(blockCtx, taskDecl.Block()); err != nil {
		return errors.Wrapf(err, "failed to compile task '%s' body", name)
	}
	funcIdx := ctx.Module.AddFunction(typeIdx, collectLocals(blockScope), blockCtx.Writer.Bytes())
	ctx.Module.AddExport(name, wasm.ExportFunc, funcIdx)
	return nil
}
