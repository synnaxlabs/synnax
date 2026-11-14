// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package types provides type inference and checking for Arc language analysis.
package types

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// typeMismatchError represents a type mismatch with an optional hint for fixing it.
type typeMismatchError struct {
	Message string
	Hint    string
	// Cast is the type that the hint says to cast the value to, if any.
	Cast string
}

func (e *typeMismatchError) Error() string { return e.Message }

// GetHint implements the hint interface of diagnostics.
func (e *typeMismatchError) GetHint() string { return e.Hint }

// Check verifies type compatibility between t1 and t2, adding constraints for type
// variables or recursively checking wrapped types for channels and series.
func Check(
	cs *constraints.System,
	t1, t2 types.Type,
	source antlr.ParserRuleContext,
	reason string,
) error {
	if t1.Kind == types.KindInvalid || t2.Kind == types.KindInvalid {
		return nil
	}

	if t1.Kind != types.KindVariable && t2.Kind != types.KindVariable {
		if !types.StructuralMatch(t1, t2) {
			return newTypeMismatchError(t1, t2, reason)
		}
	}

	// Check for wrapped type mismatch (series vs scalar, chan vs scalar) Skip this
	// check if either type is a variable - let the constraint system handle it
	t1Wrapped := t1.Kind == types.KindChan || t1.Kind == types.KindSeries
	t2Wrapped := t2.Kind == types.KindChan || t2.Kind == types.KindSeries
	if t1Wrapped != t2Wrapped && t1.Kind != types.KindVariable &&
		t2.Kind != types.KindVariable {
		resolved1 := cs.ApplySubstitutions(t1)
		resolved2 := cs.ApplySubstitutions(t2)
		if reason != "" {
			return errors.Newf(
				"type mismatch in %s: cannot assign %v to %v",
				reason,
				resolved2,
				resolved1,
			)
		}
		return errors.Newf(
			"type mismatch: cannot assign %v to %v",
			resolved2,
			resolved1,
		)
	}

	if t1.Kind == types.KindVariable && t2.Kind == types.KindVariable &&
		t2.Constraint != nil {
		// The untyped value takes the target's type, and never changes it, so a later
		// write can still fix an untyped target's type.
		target := cs.ApplySubstitutions(t1)
		if !untypedFits(target, t2) {
			return newTypeMismatchError(target, t2, reason)
		}
		return cs.AddEquality(t2, target, source, reason)
	}
	if t1.Kind == types.KindVariable || t2.Kind == types.KindVariable {
		return cs.AddEquality(t1, t2, source, reason)
	}

	if t1.Kind == types.KindSeries || t1.Kind == types.KindChan {
		err := Check(cs, t1.Unwrap(), t2.Unwrap(), source, reason+" (element types)")
		// A cast of the whole value does not convert its elements.
		var mismatch *typeMismatchError
		if errors.As(err, &mismatch) {
			mismatch.Cast = ""
		}
		return err
	}

	if t1.Kind != t2.Kind || !types.UnitsAssignable(t1.Unit, t2.Unit) {
		return newTypeMismatchError(t1, t2, reason)
	}
	return nil
}

// CheckConverted checks actual against expected when the lowering converts between
// their units. Two untyped types stay apart, so each keeps its own unit.
func CheckConverted(
	cs *constraints.System,
	expected, actual types.Type,
	source antlr.ParserRuleContext,
	reason string,
) error {
	expected, actual = cs.ApplySubstitutions(expected), cs.ApplySubstitutions(actual)
	expected.Unit, actual.Unit = nil, nil
	if expected.Kind == types.KindVariable && actual.Kind == types.KindVariable {
		if constrainedTo(expected, types.KindIntegerConstant) !=
			constrainedTo(actual, types.KindIntegerConstant) {
			return newTypeMismatchError(expected, actual, reason)
		}
		return nil
	}
	return Check(cs, expected, actual, source, reason)
}

// newTypeMismatchError creates a typeMismatchError for type mismatches.
// If both types are numeric, it includes a cast suggestion hint.
func newTypeMismatchError(
	expected, actual types.Type,
	reason string,
) *typeMismatchError {
	var msg string
	if reason != "" {
		msg = fmt.Sprintf(
			"type mismatch in %s: expected %v, got %v",
			reason,
			expected,
			actual,
		)
	} else {
		msg = fmt.Sprintf("type mismatch: expected %v, got %v", expected, actual)
	}
	var hint, cast string
	if expected.IsNumeric() && actual.IsNumeric() {
		name := concreteTypeForHint(expected)
		hint = fmt.Sprintf("hint: use %s(value) to convert", name)
		if expected.Unwrap().Kind == expected.Kind &&
			actual.Unwrap().Kind == actual.Kind {
			cast = name
		}
	}
	return &typeMismatchError{Message: msg, Hint: hint, Cast: cast}
}

// CastOf returns the type that a cast of the value fixes err with, if there is one.
func CastOf(err error) (string, bool) {
	var mismatch *typeMismatchError
	if !errors.As(err, &mismatch) || mismatch.Cast == "" {
		return "", false
	}
	return mismatch.Cast, true
}

// concreteTypeForHint returns a concrete type name for use in error hints. Converts
// constraint kinds (integer, float) to their default concrete types (i64, f64).
func concreteTypeForHint(t types.Type) string {
	if t.Kind == types.KindVariable && t.Constraint != nil {
		switch t.Constraint.Kind {
		case types.KindIntegerConstant:
			return "i64"
		case types.KindFloatConstant,
			types.KindNumericConstant,
			types.KindExactIntegerFloatConstant:
			return "f64"
		}
	}
	if t.Kind == types.KindIntegerConstant {
		return "i64"
	}
	if t.Kind == types.KindFloatConstant ||
		t.Kind == types.KindExactIntegerFloatConstant {
		return "f64"
	}
	return t.String()
}
