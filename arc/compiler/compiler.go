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
	"github.com/synnaxlabs/arc/analyzer/text"
	"github.com/synnaxlabs/arc/compiler/core"
	"github.com/synnaxlabs/arc/compiler/expression"
	"github.com/synnaxlabs/arc/compiler/statement"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

type Config struct {
	Program            text.IProgramContext
	Analysis           *text.Result
	DisableHostImports bool
}

func paramsToWasm(params types.OrderedMap[string, types.Type]) []wasm.ValueType {
	wasmParams := make([]wasm.ValueType, 0, params.Count())
	for _, paramType := range params.Iter() {
		wasmParams = append(wasmParams, wasm.ConvertType(paramType))
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
func compileTopLevelItem(ctx *core.Context, item text.ITopLevelItemContext) error {
	if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
		return compileFunctionDeclaration(ctx, funcDecl)
	}
	if taskDecl := item.TaskDeclaration(); taskDecl != nil {
		return compileTaskDeclaration(ctx, taskDecl)
	}
	if flowStmt := item.FlowStatement(); flowStmt != nil {
		return compileFlowStatement(ctx, flowStmt)
	}
	return errors.New("unknown top-level item")
}

// compileFunctionDeclaration compiles a function definition
func compileFunctionDeclaration(
	ctx *core.Context,
	funcDecl text.IFunctionDeclarationContext,
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
		params = append(params, wasm.ConvertType(paramType))
	}
	var results []wasm.ValueType
	if funcType.Return != nil {
		results = append(results, wasm.ConvertType(funcType.Return))
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
		if child.Kind == symbol.KindVariable {
			locals = append(locals, wasm.ConvertType(child.Type))
		} else if child.Kind == symbol.KindBlock {
			locals = append(locals, collectLocals(child)...)
		}
	}
	return locals
}

// compileTaskDeclaration compiles a task definition
func compileTaskDeclaration(
	ctx *core.Context,
	taskDecl text.ITaskDeclarationContext,
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
		results = append(results, wasm.ConvertType(taskType.Return))
	}
	typeIdx := ctx.Module.AddType(wasm.FunctionType{Params: wasmParams, Results: results})
	blockScope, _ := taskScope.FirstChildOfKind(symbol.KindBlock)
	blockCtx := ctx.WithScope(taskScope).WithNewWriter()
	if err = statement.CompileBlock(blockCtx, taskDecl.Block()); err != nil {
		return errors.Wrapf(err, "failed to compile task '%s' body", name)
	}
	funcIdx := ctx.Module.AddFunction(typeIdx, collectLocals(blockScope), blockCtx.Writer.Bytes())
	ctx.Module.AddExport(name, wasm.ExportFunc, funcIdx)
	return nil
}

func compileFlowStatement(
	ctx *core.Context,
	stmt text.IFlowStatementContext,
) error {
	for _, node := range stmt.AllFlowNode() {
		// Only flow nodes that we need to compile are expressions turned into anonymous
		// tasks.
		if expr := node.Expression(); expr != nil {
			if err := compileFlowExpression(ctx, expr); err != nil {
				return err
			}
		}
	}
	return nil
}

func compileFlowExpression(
	ctx *core.Context,
	expr text.IExpressionContext,
) error {
	scope, err := ctx.Scope.Root().GetChildByParserRule(expr)
	if err != nil {
		return err
	}
	taskType, ok := scope.Type.(types.Task)
	if !ok {
		return errors.Newf("expected task type for flow expression")
	}
	wasmParams := paramsToWasm(taskType.Config)
	var results []wasm.ValueType
	if taskType.Return != nil {
		results = append(results, wasm.ConvertType(taskType.Return))
	}
	typeIdx := ctx.Module.AddType(wasm.FunctionType{Params: wasmParams, Results: results})
	blockScope, _ := scope.FirstChildOfKind(symbol.KindBlock)
	exprCtx := ctx.WithScope(blockScope).WithNewWriter()
	exprType, err := expression.Compile(exprCtx, expr, nil)
	if err != nil {
		return errors.Wrap(err, "failed to compile flow expression")
	}
	if taskType.Return != nil && !types.Equal(exprType, taskType.Return) {
		if err := expression.EmitCast(exprCtx, exprType, taskType.Return); err != nil {
			return nil
		}
	}
	exprCtx.Writer.WriteReturn()
	funcIdx := ctx.Module.AddFunction(typeIdx, nil, exprCtx.Writer.Bytes())
	ctx.Module.AddExport(scope.Name, wasm.ExportFunc, funcIdx)
	return nil
}
