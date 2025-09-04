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
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler/expression"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

// compileVariableDeclaration handles both local (:=) and stateful ($=) variable declarations
func (c *Compiler) compileVariableDeclaration(decl parser.IVariableDeclarationContext) error {
	if localVar := decl.LocalVariable(); localVar != nil {
		return c.compileLocalVariable(localVar)
	}

	if statefulVar := decl.StatefulVariable(); statefulVar != nil {
		return c.compileStatefulVariable(statefulVar)
	}

	return errors.New("unknown variable declaration type")
}

// compileLocalVariable handles local variable declarations (x := expr)
func (c *Compiler) compileLocalVariable(decl parser.ILocalVariableContext) error {
	// Get the variable name
	name := decl.IDENTIFIER().GetText()

	// Look up the symbol to get its type
	scope, err := c.ctx.Symbols.Get(name)
	if err != nil {
		return errors.Wrapf(err, "variable '%s' not found in symbol table", name)
	}
	varType := scope.Symbol.Type

	// Compile the initialization expression (analyzer guarantees type correctness)
	exprType, err := c.expr.Compile(decl.Expression())
	if err != nil {
		return errors.Wrapf(err, "failed to compile initialization expression for '%s'", name)
	}

	// Add type conversion if needed (e.g., i64 literal to i32 variable)
	if needsConversion(exprType, varType) {
		emitTypeConversion(c.enc, exprType, varType)
	}

	// Allocate a local slot if not already allocated
	wasmType := expression.MapType(varType)
	localIdx, exists := c.ctx.GetLocal(name)
	if !exists {
		localIdx = c.ctx.AllocateLocal(name, wasmType)
	}

	// Store the value in the local variable
	c.enc.WriteLocalSet(localIdx)

	return nil
}

// compileStatefulVariable handles stateful variable declarations (x $= expr)
func (c *Compiler) compileStatefulVariable(decl parser.IStatefulVariableContext) error {
	// Get the variable name
	name := decl.IDENTIFIER().GetText()

	// Look up the symbol to get its type
	scope, err := c.ctx.Symbols.Get(name)
	if err != nil {
		return errors.Wrapf(err, "stateful variable '%s' not found in symbol table", name)
	}
	varType := scope.Symbol.Type

	// Compile the initialization expression (analyzer guarantees type correctness)
	_, err = c.expr.Compile(decl.Expression())
	if err != nil {
		return errors.Wrapf(err, "failed to compile initialization for stateful variable '%s'", name)
	}

	// Allocate a stateful storage key if not already allocated
	stateKey, exists := c.ctx.GetStateful(name)
	if !exists {
		stateKey = c.ctx.AllocateStateful(name)
	}

	// Emit state store operation
	// Push task ID (0 for now - runtime will provide actual ID)
	c.enc.WriteI32Const(0)
	// Push state key
	c.enc.WriteI32Const(int32(stateKey))
	// Value is already on stack from expression compilation
	// Call state store function
	importIdx := c.ctx.Imports.GetStateStore(varType)
	c.enc.WriteCall(importIdx)

	// Also store in a local for immediate use in same execution
	wasmType := expression.MapType(varType)
	localIdx, exists := c.ctx.GetLocal(name)
	if !exists {
		// Need to duplicate the value before storing to state
		// This requires loading it again
		c.emitStatefulLoad(stateKey, varType)
		localIdx = c.ctx.AllocateLocal(name, wasmType)
	}
	c.enc.WriteLocalSet(localIdx)

	return nil
}

// compileAssignment handles variable assignments (x = expr)
func (c *Compiler) compileAssignment(assign parser.IAssignmentContext) error {
	// Get the variable name
	name := assign.IDENTIFIER().GetText()

	// Look up the symbol
	scope, err := c.ctx.Symbols.Get(name)
	if err != nil {
		return errors.Wrapf(err, "variable '%s' not found", name)
	}
	sym := scope.Symbol
	varType := sym.Type

	// Compile the expression (analyzer guarantees type correctness)
	exprType, err := c.expr.Compile(assign.Expression())
	if err != nil {
		return errors.Wrapf(err, "failed to compile assignment expression for '%s'", name)
	}
	
	// Add type conversion if needed
	if needsConversion(exprType, varType) {
		emitTypeConversion(c.enc, exprType, varType)
	}

	// Handle based on variable kind
	switch sym.Kind {
	case symbol.KindVariable, symbol.KindParam:
		// Regular local variable or parameter
		localIdx, exists := c.ctx.GetLocal(name)
		if !exists {
			return errors.Newf("local variable '%s' not allocated", name)
		}
		c.enc.WriteLocalSet(localIdx)

	case symbol.KindStatefulVariable:
		// Stateful variable - store to both state and local
		if !c.ctx.Current.IsTask {
			return errors.New("stateful variables can only be used in tasks")
		}

		stateKey, exists := c.ctx.GetStateful(name)
		if !exists {
			return errors.Newf("stateful variable '%s' not allocated", name)
		}

		// Duplicate value for both state store and local store
		c.enc.WriteLocalTee(c.ctx.AllocateLocal("_tmp", expression.MapType(varType)))

		// Store to state
		c.enc.WriteI32Const(0) // Task ID
		c.enc.WriteI32Const(int32(stateKey))
		// Value is on stack
		importIdx := c.ctx.Imports.GetStateStore(varType)
		c.enc.WriteCall(importIdx)

		// Store to local (value was tee'd)
		localIdx, exists := c.ctx.GetLocal(name)
		if !exists {
			return errors.Newf("local shadow for stateful variable '%s' not allocated", name)
		}
		tmpIdx, _ := c.ctx.GetLocal("_tmp")
		c.enc.WriteLocalGet(tmpIdx)
		c.enc.WriteLocalSet(localIdx)

	default:
		return errors.Newf("cannot assign to %v '%s'", sym.Kind, name)
	}

	return nil
}

// Helper functions

func (c *Compiler) emitStatefulLoad(key uint32, t interface{}) {
	c.enc.WriteI32Const(0) // Task ID
	c.enc.WriteI32Const(int32(key))
	importIdx := c.ctx.Imports.GetStateLoad(t)
	c.enc.WriteCall(importIdx)
}

