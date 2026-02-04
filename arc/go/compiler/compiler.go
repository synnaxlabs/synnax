// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package compiler implements the Arc language compiler, translating Arc intermediate
// representation (IR) into WebAssembly bytecode.
//
// The compiler follows a multi-stage pipeline:
//  1. Expression compilation (recursive descent with operator precedence)
//  2. Statement compilation (control flow, variables, assignments)
//  3. WebAssembly module generation (types, imports, functions, exports)
//  4. Bytecode emission (WASM instruction encoding)
//
// The compiler maintains type information through the compilation process and
// generates calls to host functions for channel operations, series manipulation,
// and state persistence.
//
// Example usage:
//
//	output, err := compiler.Compile(ctx, program)
//	if err != nil {
//	    return err
//	}
//	// output.WASM contains the compiled WebAssembly bytecode
//	// output.OutputMemoryBases contains memory addresses for multi-output functions
package compiler

import (
	"context"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	"github.com/synnaxlabs/arc/compiler/statement"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// Compile translates an Arc program from intermediate representation (IR) to WebAssembly bytecode.
//
// The function processes each function and stage in the IR, allocating memory for multi-output
// stages, compiling expressions and statements into WASM instructions, and generating a complete
// WASM module with type signatures, imports, functions, and exports.
//
// Parameters:
//   - ctx_: Go context for cancellation
//   - program: Arc IR containing functions, stages, symbol table, and type information
//   - opts: Optional compiler configuration (e.g., DisableHostImport())
//
// Returns:
//   - Output: Contains WASM bytecode and memory base addresses for multi-output stages
//   - error: Compilation errors (type mismatches, undefined symbols, invalid operations)
//
// The compiler maintains type safety by propagating type hints through expression compilation
// and emitting type conversions when necessary.
func Compile(ctx context.Context, program ir.IR, opts ...Option) (Output, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	compCtx := ccontext.CreateRoot(ctx, program.Symbols, program.TypeMap, o.disableHostImports)

	importCount := compCtx.Module.ImportCount()
	for i, f := range program.Functions {
		compCtx.FunctionIndices[f.Key] = importCount + uint32(i)
	}

	outputMemoryCounter := uint32(0x1000)
	outputMemoryBases := make(map[string]uint32)

	for _, i := range program.Functions {
		params := slices.Concat(i.Config, i.Inputs)
		var returnType types.Type
		defaultOutput, hasDefaultOutput := i.Outputs.Get(ir.DefaultOutputParam)
		hasNamedOutputs := len(i.Outputs) > 1 || (len(i.Outputs) == 1 && !hasDefaultOutput)
		if !hasNamedOutputs {
			returnType = defaultOutput.Type
		}

		var outputMemoryBase uint32
		if hasNamedOutputs {
			outputMemoryBase = outputMemoryCounter
			outputMemoryBases[i.Key] = outputMemoryBase
			var size uint32 = 8 // dirty flags
			for _, oParam := range i.Outputs {
				size += uint32(oParam.Type.Density())
			}
			outputMemoryCounter += size
		}

		if err := compileItem(compCtx, i.Key, i.Body.AST, params, returnType, i.Outputs, outputMemoryBase); err != nil {
			return Output{}, err
		}
	}

	compCtx.Module.EnableMemory()
	compCtx.Module.AddExport("memory", wasm.ExportKindMemory, 0)

	return Output{WASM: compCtx.Module.Generate(), OutputMemoryBases: outputMemoryBases}, nil
}

func compileItem(
	rootCtx ccontext.Context[antlr.ParserRuleContext],
	key string,
	body antlr.ParserRuleContext,
	params types.Params,
	results types.Type,
	outputs types.Params,
	outputMemoryBase uint32,
) error {
	scope, err := rootCtx.Scope.Resolve(rootCtx, key)
	if err != nil {
		return err
	}
	wasmParams := make([]wasm.ValueType, 0, len(params))
	for _, param := range params {
		wasmParams = append(wasmParams, wasm.ConvertType(param.Type))
	}
	var wasmResults []wasm.ValueType
	if results.IsValid() {
		wasmResults = append(wasmResults, wasm.ConvertType(results))
	}
	wasmFuncIdx := rootCtx.FunctionIndices[key]
	ctx := ccontext.Child(rootCtx, body).WithScope(scope).WithNewWriter().WithFunctionIndex(wasmFuncIdx)
	ctx.Outputs = outputs
	ctx.OutputMemoryBase = outputMemoryBase

	funcT := wasm.FunctionType{Params: wasmParams, Results: wasmResults}
	typeIdx := ctx.Module.AddType(funcT)

	if outputMemoryBase > 0 {
		ctx.Writer.WriteI32Const(int32(outputMemoryBase))
		ctx.Writer.WriteI64Const(0)
		ctx.Writer.WriteMemoryOp(wasm.OpI64Store, 3, 0)
	}

	if blockCtx, ok := body.(parser.IBlockContext); ok {
		_, err = statement.CompileBlock(ccontext.Child(ctx, blockCtx))
		if err != nil {
			return errors.Wrapf(err, "failed to compile function '%s' body", ctx.Scope.Name)
		}
	} else if exprCtx, ok := body.(parser.IExpressionContext); ok {
		if err = compileExpression(ccontext.Child(ctx, exprCtx)); err != nil {
			return errors.Wrapf(err, "failed to compile expression '%s'", ctx.Scope.Name)
		}
	} else {
		return errors.Newf("unsupported body type for '%s'", key)
	}

	funcIdx := ctx.Module.AddFunction(typeIdx, collectLocals(ctx.Scope), ctx.Writer.Bytes())
	ctx.Module.AddExport(ctx.Scope.Name, wasm.ExportKindFunc, funcIdx)
	return nil
}

func compileExpression(ctx ccontext.Context[parser.IExpressionContext]) error {
	_, err := expression.Compile(ctx)
	return err
}

func collectLocals(scope *symbol.Scope) []wasm.ValueType {
	var locals []wasm.ValueType
	for _, child := range scope.Children {
		switch child.Kind {
		case symbol.KindVariable, symbol.KindStatefulVariable, symbol.KindOutput:
			locals = append(locals, wasm.ConvertType(child.Type))
		case symbol.KindBlock:
			locals = append(locals, collectLocals(child)...)
		default:
		}
	}
	return locals
}
