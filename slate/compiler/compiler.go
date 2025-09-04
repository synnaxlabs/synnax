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
	"github.com/synnaxlabs/slate/compiler/runtime"
	"github.com/synnaxlabs/slate/compiler/statement"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// Compile generates WASM module from the analyzed program and graph
func Compile(
	prog parser.IProgramContext,
	analysis *analyzer.Result,
) ([]byte, error) {
	mod := wasm.NewModule()
	// Pass mod to context so functions can be added to it
	ctx := &core.Context{
		Module:  mod,
		Scope:   analysis.Symbols,
		Writer:  wasm.NewWriter(),
		Imports: runtime.SetupImports(mod),
	}
	for _, item := range prog.AllTopLevelItem() {
		if err := compileTopLevelItem(ctx, item); err != nil {
			return nil, err
		}
	}
	return mod.Generate(), nil
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
	funcScope, err := ctx.Scope.Get(name)
	if err != nil {
		return errors.Wrapf(err, "function '%s' not found in symbols", name)
	}
	
	// Get function type to determine parameters and return type
	funcType, ok := funcScope.Symbol.Type.(types.Function)
	if !ok {
		return errors.Newf("expected function type for '%s'", name)
	}
	
	// Convert parameter and return types to WASM types
	var params []wasm.ValueType
	for _, paramType := range funcType.Params.Iter() {
		wasmType := typeToWASM(paramType)
		params = append(params, wasmType)
	}
	
	var results []wasm.ValueType
	if funcType.Return != nil {
		results = append(results, typeToWASM(funcType.Return))
	}
	
	// Create or get function type
	typeIdx := ctx.Module.AddType(wasm.FunctionType{
		Params:  params,
		Results: results,
	})
	
	// Compile function body
	blockScope, _ := funcScope.FirstChildOfKind(symbol.KindBlock)
	blockCtx := &core.Context{
		Imports: ctx.Imports,
		Module:  ctx.Module,
		Writer:  wasm.NewWriter(),
		Scope:   blockScope,
	}
	if err = statement.CompileBlock(blockCtx, funcDecl.Block()); err != nil {
		return errors.Wrapf(err, "failed to compile function '%s' body", name)
	}
	
	// Collect locals (variables declared in the function, not parameters)
	var locals []wasm.ValueType
	for _, child := range blockScope.Children {
		if child.Symbol != nil && child.Symbol.Kind == symbol.KindVariable {
			locals = append(locals, typeToWASM(child.Symbol.Type))
		}
	}
	
	// Add function to module
	funcIdx := ctx.Module.AddFunction(typeIdx, locals, blockCtx.Writer.Bytes())
	
	// Export the function
	ctx.Module.AddExport(name, wasm.ExportFunc, funcIdx)
	
	return nil
}

// compileTaskDeclaration compiles a task definition
func compileTaskDeclaration(
	ctx *core.Context,
	taskDecl parser.ITaskDeclarationContext,
) error {
	name := taskDecl.IDENTIFIER().GetText()
	taskScope, err := ctx.Scope.Get(name)
	if err != nil {
		return errors.Wrapf(err, "task '%s' not found in symbols", name)
	}
	
	// Get task type to determine parameters
	taskType, ok := taskScope.Symbol.Type.(types.Task)
	if !ok {
		return errors.Newf("expected task type for '%s'", name)
	}
	
	// Convert parameter types to WASM types (tasks don't have return values)
	var params []wasm.ValueType
	for _, paramType := range taskType.Params.Iter() {
		wasmType := typeToWASM(paramType)
		params = append(params, wasmType)
	}
	
	// Create or get function type (tasks are functions with no return)
	typeIdx := ctx.Module.AddType(wasm.FunctionType{
		Params:  params,
		Results: []wasm.ValueType{}, // Tasks don't return values
	})
	
	// Compile task body
	blockScope, _ := taskScope.FirstChildOfKind(symbol.KindBlock)
	blockCtx := &core.Context{
		Imports: ctx.Imports,
		Module:  ctx.Module,
		Writer:  wasm.NewWriter(),
		Scope:   blockScope,
	}
	if err = statement.CompileBlock(blockCtx, taskDecl.Block()); err != nil {
		return errors.Wrapf(err, "failed to compile task '%s' body", name)
	}
	
	// Collect locals (variables declared in the task, not parameters)
	var locals []wasm.ValueType
	for _, child := range blockScope.Children {
		if child.Symbol != nil && child.Symbol.Kind == symbol.KindVariable {
			locals = append(locals, typeToWASM(child.Symbol.Type))
		}
	}
	
	// Add function to module (tasks are compiled as WASM functions)
	funcIdx := ctx.Module.AddFunction(typeIdx, locals, blockCtx.Writer.Bytes())
	
	// Export the task
	ctx.Module.AddExport(name, wasm.ExportFunc, funcIdx)
	
	return nil
}
