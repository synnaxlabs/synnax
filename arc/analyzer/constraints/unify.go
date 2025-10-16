// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraints

import (
	"fmt"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/errors"
)

// Unify solves all constraints in the system and produces type substitutions
func (s *System) Unify() error {
	// Use a simple unification algorithm
	// Process each constraint and try to unify the types
	for _, c := range s.constraints {
		if err := s.unifyTypes(c.Left, c.Right, c); err != nil {
			return errors.Wrapf(err, "failed to unify %v and %v: %s", c.Left, c.Right, c.Reason)
		}
	}

	// Check that all type variables have been resolved
	for name, tv := range s.typeVars {
		if _, resolved := s.substitutions[name]; !resolved {
			// If not resolved, check if it has a constraint that provides a default
			if tv.Constraint != nil {
				// Use the constraint as a default (e.g., NumericConstraint -> F64)
				s.substitutions[name] = defaultTypeForConstraint(tv.Constraint)
			} else {
				return errors.Newf("type variable %s could not be resolved", name)
			}
		}
	}

	return nil
}

// unifyTypes attempts to unify two types, updating substitutions as needed
func (s *System) unifyTypes(t1, t2 ir.Type, source Constraint) error {
	return s.unifyTypesWithVisited(t1, t2, source, make(map[string]bool))
}

// unifyTypesWithVisited is the internal recursive function with cycle detection
func (s *System) unifyTypesWithVisited(t1, t2 ir.Type, source Constraint, visiting map[string]bool) error {
	// Apply existing substitutions first
	t1 = s.ApplySubstitutions(t1)
	t2 = s.ApplySubstitutions(t2)

	// If they're already equal, we're done
	if ir.Equal(t1, t2) {
		return nil
	}

	// Handle type variables with cycle detection
	if tv1, ok := t1.(ir.TypeVariable); ok {
		if visiting[tv1.Name] {
			// We're already visiting this type variable, which means there's a cycle
			// Break the cycle by not recursing further
			return nil
		}
		visiting[tv1.Name] = true
		err := s.unifyTypeVariableWithVisited(tv1, t2, source, visiting)
		delete(visiting, tv1.Name)
		return err
	}
	if tv2, ok := t2.(ir.TypeVariable); ok {
		if visiting[tv2.Name] {
			// We're already visiting this type variable, which means there's a cycle
			// Break the cycle by not recursing further
			return nil
		}
		visiting[tv2.Name] = true
		err := s.unifyTypeVariableWithVisited(tv2, t1, source, visiting)
		delete(visiting, tv2.Name)
		return err
	}

	// Handle compound types
	if ch1, ok1 := t1.(ir.Chan); ok1 {
		if ch2, ok2 := t2.(ir.Chan); ok2 {
			return s.unifyTypesWithVisited(ch1.ValueType, ch2.ValueType, source, visiting)
		}
		return errors.Newf("cannot unify channel %v with %v", t1, t2)
	}

	if s1, ok1 := t1.(ir.Series); ok1 {
		if s2, ok2 := t2.(ir.Series); ok2 {
			return s.unifyTypesWithVisited(s1.ValueType, s2.ValueType, source, visiting)
		}
		return errors.Newf("cannot unify series %v with %v", t1, t2)
	}

	// Handle compatible types (for numeric promotion)
	if source.Kind == KindCompatible && ir.IsNumeric(t1) && ir.IsNumeric(t2) {
		// For compatible constraints, both numeric types can work together
		// The actual promotion happens at codegen time
		return nil
	}

	// Types are not unifiable
	return errors.Newf("types %v and %v are not unifiable", t1, t2)
}

