// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compoundOpToString(compoundOp parser.ICompoundOpContext) string {
	switch {
	case compoundOp.PLUS_ASSIGN() != nil:
		return "+"
	case compoundOp.MINUS_ASSIGN() != nil:
		return "-"
	case compoundOp.STAR_ASSIGN() != nil:
		return "*"
	case compoundOp.SLASH_ASSIGN() != nil:
		return "/"
	case compoundOp.PERCENT_ASSIGN() != nil:
		return "%"
	default:
		return ""
	}
}

func compileVariableDeclaration(ctx context.Context[parser.IVariableDeclarationContext]) error {
	if localVar := ctx.AST.LocalVariable(); localVar != nil {
		return compileLocalVariable(context.Child(ctx, localVar))
	}
	if statefulVar := ctx.AST.StatefulVariable(); statefulVar != nil {
		return compileStatefulVariable(context.Child(ctx, statefulVar))
	}
	return errors.New("unknown variable declaration type")
}

func compileLocalVariable(ctx context.Context[parser.ILocalVariableContext]) error {
	name := ctx.AST.IDENTIFIER().GetText()
	varScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return err
	}
	varType := varScope.Type
	exprCtx := context.Child(ctx, ctx.AST.Expression()).WithHint(varType)
	exprType, err := expression.Compile(exprCtx)
	if err != nil {
		return errors.Wrapf(err, "failed to compile initialization expression for '%s'", name)
	}
	if !types.Equal(varType, exprType) {
		if err = expression.EmitCast(ctx, exprType, varType); err != nil {
			return err
		}
	}
	ctx.Writer.WriteLocalSet(varScope.ID)
	return nil
}

// compileStatefulVariable handles stateful variable declarations (x $= expr)
func compileStatefulVariable(
	ctx context.Context[parser.IStatefulVariableContext],
) error {
	name := ctx.AST.IDENTIFIER().GetText()
	scope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return err
	}
	varType := scope.Type

	// Emit state load-or-initialize operation
	// Push state key (variable ID)
	ctx.Writer.WriteI32Const(int32(scope.ID))

	// Compile the initialization expression (analyzer guarantees type correctness)
	exprCtx := context.Child(ctx, ctx.AST.Expression()).WithHint(varType)
	_, err = expression.Compile(exprCtx)
	if err != nil {
		return errors.Wrapf(err, "failed to compile initialization for stateful variable '%s'", name)
	}

	// Stack is now: [varID, initValue/initHandle]
	// Call appropriate state load function based on type
	var importIdx uint32
	if varType.Kind == types.KindSeries {
		// Series types use handle-based state operations
		importIdx, err = ctx.Imports.GetStateLoadSeries(*varType.Elem)
	} else {
		// Primitive types use value-based state operations
		importIdx, err = ctx.Imports.GetStateLoad(varType)
	}
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(importIdx)

	// Stack is now: [value/handle]
	// Store in local variable
	ctx.Writer.WriteLocalSet(scope.ID)
	return nil
}

// compileIndexedAssignment handles indexed assignment statements (series[i] = value)
func compileIndexedAssignment(
	ctx context.Context[parser.IAssignmentContext],
	scope *symbol.Scope,
	indexOrSlice parser.IIndexOrSliceContext,
) error {
	elemType := scope.Type.Unwrap()

	// Step 1: Push series handle onto stack
	ctx.Writer.WriteLocalGet(scope.ID)

	// Step 2: Compile and push index expression
	indexExpressions := indexOrSlice.AllExpression()
	if _, err := expression.Compile(
		context.Child(ctx, indexExpressions[0]).WithHint(types.I32()),
	); err != nil {
		return errors.Wrap(err, "failed to compile index expression")
	}

	// Step 3: Compile and push value expression
	if _, err := expression.Compile(
		context.Child(ctx, ctx.AST.Expression()).WithHint(elemType),
	); err != nil {
		return errors.Wrap(err, "failed to compile value expression")
	}

	// Stack is now: [series_handle, index, value]
	// Step 4: Call series_set_element_<type>(handle, index, value) -> handle
	importIdx, err := ctx.Imports.GetSeriesSetElement(elemType)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(importIdx)

	// Drop the returned handle since we don't need it for assignment
	ctx.Writer.WriteOpcode(wasm.OpDrop)

	return nil
}

