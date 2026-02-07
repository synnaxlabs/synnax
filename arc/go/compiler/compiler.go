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
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/statement"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

type compiledFunction struct {
	scopeName string
	typeIdx   uint32
	locals    []wasm.ValueType
	writer    *wasm.Writer
}

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

	var symResolver symbol.Resolver
	if !o.disableHostImports {
		symResolver = o.hostSymbols
	}
	resolver := resolve.NewResolver(symResolver)

	compCtx := ccontext.CreateRoot(ctx, program.Symbols, program.TypeMap, resolver)

	for i, f := range program.Functions {
		resolver.RegisterLocal(f.Key, uint32(i))
	}

	outputMemoryCounter := uint32(0x1000)
	outputMemoryBases := make(map[string]uint32)

	var compiled []compiledFunction
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
			var size uint32 = 8
			for _, oParam := range i.Outputs {
				size += uint32(oParam.Type.Density())
			}
			outputMemoryCounter += size
		}

		cf, err := compileItem(compCtx, i.Key, i.Body.AST, params, returnType, i.Outputs, outputMemoryBase)
		if err != nil {
			return Output{}, err
		}
		compiled = append(compiled, cf)
	}

	if err := resolver.FinalizeAndPatch(compCtx.Module); err != nil {
		return Output{}, err
	}

	for _, cf := range compiled {
		funcIdx := compCtx.Module.AddFunction(cf.typeIdx, cf.locals, cf.writer.Bytes())
		compCtx.Module.AddExport(cf.scopeName, wasm.ExportKindFunc, funcIdx)
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
) (compiledFunction, error) {
	scope, err := rootCtx.Scope.Resolve(rootCtx, key)
	if err != nil {
		return compiledFunction{}, err
	}
	wasmParams := make([]wasm.ValueType, 0, len(params))
	for _, param := range params {
		wasmParams = append(wasmParams, wasm.ConvertType(param.Type))
	}
	var wasmResults []wasm.ValueType
	if results.IsValid() {
		wasmResults = append(wasmResults, wasm.ConvertType(results))
	}
	ctx := ccontext.Child(rootCtx, body).WithScope(scope).WithNewWriter()
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
			return compiledFunction{}, errors.Wrapf(err, "failed to compile function '%s' body", ctx.Scope.Name)
		}
	} else if exprCtx, ok := body.(parser.IExpressionContext); ok {
		if err = compileExpression(ccontext.Child(ctx, exprCtx)); err != nil {
			return compiledFunction{}, errors.Wrapf(err, "failed to compile expression '%s'", ctx.Scope.Name)
		}
	} else {
		return compiledFunction{}, errors.Newf("unsupported body type for '%s'", key)
	}

	return compiledFunction{
		scopeName: ctx.Scope.Name,
		typeIdx:   typeIdx,
		locals:    collectLocals(ctx.Scope),
		writer:    ctx.Writer,
	}, nil
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
