// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/errors"
)

// CheckEqual checks if two types are equal, adding constraints for type variables
func CheckEqual(
	cs *constraints.System,
	t1, t2 ir.Type,
	source antlr.ParserRuleContext,
	reason string,
) error {
	// If either is a type variable, add an equality constraint
	if ir.IsTypeVariable(t1) || ir.IsTypeVariable(t2) {
		cs.AddEquality(t1, t2, source, reason)
		return nil // Defer actual checking to unification pass
	}

	// Handle compound types
	if ch1, ok1 := t1.(ir.Chan); ok1 {
		if ch2, ok2 := t2.(ir.Chan); ok2 {
			return CheckEqual(cs, ch1.ValueType, ch2.ValueType, source, reason+" (channel value types)")
		}
		return errors.Newf("type mismatch: expected %v, got %v", t1, t2)
	}

	if s1, ok1 := t1.(ir.Series); ok1 {
		if s2, ok2 := t2.(ir.Series); ok2 {
			return CheckEqual(cs, s1.ValueType, s2.ValueType, source, reason+" (series element types)")
		}
		return errors.Newf("type mismatch: expected %v, got %v", t1, t2)
	}

	// Both are concrete types, check equality
	if !ir.Equal(t1, t2) {
		return errors.Newf("type mismatch: expected %v, got %v", t1, t2)
	}

	return nil
}

// InferBinaryOpType is the original type inference without type variables
// This is called when both operands are concrete types
func InferBinaryOpType(left, right ir.Type, op string) (ir.Type, error) {
	switch op {
	case "+", "-", "*", "/", "%", "^":
		// Numeric operations
		if !ir.IsNumeric(left) || !ir.IsNumeric(right) {
			return nil, errors.Newf("operator %s requires numeric operands", op)
		}
		// For now, just return the left type - proper promotion should be handled elsewhere
		// if both types are concrete and numeric
		if ir.Equal(left, right) {
			return left, nil
		}
		// Simple promotion: prefer float over int, 64-bit over 32-bit
		if ir.IsFloat(left) || ir.IsFloat(right) {
			if ir.Is64Bit(left) || ir.Is64Bit(right) {
				return ir.F64{}, nil
			}
			return ir.F32{}, nil
		}
		if ir.Is64Bit(left) || ir.Is64Bit(right) {
			if ir.IsSignedInteger(left) || ir.IsSignedInteger(right) {
				return ir.I64{}, nil
			}
			return ir.U64{}, nil
		}
		if ir.IsSignedInteger(left) || ir.IsSignedInteger(right) {
			return ir.I32{}, nil
		}
		return ir.U32{}, nil
	case "<", ">", "<=", ">=":
		// Comparison operations
		if !ir.IsNumeric(left) || !ir.IsNumeric(right) {
			return nil, errors.Newf("comparison operator %s requires numeric operands", op)
		}
		return ir.U8{}, nil
	case "==", "!=":
		// Equality can work on any matching types
		if !ir.Equal(left, right) {
			return nil, errors.Newf("equality operator %s requires compatible types", op)
		}
		return ir.U8{}, nil
	case "&&", "||":
		// Logical operations require booleans (U8)
		if !ir.IsBool(left) || !ir.IsBool(right) {
			return nil, errors.Newf("logical operator %s requires boolean operands", op)
		}
		return ir.U8{}, nil
	default:
		return nil, errors.Newf("unknown operator: %s", op)
	}
}

// HasTypeVariables checks if a type contains any type variables
func HasTypeVariables(t ir.Type) bool {
	if ir.IsTypeVariable(t) {
		return true
	}

	// Check compound types
	if ch, ok := t.(ir.Chan); ok {
		return HasTypeVariables(ch.ValueType)
	}
	if s, ok := t.(ir.Series); ok {
		return HasTypeVariables(s.ValueType)
	}

	return false
}