// compileIndexedCompoundAssignment handles indexed compound assignment statements (series[i] += value)
// Equivalent to: arr[i] = arr[i] op expr
func compileIndexedCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	scope *symbol.Scope,
	indexOrSlice parser.IIndexOrSliceContext,
	compoundOp parser.ICompoundOpContext,
) error {
	elemType := scope.Type.Unwrap()
	indexExprs := indexOrSlice.AllExpression()
	op := compoundOpToString(compoundOp)

	// Step 1: Push handle and index for eventual set_element call
	ctx.Writer.WriteLocalGet(scope.ID)
	if _, err := expression.Compile(
		context.Child(ctx, indexExprs[0]).WithHint(types.I32()),
	); err != nil {
		return errors.Wrap(err, "failed to compile index expression")
	}
	// Stack: [handle, index]

	// Step 2: Read current value via series_index(handle, index)
	// We compile the index expression again (safe since index expressions are pure)
	ctx.Writer.WriteLocalGet(scope.ID)
	if _, err := expression.Compile(
		context.Child(ctx, indexExprs[0]).WithHint(types.I32()),
	); err != nil {
		return errors.Wrap(err, "failed to compile index expression")
	}
	indexImport, err := ctx.Imports.GetSeriesIndex(elemType)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(indexImport)
	// Stack: [handle, index, current_value]

	// Step 3: Compile RHS expression
	exprType, err := expression.Compile(
		context.Child(ctx, ctx.AST.Expression()).WithHint(elemType),
	)
	if err != nil {
		return errors.Wrap(err, "failed to compile value expression")
	}
	if !types.Equal(elemType, exprType) {
		if err = expression.EmitCast(ctx, exprType, elemType); err != nil {
			return err
		}
	}
	// Stack: [handle, index, current_value, expr_value]

	// Step 4: Apply binary operation
	if elemType.Kind == types.KindString && op == "+" {
		ctx.Writer.WriteCall(ctx.Imports.StringConcat)
	} else {
		if err = ctx.Writer.WriteBinaryOpInferred(op, elemType); err != nil {
			return err
		}
	}
	// Stack: [handle, index, new_value]

	// Step 5: Call series_set_element and drop result
	setImport, err := ctx.Imports.GetSeriesSetElement(elemType)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(setImport)
	ctx.Writer.WriteOpcode(wasm.OpDrop)
	// Stack: []

	return nil
}

// compileSeriesCompoundAssignment handles whole-series compound assignment (series += value)
// Equivalent to: series = series op expr (broadcast or element-wise)
func compileSeriesCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	scope *symbol.Scope,
	compoundOp parser.ICompoundOpContext,
) error {
	sym := scope.Symbol
	varType := sym.Type
	elemType := *varType.Elem
	op := compoundOpToString(compoundOp)

	ctx.Writer.WriteLocalGet(scope.ID)

	exprType, err := expression.Compile(
		context.Child(ctx, ctx.AST.Expression()).WithHint(elemType),
	)
	if err != nil {
		return errors.Wrap(err, "failed to compile value expression")
	}

	isScalar := exprType.Kind != types.KindSeries
	funcIdx, err := ctx.Imports.GetSeriesArithmetic(op, elemType, isScalar)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(funcIdx)

	switch sym.Kind {
	case symbol.KindVariable, symbol.KindInput:
		ctx.Writer.WriteLocalSet(scope.ID)
	case symbol.KindStatefulVariable:
		ctx.Writer.WriteLocalSet(scope.ID)
		ctx.Writer.WriteI32Const(int32(scope.ID))
		ctx.Writer.WriteLocalGet(scope.ID)
		importIdx, err := ctx.Imports.GetStateStoreSeries(elemType)
		if err != nil {
			return err
		}
		ctx.Writer.WriteCall(importIdx)
	default:
		return errors.Newf("compound assignment not supported for %v", sym.Kind)
	}

	return nil
}

func compileCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	scope *symbol.Scope,
	compoundOp parser.ICompoundOpContext,
) error {
	// Handle indexed compound assignment (arr[i] += value)
	if indexOrSlice := ctx.AST.IndexOrSlice(); indexOrSlice != nil {
		return compileIndexedCompoundAssignment(ctx, scope, indexOrSlice, compoundOp)
	}

	sym := scope.Symbol
	varType := sym.Type

	// Handle whole-series compound assignment
	if varType.Kind == types.KindSeries {
		return compileSeriesCompoundAssignment(ctx, scope, compoundOp)
	}

	op := compoundOpToString(compoundOp)
	ctx.Writer.WriteLocalGet(scope.ID)

	exprType, err := expression.Compile(context.Child(ctx, ctx.AST.Expression()).WithHint(varType))
	if err != nil {
		return err
	}

	if !types.Equal(varType, exprType) {
		if err = expression.EmitCast(ctx, exprType, varType); err != nil {
			return err
		}
	}

	if varType.Kind == types.KindString && op == "+" {
		ctx.Writer.WriteCall(ctx.Imports.StringConcat)
	} else {
		if err = ctx.Writer.WriteBinaryOpInferred(op, varType); err != nil {
			return err
		}
	}

	switch sym.Kind {
	case symbol.KindVariable, symbol.KindInput:
		ctx.Writer.WriteLocalSet(scope.ID)
	case symbol.KindStatefulVariable:
		ctx.Writer.WriteLocalSet(scope.ID)
		ctx.Writer.WriteI32Const(int32(scope.ID))
		ctx.Writer.WriteLocalGet(scope.ID)
		resolveImportF := lo.Ternary(
			varType.Kind == types.KindSeries,
			ctx.Imports.GetStateStoreSeries,
			ctx.Imports.GetStateStore,
		)
		importIdx, err := resolveImportF(varType.Unwrap())
		if err != nil {
			return err
		}
		ctx.Writer.WriteCall(importIdx)
	default:
		return errors.Newf("compound assignment not supported for %v", sym.Kind)
	}

	return nil
}

func compileAssignment(
	ctx context.Context[parser.IAssignmentContext],
) error {
	name := ctx.AST.IDENTIFIER().GetText()
	scope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return err
	}

	if compoundOp := ctx.AST.CompoundOp(); compoundOp != nil {
		return compileCompoundAssignment(ctx, scope, compoundOp)
	}

	if indexOrSlice := ctx.AST.IndexOrSlice(); indexOrSlice != nil {
		return compileIndexedAssignment(ctx, scope, indexOrSlice)
	}

	sym := scope.Symbol
	varType := sym.Type

	// For channel writes, push the channel ID before compiling the expression.
	// This avoids needing a temporary local variable to rearrange the stack.
	if sym.Kind == symbol.KindChannel {
		// For direct channel references, scope.ID is the Synnax channel key
		ctx.Writer.WriteI32Const(int32(scope.ID))
	} else if sym.Kind == symbol.KindConfig && varType.Kind == types.KindChan {
		// For config params with channel type, scope.ID is a WASM local index
		// that holds the channel key at runtime - read it from the local
		ctx.Writer.WriteLocalGet(scope.ID)
	}

	targetType := varType.UnwrapChan()
	exprType, err := expression.Compile(context.Child(ctx, ctx.AST.Expression()).WithHint(targetType))
	if err != nil {
		return errors.Wrapf(err, "failed to compile assignment expression for '%s'", name)
	}
	if !types.Equal(targetType, exprType) {
		if err = expression.EmitCast(ctx, exprType, targetType); err != nil {
			return err
		}
	}

	switch sym.Kind {
	case symbol.KindVariable, symbol.KindInput:
		ctx.Writer.WriteLocalSet(scope.ID)
	case symbol.KindStatefulVariable:
		// Stack: [value]
		// Need: [varID, value]
		ctx.Writer.WriteLocalSet(scope.ID)
		ctx.Writer.WriteI32Const(int32(scope.ID))
		ctx.Writer.WriteLocalGet(scope.ID)
		resolveImportF := lo.Ternary(
			varType.Kind == types.KindSeries,
			ctx.Imports.GetStateStoreSeries,
			ctx.Imports.GetStateStore,
		)
		importIdx, err := resolveImportF(varType.Unwrap())
		if err != nil {
			return err
		}
		ctx.Writer.WriteCall(importIdx)
	case symbol.KindChannel:
		// Stack is already [channelID, value] from pushing ID before expression
		importIdx, err := ctx.Imports.GetChannelWrite(varType.Unwrap())
		if err != nil {
			return err
		}
		ctx.Writer.WriteCall(importIdx)
	case symbol.KindConfig:
		// Config params may have channel types - if so, write to the channel
		if varType.Kind == types.KindChan {
			// Stack is already [channelID, value] from pushing ID before expression
			importIdx, err := ctx.Imports.GetChannelWrite(varType.Unwrap())
			if err != nil {
				return err
			}
			ctx.Writer.WriteCall(importIdx)
		} else {
			// Non-channel config param - just set the local
			ctx.Writer.WriteLocalSet(scope.ID)
		}
	case symbol.KindOutput:
		// Named output - needs special handling for multi-output routing
		if err := compileOutputAssignment(ctx, name, scope); err != nil {
			return err
		}
	default:
		return errors.Newf("cannot assign to %v '%s'", sym.Kind, name)
	}

	return nil
}

