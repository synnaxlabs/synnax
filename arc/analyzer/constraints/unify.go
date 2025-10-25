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
	"maps"
	"strings"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func (s *System) Unify() error {
	const maxIterations = 100
	for iteration := 0; iteration < maxIterations; iteration++ {
		var (
			changed      = false
			previousSubs = maps.Clone(s.Substitutions)
		)
		for _, c := range s.Constraints {
			if err := s.unifyTypes(c.Left, c.Right, c); err != nil {
				return errors.Wrapf(err, "failed to unify %v and %v: %s", c.Left, c.Right, c.Reason)
			}
		}
		if len(s.Substitutions) != len(previousSubs) {
			changed = true
		} else {
			for k, newVal := range s.Substitutions {
				if oldVal, exists := previousSubs[k]; !exists || !types.Equal(oldVal, newVal) {
					changed = true
					break
				}
			}
		}
		// If no changes occurred, we've reached a fixpoint
		if !changed {
			break
		}
		// Safety check for infinite loops
		if iteration == maxIterations-1 {
			return errors.Newf("type unification did not converge after %d iterations (possible constraint conflict)", maxIterations)
		}
	}

	// Resolve remaining unresolved type variables with defaults
	for name, tv := range s.TypeVars {
		if _, resolved := s.Substitutions[name]; !resolved {
			if tv.Constraint != nil {
				s.Substitutions[name] = defaultTypeForConstraint(*tv.Constraint)
			} else {
				return errors.Newf("type variable %s could not be resolved", name)
			}
		}
	}
	return nil
}

func (s *System) unifyTypes(t1, t2 types.Type, source Constraint) error {
	return s.unifyTypesWithVisited(t1, t2, source, make(map[string]bool))
}

func (s *System) unifyTypesWithVisited(t1, t2 types.Type, source Constraint, visiting map[string]bool) error {
	// Check for type variables BEFORE applying substitutions
	// This preserves the original type variable for updating
	if t1.Kind == types.KindTypeVariable {
		if visiting[t1.Name] {
			return nil
		}
		visiting[t1.Name] = true
		err := s.unifyTypeVariableWithVisited(t1, t2, source, visiting)
		delete(visiting, t1.Name)
		return err
	}
	if t2.Kind == types.KindTypeVariable {
		if visiting[t2.Name] {
			return nil
		}
		visiting[t2.Name] = true
		err := s.unifyTypeVariableWithVisited(t2, t1, source, visiting)
		delete(visiting, t2.Name)
		return err
	}

	// Now apply substitutions for non-type-variable types
	t1 = s.ApplySubstitutions(t1)
	t2 = s.ApplySubstitutions(t2)
	if types.Equal(t1, t2) {
		return nil
	}

	if t1.Kind == types.KindChan || t1.Kind == types.KindSeries {
		if t2.Kind == types.KindChan || t2.Kind == types.KindSeries {
			return s.unifyTypesWithVisited(t1.Unwrap(), t2.Unwrap(), source, visiting)
		}
		return errors.Newf("cannot unify channel %v with %v", t1, t2)
	}

	if source.Kind == KindCompatible && t1.IsNumeric() && t2.IsNumeric() {
		return nil
	}

	return errors.Newf("types %v and %v are not unifiable", t1, t2)
}