// unifyTypeVariableWithVisited is the internal recursive function with cycle detection
func (s *System) unifyTypeVariableWithVisited(tv ir.TypeVariable, other ir.Type, source Constraint, visiting map[string]bool) error {
	// Check if the type variable already has a substitution
	if existing, exists := s.substitutions[tv.Name]; exists {
		// Try to unify the existing substitution with the new type
		return s.unifyTypesWithVisited(existing, other, source, visiting)
	}

	// Check if other is also a type variable
	if otherTV, ok := other.(ir.TypeVariable); ok {
		// Check if the other variable has a substitution
		if otherSub, exists := s.substitutions[otherTV.Name]; exists {
			return s.unifyTypeVariableWithVisited(tv, otherSub, source, visiting)
		}

		// Neither has a substitution - link them
		// Choose the one with more specific constraint
		if tv.Constraint != nil && otherTV.Constraint == nil {
			s.substitutions[otherTV.Name] = tv
			return nil
		} else if otherTV.Constraint != nil && tv.Constraint == nil {
			s.substitutions[tv.Name] = otherTV
			return nil
		} else if tv.Name != otherTV.Name {
			// Both have constraints or neither do - link one to the other
			s.substitutions[tv.Name] = otherTV
			return nil
		}
		// Same variable, no-op
		return nil
	}

	// Check if the type satisfies the variable's constraint
	if tv.Constraint != nil {
		if nc, ok := tv.Constraint.(ir.NumericConstraint); ok {
			if !ir.IsNumeric(other) {
				return errors.Newf("type %v does not satisfy constraint %v", other, nc)
			}
		} else if !ir.Equal(tv.Constraint, other) {
			// Constraint is a concrete type, must match exactly
			// Unless we're in a compatible constraint and both are numeric
			if source.Kind == KindCompatible && ir.IsNumeric(tv.Constraint) && ir.IsNumeric(other) {
				// Allow different numeric types for compatible constraints
				// Use the more general type
				if ir.IsFloat(tv.Constraint) || ir.IsFloat(other) {
					if ir.Is64Bit(tv.Constraint) || ir.Is64Bit(other) {
						other = ir.F64{}
					} else {
						other = ir.F32{}
					}
				} else if ir.Is64Bit(tv.Constraint) || ir.Is64Bit(other) {
					if ir.IsSignedInteger(tv.Constraint) || ir.IsSignedInteger(other) {
						other = ir.I64{}
					} else {
						other = ir.U64{}
					}
				} else if ir.IsSignedInteger(tv.Constraint) || ir.IsSignedInteger(other) {
					other = ir.I32{}
				} else {
					other = ir.U32{}
				}
			} else {
				return errors.Newf("type %v does not match constraint %v", other, tv.Constraint)
			}
		}
	}

	// Occurs check: make sure tv doesn't occur in other
	if occursIn(tv, other) {
		return errors.Newf("cyclic type: %s occurs in %v", tv.Name, other)
	}

	// Create the substitution
	s.substitutions[tv.Name] = other
	return nil
}

// occursIn checks if a type variable occurs in another type (prevents infinite types)
func occursIn(tv ir.TypeVariable, t ir.Type) bool {
	if t == nil {
		return false
	}

	// Check if it's the same type variable
	if otherTV, ok := t.(ir.TypeVariable); ok {
		return tv.Name == otherTV.Name
	}

	// Check compound types
	if ch, ok := t.(ir.Chan); ok {
		return occursIn(tv, ch.ValueType)
	}
	if series, ok := t.(ir.Series); ok {
		return occursIn(tv, series.ValueType)
	}

	return false
}

// defaultTypeForConstraint returns a default concrete type for a constraint
func defaultTypeForConstraint(constraint ir.Type) ir.Type {
	switch constraint.(type) {
	case ir.NumericConstraint:
		return ir.F64{} // Default to F64 for numeric
	default:
		// If it's a concrete type constraint, use it directly
		return constraint
	}
}

// DebugString returns a detailed debug string showing the unification process
func (s *System) DebugString() string {
	result := "=== Type Unification Debug ===\n"

	// Show type variables
	result += fmt.Sprintf("\nType Variables (%d):\n", len(s.typeVars))
	for name, tv := range s.typeVars {
		result += fmt.Sprintf("  %s", name)
		if tv.Constraint != nil {
			result += fmt.Sprintf(" : %v", tv.Constraint)
		}
		if sub, exists := s.substitutions[name]; exists {
			result += fmt.Sprintf(" => %v", sub)
		} else {
			result += " (unresolved)"
		}
		result += "\n"
	}

	// Show constraints
	result += fmt.Sprintf("\nConstraints (%d):\n", len(s.constraints))
	for i, c := range s.constraints {
		kindStr := "≡"
		if c.Kind == KindCompatible {
			kindStr = "~"
		}
		result += fmt.Sprintf("  [%d] %v %s %v", i, c.Left, kindStr, c.Right)
		if c.Reason != "" {
			result += fmt.Sprintf(" // %s", c.Reason)
		}
		result += "\n"
	}

	// Show final substitutions
	result += fmt.Sprintf("\nSubstitutions (%d):\n", len(s.substitutions))
	for name, t := range s.substitutions {
		result += fmt.Sprintf("  %s => %v\n", name, t)
	}

	return result
}
