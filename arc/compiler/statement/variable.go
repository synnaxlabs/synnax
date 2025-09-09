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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
)

// compileVariableDeclaration handles both local (:=) and stateful ($=) variable declarations
func compileVariableDeclaration(ctx context.Context[parser.IVariableDeclarationContext]) error {
	if localVar := ctx.AST.LocalVariable(); localVar != nil {
		return compileLocalVariable(context.Child(ctx, localVar))
	}
	if statefulVar := ctx.AST.StatefulVariable(); statefulVar != nil {
		return compileStatefulVariable(context.Child(ctx, statefulVar))
	}
	return errors.New("unknown variable declaration type")
}

// compileLocalVariable handles local variable declarations (x := expr)
func compileLocalVariable(ctx context.Context[parser.ILocalVariableContext]) error {
	name := ctx.AST.IDENTIFIER().GetText()
	varScope, err := ctx.Scope.Resolve(name)
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
	local, err := ctx.Scope.Resolve(name)
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
	scope, err := ctx.Scope.Resolve(name)
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
	// Push stage ID (0 for now - runtime will provide actual ID)
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
	scope, err := ctx.Scope.Resolve(name)
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
	case ir.KindVariable, ir.KindParam:
		// Regular local variable or parameter
		local, err := ctx.Scope.Resolve(name)
		if err != nil {
			return errors.Newf("local variable '%s' not allocated", name)
		}
		ctx.Writer.WriteLocalSet(local.ID)
	case ir.KindStatefulVariable:
		stateIdx, err := ctx.Scope.Resolve(name)
		if err != nil {
			return errors.Newf("stateful variable '%s' not allocated", name)
		}
		ctx.Writer.WriteI32Const(0) // Stage ID
		ctx.Writer.WriteI32Const(int32(stateIdx.ID))
		importIdx, err := ctx.Imports.GetStateStore(varType)
		if err != nil {
			return err
		}
		ctx.Writer.WriteCall(importIdx)
	default:
		return errors.Newf("cannot assign to %v '%s'", sym.Kind, name)
	}

	return nil
}
