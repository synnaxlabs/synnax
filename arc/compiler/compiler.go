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

// Compile generates a compiled WASM module from the provided IR.
//
// Compilation strategy for multi-output stages:
//   - Stages/functions with single return types compile to WASM functions with return values
//   - Stages/functions with named outputs compile to void WASM functions
//   - Named outputs become local variables in the function body
//   - Multi-output stages write to a reserved memory region:
//     [base_addr + 0]: dirty_flags (i64 bitmap)
//     [base_addr + 8]: output0 value
//     [base_addr + 8 + sizeof(output0)]: output1 value
//     ...
//   - Each multi-output stage/function gets its own memory region
func Compile(ctx_ context.Context, program ir.IR, opts ...Option) (Output, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	ctx := ccontext.CreateRoot(ctx_, program.Symbols, o.disableHostImports)

	// Output memory counter - starts at 0x1000
	outputMemoryCounter := uint32(0x1000)
	hasMultiOutput := false
	outputMemoryBases := make(map[string]uint32)

	for _, i := range program.Stages {
		params := slices.Concat(i.Config.Values, i.Params.Values)
		// Get return type - check for single ir.DefaultOutputParam vs multi-output
		var returnType types.Type
		_, hasDefaultOutput := i.Outputs.Get(ir.DefaultOutputParam)
		hasNamedOutputs := i.Outputs.Count() > 1 || (i.Outputs.Count() == 1 && !hasDefaultOutput)
		if !hasNamedOutputs {
			// Single output case
			returnType, _ = i.Outputs.Get(ir.DefaultOutputParam)
		}

		var outputMemoryBase uint32
		if hasNamedOutputs {
			hasMultiOutput = true
			outputMemoryBase = outputMemoryCounter
			// Track this stage's memory base
			outputMemoryBases[i.Key] = outputMemoryBase
			// Calculate size: 8 bytes for dirty flags + size of all outputs
			size := uint32(8) // dirty flags
			for _, outputType := range i.Outputs.Values {
				size += wasm.SizeOf(outputType)
			}
			outputMemoryCounter += size
		}

		if err := compileItem(ctx, i.Key, i.Body.AST, params, returnType, i.Outputs, outputMemoryBase); err != nil {
			return Output{}, err
		}
	}

	for _, i := range program.Functions {
		// Get return type - check for single ir.DefaultOutputParam vs multi-output
		var returnType types.Type
		_, hasDefaultOutput := i.Outputs.Get(ir.DefaultOutputParam)
		hasNamedOutputs := i.Outputs.Count() > 1 || (i.Outputs.Count() == 1 && !hasDefaultOutput)
		if !hasNamedOutputs {
			// Single output case
			returnType, _ = i.Outputs.Get(ir.DefaultOutputParam)
		}

		var outputMemoryBase uint32
		if hasNamedOutputs {
			hasMultiOutput = true
			outputMemoryBase = outputMemoryCounter
			// Track this function's memory base
			outputMemoryBases[i.Key] = outputMemoryBase
			// Calculate size: 8 bytes for dirty flags + size of all outputs
			size := uint32(8) // dirty flags
			for _, outputType := range i.Outputs.Values {
				size += wasm.SizeOf(outputType)
			}
			outputMemoryCounter += size
		}

		if err := compileItem(ctx, i.Key, i.Body.AST, i.Params.Values, returnType, i.Outputs, outputMemoryBase); err != nil {
			return Output{}, err
		}
	}

	// Enable memory if we have multi-output stages/functions
	if hasMultiOutput {
		ctx.Module.EnableMemory()
	}

	return Output{
		WASM:              ctx.Module.Generate(),
		OutputMemoryBases: outputMemoryBases,
	}, nil
}

func compileItem(
	rootCtx ccontext.Context[antlr.ParserRuleContext],
	key string,
	body antlr.ParserRuleContext,
	params []types.Type,
	results types.Type,
	outputs types.Params,
	outputMemoryBase uint32,
) error {
	scope, err := rootCtx.Scope.Resolve(rootCtx, key)
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
	ctx := ccontext.Child(rootCtx, body).WithScope(scope).WithNewWriter()
	// Set output information for multi-output stages/functions
	ctx.Outputs = outputs
	ctx.OutputMemoryBase = outputMemoryBase

	funcT := wasm.FunctionType{Params: wasmParams, Results: wasmResults}
	typeIdx := ctx.Module.AddType(funcT)

	// Clear dirty flags at start of multi-output functions
	if outputMemoryBase > 0 {
		ctx.Writer.WriteI32Const(int32(outputMemoryBase))
		ctx.Writer.WriteI64Const(0)
		ctx.Writer.WriteMemoryOp(wasm.OpI64Store, 3, 0)
	}

	// Check if body is a block or expression and compile accordingly
	if blockCtx, ok := body.(parser.IBlockContext); ok {
		// Traditional block-based function/stage
		if err = statement.CompileBlock(ccontext.Child(ctx, blockCtx)); err != nil {
			return errors.Wrapf(err, "failed to compile function '%s' body", ctx.Scope.Name)
		}
	} else if exprCtx, ok := body.(parser.IExpressionContext); ok {
		// Flow expression - compile expression and add return
		if err = compileExpression(ccontext.Child(ctx, exprCtx)); err != nil {
			return errors.Wrapf(err, "failed to compile expression '%s'", ctx.Scope.Name)
		}
	} else {
		return errors.Newf("unsupported body type for '%s'", key)
	}

	funcIdx := ctx.Module.AddFunction(typeIdx, collectLocals(ctx.Scope), ctx.Writer.Bytes())
	ctx.Module.AddExport(ctx.Scope.Name, wasm.ExportFunc, funcIdx)
	return nil
}

// compileExpression compiles a flow expression (leaves result on stack)
func compileExpression(ctx ccontext.Context[parser.IExpressionContext]) error {
	// Compile the expression - this will leave the result on the stack
	_, err := expression.Compile(ctx)
	return err
}

func collectLocals(scope *symbol.Scope) []wasm.ValueType {
	var locals []wasm.ValueType
	for _, child := range scope.Children {
		if child.Kind == symbol.KindVariable {
			locals = append(locals, wasm.ConvertType(child.Type))
		} else if child.Kind == symbol.KindOutput {
			// Named outputs are treated as local variables in the function body
			locals = append(locals, wasm.ConvertType(child.Type))
		} else if child.Kind == symbol.KindBlock {
			locals = append(locals, collectLocals(child)...)
		}
	}
	return locals
}