// compileOutputAssignment handles assignment to named outputs in multi-output functions
// Memory layout at OutputMemoryBase:
//
//	[0:8]   dirty_flags (i64 bitmap)
//	[8:..] output values in declaration order
func compileOutputAssignment(
	ctx context.Context[parser.IAssignmentContext],
	outputName string,
	scope *symbol.Scope,
) error {
	// Value is already on stack from expression compilation

	// Step 1: Store in local variable
	ctx.Writer.WriteLocalSet(scope.ID)

	// Step 2: Find the output index
	outputIndex := -1
	for i, oParam := range ctx.Outputs {
		if oParam.Name == outputName {
			outputIndex = i
			break
		}
	}
	if outputIndex == -1 {
		return errors.Newf("output '%s' not found in outputs list", outputName)
	}

	// Step 3: Calculate memory offset for this output
	offset := ctx.OutputMemoryBase + 8 // Skip dirty flags
	for i := 0; i < outputIndex; i++ {
		offset += uint32(ctx.Outputs[i].Type.Density())
	}

	// Step 4: Write value to output memory
	// Store instruction needs: [address, value]
	ctx.Writer.WriteI32Const(int32(offset))
	ctx.Writer.WriteLocalGet(scope.ID)
	// Stack: [address, value]

	// Write the appropriate store instruction based on type
	switch scope.Type.Kind {
	case types.KindI8, types.KindU8:
		ctx.Writer.WriteMemoryOp(wasm.OpI32Store8, 0, 0)
	case types.KindI16, types.KindU16:
		ctx.Writer.WriteMemoryOp(wasm.OpI32Store16, 1, 0)
	case types.KindI32, types.KindU32:
		ctx.Writer.WriteMemoryOp(wasm.OpI32Store, 2, 0)
	case types.KindI64, types.KindU64:
		ctx.Writer.WriteMemoryOp(wasm.OpI64Store, 3, 0)
	case types.KindF32:
		ctx.Writer.WriteMemoryOp(wasm.OpF32Store, 2, 0)
	case types.KindF64:
		ctx.Writer.WriteMemoryOp(wasm.OpF64Store, 3, 0)
	default:
		return errors.Newf("unsupported output type %v", scope.Type)
	}

	// Step 5: Set the dirty flag bit
	// dirty_flags |= (1 << outputIndex)
	// Store instruction needs: [address, value]
	ctx.Writer.WriteI32Const(int32(ctx.OutputMemoryBase))
	// Load current flags, compute new value, store
	ctx.Writer.WriteI32Const(int32(ctx.OutputMemoryBase))
	ctx.Writer.WriteMemoryOp(wasm.OpI64Load, 3, 0)
	ctx.Writer.WriteI64Const(1)
	ctx.Writer.WriteI64Const(int64(outputIndex))
	ctx.Writer.WriteOpcode(wasm.OpI64Shl)
	ctx.Writer.WriteOpcode(wasm.OpI64Or)
	// Stack: [address, new_value]
	ctx.Writer.WriteMemoryOp(wasm.OpI64Store, 3, 0)
	return nil
}
