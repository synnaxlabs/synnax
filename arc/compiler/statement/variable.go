// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement

import (
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

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
		return errors.Wrapf(err, "variable '%s' not found in symbol table", name)
	}
	varType := varScope.Type
	exprType, err := expression.Compile(context.Child(ctx, ctx.AST.Expression()).WithHint(varType))
	if err != nil {
		return errors.Wrapf(err, "failed to compile initialization expression for '%s'", name)
	}
	if varType != exprType {
		if err = expression.EmitCast(ctx, exprType, varType); err != nil {
			return err
		}
	}
	local, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return errors.Wrapf(err, "failed to lookup local variable '%s'", name)
	}
	ctx.Writer.WriteLocalSet(local.ID)
	return nil
}

// compileStatefulVariable handles stateful variable declarations (x $= expr)
func compileStatefulVariable(
	ctx context.Context[parser.IStatefulVariableContext],
) error {
	// Resolve the variable name
	name := ctx.AST.IDENTIFIER().GetText()

	// Look up the symbol to get its type
	scope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return errors.Wrapf(err, "stateful variable '%s' not found in symbol table", name)
	}
	varType := scope.Type
	// Compile the initialization expression (analyzer guarantees type correctness)
	_, err = expression.Compile(context.Child(ctx, ctx.AST.Expression()).WithHint(varType))
	if err != nil {
		return errors.Wrapf(err, "failed to compile initialization for stateful variable '%s'", name)
	}
	// Emit state store operation
	// Push func ID (0 for now - runtime will provide actual ID)
	ctx.Writer.WriteI32Const(0)
	// Push state key
	ctx.Writer.WriteI32Const(int32(scope.ID))
	// Value is already on stack from expression compilation
	// Call state store function
	importIdx, err := ctx.Imports.GetStateStore(varType)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(importIdx)
	ctx.Writer.WriteLocalSet(scope.ID)
	return nil
}

// compileAssignment handles variable assignments (x = expr)
func compileAssignment(
	ctx context.Context[parser.IAssignmentContext],
) error {
	// Resolve the variable name
	name := ctx.AST.IDENTIFIER().GetText()
	// Look up the symbol
	scope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return errors.Wrapf(err, "variable '%s' not found", name)
	}
	sym := scope.Symbol
	varType := sym.Type
	// Compile the expression (analyzer guarantees type correctness)
	exprType, err := expression.Compile(context.Child(ctx, ctx.AST.Expression()).WithHint(varType))
	if err != nil {
		return errors.Wrapf(err, "failed to compile assignment expression for '%s'", name)
	}
	if varType != exprType {
		if err = expression.EmitCast(ctx, exprType, varType); err != nil {
			return err
		}
	}
	// Handle based on variable kind
	switch sym.Kind {
	case symbol.KindVariable, symbol.KindInput:
		// Regular local variable or input
		local, err := ctx.Scope.Resolve(ctx, name)
		if err != nil {
			return errors.Newf("local variable '%s' not allocated", name)
		}
		ctx.Writer.WriteLocalSet(local.ID)
	case symbol.KindStatefulVariable:
		stateIdx, err := ctx.Scope.Resolve(ctx, name)
		if err != nil {
			return errors.Newf("stateful variable '%s' not allocated", name)
		}
		ctx.Writer.WriteI32Const(0) // func ID
		ctx.Writer.WriteI32Const(int32(stateIdx.ID))
		importIdx, err := ctx.Imports.GetStateStore(varType)
		if err != nil {
			return err
		}
		ctx.Writer.WriteCall(importIdx)
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

// compileOutputAssignment handles assignment to named outputs in multi-output stages/functions
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
	for i, key := range ctx.Outputs.Keys {
		if key == outputName {
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
		offset += wasm.SizeOf(ctx.Outputs.Values[i])
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
	case types.KindI64, types.KindU64, types.KindTimeStamp, types.KindTimeSpan:
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