// unifyTypeVariableWithVisited is the internal recursive function with cycle detection
func (s *System) unifyTypeVariableWithVisited(tv types.Type, other types.Type, source Constraint, visiting map[string]bool) error {
	if existing, exists := s.Substitutions[tv.Name]; exists {
		// Type variable already has a substitution
		// If we're in a compatible context with numeric types, we may need to promote
		// BUT: Only promote if both are CONCRETE types. If either is a type variable,
		// just recursively unify without promotion.
		if source.Kind == KindCompatible &&
			existing.Kind != types.KindTypeVariable &&
			other.Kind != types.KindTypeVariable &&
			existing.IsNumeric() && other.IsNumeric() && !types.Equal(existing, other) {
			// Compute the promoted type
			promoted := promoteNumericTypes(existing, other)
			// Always update to promoted type (even if same as existing)
			s.Substitutions[tv.Name] = promoted
			return s.unifyTypesWithVisited(promoted, other, source, visiting)
		}
		return s.unifyTypesWithVisited(existing, other, source, visiting)
	}

	if other.Kind == types.KindTypeVariable {
		if otherSub, exists := s.Substitutions[other.Name]; exists {
			return s.unifyTypeVariableWithVisited(tv, otherSub, source, visiting)
		}
		if tv.Constraint != nil && other.Constraint == nil {
			s.Substitutions[other.Name] = tv
			return nil
		} else if other.Constraint != nil && tv.Constraint == nil {
			s.Substitutions[tv.Name] = other
			return nil
		} else if tv.Name != other.Name {
			s.Substitutions[tv.Name] = other
			return nil
		}
		return nil
	}

	// Unwrap channels to their value type for compatibility checking
	checkType := other.Unwrap()

	if tv.Constraint == nil {
		if occursIn(tv, other) {
			return errors.Newf("cyclic type: %s occurs in %v", tv.Name, other)
		}
	} else if tv.Constraint.Kind == types.KindNumericConstant {
		if !checkType.IsNumeric() {
			return errors.Newf("type %v does not satisfy constraint %v", other, tv)
		}
	} else if tv.Constraint.Kind == types.KindIntegerConstant {
		// Integer constraint: accepts any numeric type (for literal coercion)
		// Integer literals can be coerced to floats: `x f32 := 42` is valid
		if !checkType.IsNumeric() {
			return errors.Newf("type %v does not satisfy integer constraint", other)
		}
	} else if tv.Constraint.Kind == types.KindFloatConstant {
		// Float constraint: accepts float types, or any numeric if compatible context
		if source.Kind == KindCompatible {
			if !checkType.IsNumeric() {
				return errors.Newf("type %v does not satisfy float constraint in compatible context", other)
			}
		} else {
			// In equality/assignment context, only accept floats
			if !checkType.IsFloat() {
				return errors.Newf("type %v does not satisfy float constraint", other)
			}
		}
	}

	// For constraint kinds (IntegerConstant, FloatConstant, NumericConstant),
	// we've already validated compatibility above, so skip exact match check
	isConstraintKind := tv.Constraint != nil && (tv.Constraint.Kind == types.KindIntegerConstant ||
		tv.Constraint.Kind == types.KindFloatConstant ||
		tv.Constraint.Kind == types.KindNumericConstant)

	if !isConstraintKind && tv.Constraint != nil && !types.Equal(*tv.Constraint, other) {
		if source.Kind == KindCompatible && tv.Constraint.IsNumeric() && other.IsNumeric() {
			if tv.Constraint.IsFloat() || other.IsFloat() {
				if tv.Constraint.Is64Bit() || other.Is64Bit() {
					other = types.F64()
				} else {
					other = types.F32()
				}
			} else if tv.Constraint.Is64Bit() || other.Is64Bit() {
				if tv.Constraint.IsSignedInteger() || other.IsSignedInteger() {
					other = types.F64()
				} else {
					other = types.U64()
				}
			} else if tv.Constraint.IsSignedInteger() || other.IsSignedInteger() {
				other = types.I32()
			} else {
				other = types.U32()
			}
		} else {
			return errors.Newf("type %v does not match constraint %v", other, tv.Constraint)
		}
	}
	s.Substitutions[tv.Name] = other
	return nil
}

func occursIn(lhs, rhs types.Type) bool {
	if rhs.Kind == types.KindTypeVariable {
		return lhs.Name == rhs.Name
	}
	if rhs.Kind == types.KindChan || rhs.Kind == types.KindSeries {
		return occursIn(lhs, rhs.Unwrap())
	}
	return false
}

func defaultTypeForConstraint(constraint types.Type) types.Type {
	switch constraint.Kind {
	case types.KindNumericConstant:
		return types.F64()
	case types.KindIntegerConstant:
		return types.I64()
	case types.KindFloatConstant:
		return types.F64()
	default:
		return constraint
	}
}

// promoteNumericTypes returns the promoted type when unifying two numeric types
// Implements the promotion rules from promotion_test.go:
// Rule 1: Float promotion (if either is float)
// Rule 2: 64-bit integer promotion (if either is 64-bit)
// Rule 3: 32-bit promotion (for smaller types)
func promoteNumericTypes(t1, t2 types.Type) types.Type {
	// Rule 1: Float Promotion
	if t1.IsFloat() || t2.IsFloat() {
		if t1.Is64Bit() || t2.Is64Bit() {
			return types.F64()
		}
		return types.F32()
	}

	// Rule 2: 64-bit Integer Promotion
	if t1.Is64Bit() || t2.Is64Bit() {
		// Mixed signedness at 64-bit promotes to F64
		if t1.IsSignedInteger() != t2.IsSignedInteger() {
			return types.F64()
		}
		// Both unsigned 64-bit
		if t1.IsUnsignedInteger() && t2.IsUnsignedInteger() {
			return types.U64()
		}
		// Different widths with signed
		if t1.IsSignedInteger() || t2.IsSignedInteger() {
			return types.F64()
		}
	}

	// Rule 3: 32-bit and Smaller Integer Promotion
	// Mixed signedness at 32-bit or below
	if t1.IsSignedInteger() || t2.IsSignedInteger() {
		return types.I32()
	}
	return types.U32()
}

func (s *System) String() string {
	var b strings.Builder
	b.WriteString("=== Type Unification ===\n")

	b.WriteString(fmt.Sprintf("\nType Variables (%d):\n", len(s.TypeVars)))
	for name, tv := range s.TypeVars {
		b.WriteString(fmt.Sprintf("  %s", name))
		if tv.Constraint != nil {
			b.WriteString(fmt.Sprintf(" : %v", tv.Constraint))
		}
		if sub, exists := s.Substitutions[name]; exists {
			b.WriteString(fmt.Sprintf(" => %v", sub))
		} else {
			b.WriteString(" (unresolved)")
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\nConstraints (%d):\n", len(s.Constraints)))
	for i, c := range s.Constraints {
		kindStr := "≡"
		if c.Kind == KindCompatible {
			kindStr = "~"
		}
		b.WriteString(fmt.Sprintf("  [%d] %v %s %v", i, c.Left, kindStr, c.Right))
		if c.Reason != "" {
			b.WriteString(fmt.Sprintf(" // %s", c.Reason))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\nSubstitutions (%d):\n", len(s.Substitutions)))
	for name, t := range s.Substitutions {
		b.WriteString(fmt.Sprintf("  %s => %v\n", name, t))
	}

	return b.String()
}
