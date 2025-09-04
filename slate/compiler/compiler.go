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
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler/core"
	"github.com/synnaxlabs/slate/compiler/statement"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

// Compile generates WASM module and metadata from the analyzed program
func Compile(
	prog parser.IProgramContext,
	analysis *analyzer.Result,
) (*Module, error) {
	var (
		module    = wasm.NewModule()
		importIdx = wasm.SetupTypedImports(module)
		ctx       = core.NewContext(importIdx, analysis.Symbols)
	)
	for _, item := range prog.AllTopLevelItem() {
		if err := compileTopLevelItem(ctx, item); err != nil {
			return nil, err
		}
	}
	mod := &Module{}
	mod.Module = module.Generate()
	return mod, nil
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
	blockScope, _ := funcScope.FirstChildOfKind(symbol.KindBlock)
	if err = statement.CompileBlock(core.NewContext(ctx.Imports, blockScope), funcDecl.Block()); err != nil {
		return errors.Wrapf(err, "failed to compile function '%s' body", name)
	}
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
	blockScope, _ := taskScope.FirstChildOfKind(symbol.KindBlock)
	if err = statement.CompileBlock(core.NewContext(ctx.Imports, blockScope), taskDecl.Block()); err != nil {
		return errors.Wrapf(err, "failed to compile task '%s' body", name)
	}
	return nil
}
