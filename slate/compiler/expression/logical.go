// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
)

// compileLogicalOrImpl handles || operations with short-circuit evaluation
func (e *Compiler) compileLogicalOrImpl(expr parser.ILogicalOrExpressionContext) (types.Type, error) {
	ands := expr.AllLogicalAndExpression()
	
	// Compile first operand
	_, err := e.compileLogicalAnd(ands[0])
	if err != nil {
		return nil, err
	}
	
	// Normalize the first operand
	e.normalizeBoolean()
	
	// Process remaining operands with short-circuit evaluation
	for i := 1; i < len(ands); i++ {
		// The stack has the current boolean value (0 or 1)
		// If it's true (1), we skip evaluation of the right operand
		
		// Use if-else block for short-circuit evaluation
		e.encoder.WriteIf(wasm.BlockTypeI32)
		
		// True case: value is already 1, keep it
		e.encoder.WriteI32Const(1)
		
		e.encoder.WriteOpcode(wasm.OpElse)
		
		// False case: evaluate right operand
		_, err := e.compileLogicalAnd(ands[i])
		if err != nil {
			return nil, err
		}
		
		// Normalize the result
		e.normalizeBoolean()
		
		e.encoder.WriteOpcode(wasm.OpEnd)
	}
	
	return types.U8{}, nil
}

// compileLogicalAndImpl handles && operations with short-circuit evaluation
func (e *Compiler) compileLogicalAndImpl(expr parser.ILogicalAndExpressionContext) (types.Type, error) {
	eqs := expr.AllEqualityExpression()
	
	// Compile first operand
	_, err := e.compileEquality(eqs[0])
	if err != nil {
		return nil, err
	}
	
	// Normalize the first operand
	e.normalizeBoolean()
	
	// Process remaining operands with short-circuit evaluation
	for i := 1; i < len(eqs); i++ {
		// The stack has the current boolean value (0 or 1)
		// If it's false (0), we skip evaluation of the right operand
		
		// Use if-else block for short-circuit evaluation
		e.encoder.WriteOpcode(wasm.OpI32Eqz) // Invert: 0 -> 1, 1 -> 0
		e.encoder.WriteIf(wasm.BlockTypeI32)
		
		// True case (was zero): result is 0
		e.encoder.WriteI32Const(0)
		
		e.encoder.WriteOpcode(wasm.OpElse)
		
		// False case (was non-zero): evaluate right operand
		_, err := e.compileEquality(eqs[i])
		if err != nil {
			return nil, err
		}
		
		// Normalize the result
		e.normalizeBoolean()
		
		e.encoder.WriteOpcode(wasm.OpEnd)
	}
	
	return types.U8{}, nil
}

// normalizeBoolean converts any non-zero i32 value to 1
func (e *Compiler) normalizeBoolean() {
	// Convert any non-zero value to 1
	// value != 0 ? 1 : 0
	// This is equivalent to: (value != 0)
	e.encoder.WriteI32Const(0)
	e.encoder.WriteOpcode(wasm.OpI32Ne)
}