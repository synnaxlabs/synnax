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

// Check verifies type compatibility between t1 and t2, adding constraints for type variables
// or recursively checking wrapped types for channels and series.
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
			msg := formatTypeMismatch(t1, t2, reason)
			return errors.New(msg)
		}
	}

	// Check for wrapped type mismatch (series vs scalar, chan vs scalar)
	t1Wrapped := t1.Kind == types.KindChan || t1.Kind == types.KindSeries
	t2Wrapped := t2.Kind == types.KindChan || t2.Kind == types.KindSeries
	if t1Wrapped != t2Wrapped {
		resolved1 := cs.ApplySubstitutions(t1)
		resolved2 := cs.ApplySubstitutions(t2)
		if reason != "" {
			return errors.Newf("type mismatch in %s: cannot assign %v to %v", reason, resolved2, resolved1)
		}
		return errors.Newf("type mismatch: cannot assign %v to %v", resolved2, resolved1)
	}

	if t1.Kind == types.KindVariable || t2.Kind == types.KindVariable {
		return cs.AddEquality(t1, t2, source, reason)
	}

	if t1.Kind == types.KindSeries || t1.Kind == types.KindChan {
		return Check(cs, t1.Unwrap(), t2.Unwrap(), source, reason+" (element types)")
	}

	if !types.Equal(t1, t2) {
		msg := formatTypeMismatch(t1, t2, reason)
		return errors.New(msg)
	}
	return nil
}

// formatTypeMismatch creates a descriptive error message for type mismatches.
// If both types are numeric, it includes a cast suggestion hint.
func formatTypeMismatch(expected, actual types.Type, reason string) string {
	var msg string
	if reason != "" {
		msg = fmt.Sprintf("type mismatch in %s: expected %v, got %v", reason, expected, actual)
	} else {
		msg = fmt.Sprintf("type mismatch: expected %v, got %v", expected, actual)
	}
	if expected.IsNumeric() && actual.IsNumeric() {
		hintType := concreteTypeForHint(expected)
		msg += fmt.Sprintf(" (hint: use %s(value) to convert)", hintType)
	}
	return msg
}

// concreteTypeForHint returns a concrete type name for use in error hints.
// Converts constraint kinds (integer, float) to their default concrete types (i64, f64).
func concreteTypeForHint(t types.Type) string {
	if t.Kind == types.KindVariable && t.Constraint != nil {
		switch t.Constraint.Kind {
		case types.KindIntegerConstant:
			return "i64"
		case types.KindFloatConstant:
			return "f64"
		case types.KindNumericConstant:
			return "f64"
		}
	}
	if t.Kind == types.KindIntegerConstant {
		return "i64"
	}
	if t.Kind == types.KindFloatConstant {
		return "f64"
	}
	return t.String()
}
