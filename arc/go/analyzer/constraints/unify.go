// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/set"
)

// Sentinel errors for type unification failures.
var (
	// errConstraints is the base constraints error type.
	errConstraints = errors.New("constraints")
	// errCyclicType indicates a type variable occurs within its own substitution.
	errCyclicType = errors.Wrap(errConstraints, "cyclic type dependency")
	// errConstraintViolation indicates a type does not satisfy a type variable's
	// constraint.
	errConstraintViolation = errors.Wrap(errConstraints, "constraint violation")
	// errUnresolvable indicates two types cannot be unified.
	errUnresolvable = errors.Wrap(errConstraints, "types are not unifiable")
	// ErrUnresolvedVariable indicates a type variable could not be resolved to a
	// concrete type.
	ErrUnresolvedVariable = errors.Wrap(errConstraints, "unresolved type variable")
	// errConvergence indicates the unification algorithm did not converge within
	// iteration limit.
	errConvergence = errors.Wrap(errConstraints, "unification did not converge")
)

// UnificationError captures context about a failed type unification.
type UnificationError struct {
	// Constraint that failed to unify.
	Constraint *Constraint
	// Left is the resolved left type after substitutions.
	Left types.Type
	// Right is the resolved right type after substitutions.
	Right types.Type
	// Message is the user-facing error message.
	Message string
	// Hint provides an optional suggestion for fixing the error (e.g., type
	// conversion).
	Hint string
}

func (e *UnificationError) Error() string { return e.Message }

// GetHint implements the hint interface of diagnostics.
func (e *UnificationError) GetHint() string { return e.Hint }

var _ error = (*UnificationError)(nil)

const maxUnificationIterations = 100

// Unify solves all accumulated constraints by computing type variable substitutions.
// Returns an error if constraints conflict or cannot converge within iteration limit.
func (s *System) Unify() error {
	for iteration := range maxUnificationIterations {
		var (
			changed      bool
			previousSubs = maps.Clone(s.Substitutions)
		)
		for _, c := range s.Constraints {
			if err := s.unifyTypes(c.Left, c.Right, c); err != nil {
				return errors.Wrapf(
					err,
					"failed to unify %v and %v: %s",
					c.Left,
					c.Right,
					c.Reason,
				)
			}
		}
		if len(s.Substitutions) != len(previousSubs) {
			changed = true
		} else {
			for k, newVal := range s.Substitutions {
				if oldVal, exists := previousSubs[k]; !exists ||
					!types.Equal(oldVal, newVal) {
					changed = true
					break
				}
			}
		}
		if !changed {
			break
		}
		if iteration == maxUnificationIterations-1 {
			return errors.Wrapf(
				errConvergence,
				"after %d iterations",
				maxUnificationIterations,
			)
		}
	}

	for name, tv := range s.TypeVars {
		if _, resolved := s.Substitutions[name]; !resolved {
			if t, ok := s.Defaults[name]; ok {
				s.Substitutions[name] = t
			} else if tv.Constraint != nil {
				s.Substitutions[name] = defaultTypeForConstraint(*tv.Constraint)
			} else {
				return errors.Wrapf(ErrUnresolvedVariable, "%s", name)
			}
		}
	}
	return nil
}

// UnifyConstraint attempts to unify a single constraint immediately.
// Returns a UnificationError with context if types are incompatible.
func (s *System) UnifyConstraint(c Constraint) error {
	if err := s.unifyTypes(c.Left, c.Right, c); err != nil {
		left := s.ApplySubstitutions(c.Left)
		right := s.ApplySubstitutions(c.Right)
		msg := fmt.Sprintf("type mismatch: %v is not compatible with %v", right, left)
		if c.Reason != "" {
			msg = fmt.Sprintf(
				"type mismatch in %s: %v is not compatible with %v",
				c.Reason,
				right,
				left,
			)
		}
		var hint string
		if left.IsNumeric() && right.IsNumeric() {
			hintType := concreteTypeForHint(left)
			hint = fmt.Sprintf("hint: use %s(value) to convert", hintType)
		}
		return &UnificationError{
			Constraint: &c,
			Left:       left,
			Right:      right,
			Message:    msg,
			Hint:       hint,
		}
	}
	return nil
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

func (s *System) unifyTypes(t1, t2 types.Type, source Constraint) error {
	return s.unifyTypesWithVisited(t1, t2, source, make(set.Set[string]))
}

func (s *System) unifyTypesWithVisited(
	t1, t2 types.Type,
	source Constraint,
	visiting set.Set[string],
) error {
	// Check for type variables BEFORE applying substitutions
	// This preserves the original type variable for updating
	if t1.Kind == types.KindVariable {
		if visiting.Contains(t1.Name) {
			return nil
		}
		visiting.Add(t1.Name)
		defer visiting.Remove(t1.Name)
		return s.unifyTypeVariableWithVisited(t1, t2, source, visiting)
	}
	if t2.Kind == types.KindVariable {
		if visiting.Contains(t2.Name) {
			return nil
		}
		visiting.Add(t2.Name)
		defer visiting.Remove(t2.Name)
		return s.unifyTypeVariableWithVisited(t2, t1, source, visiting)
	}

	// Now apply substitutions for non-type-variable types
	t1 = s.ApplySubstitutions(t1)
	t2 = s.ApplySubstitutions(t2)
	if types.Equal(t1, t2) {
		return nil
	}

	// Handle compound types (channel, series) - must be same kind to unify
	if t1.Kind == types.KindChan || t1.Kind == types.KindSeries {
		if t1.Kind != t2.Kind {
			return errors.Wrapf(errUnresolvable, "%v and %v", t1, t2)
		}
		return s.unifyTypesWithVisited(t1.Unwrap(), t2.Unwrap(), source, visiting)
	}

	if source.Kind == kindCompatible && t1.IsNumeric() && t2.IsNumeric() {
		return nil
	}

	return errors.Wrapf(errUnresolvable, "%v and %v", t1, t2)
}

// numericYields reports whether the constrained variable a takes the type of the
// constrained variable b when one of them is numeric. A float literal fixes a numeric
// variable as a float, and an open series element fixes an integer literal.
func (s *System) numericYields(a, b types.Type) bool {
	if a.Constraint.Kind == types.KindNumericConstant {
		return b.Constraint.Kind != types.KindNumericConstant &&
			b.Constraint.Kind != types.KindIntegerConstant
	}
	return a.Constraint.Kind == types.KindIntegerConstant &&
		s.openNumerics.Contains(b.Name)
}

// integerIntoGeneric returns the integer literal and the generic numeric variable
// among a and b, when they are one of each. A generic numeric variable is one that
// does not stand for an open series element.
func (s *System) integerIntoGeneric(a, b types.Type) (types.Type, types.Type, bool) {
	isKind := func(t types.Type, k types.Kind) bool {
		return t.Constraint != nil && t.Constraint.Kind == k
	}
	isGeneric := func(t types.Type) bool {
		return isKind(t, types.KindNumericConstant) && !s.openNumerics.Contains(t.Name)
	}
	if isKind(a, types.KindIntegerConstant) && isGeneric(b) {
		return a, b, true
	}
	if isKind(b, types.KindIntegerConstant) && isGeneric(a) {
		return b, a, true
	}
	return types.Type{}, types.Type{}, false
}

// openIntegerTarget returns the one of a and b that is the open element of an empty
// series, unbound and with a default, when the other is an integer literal.
func (s *System) openIntegerTarget(a, b types.Type) (types.Type, bool) {
	isOpen := func(t types.Type) bool {
		_, hasDefault := s.Defaults[t.Name]
		return t.Constraint == nil && hasDefault
	}
	isInteger := func(t types.Type) bool {
		return t.Constraint != nil && t.Constraint.Kind == types.KindIntegerConstant
	}
	if isOpen(a) && isInteger(b) {
		return a, true
	}
	if isOpen(b) && isInteger(a) {
		return b, true
	}
	return types.Type{}, false
}

// unifyTypeVariableWithVisited is the internal recursive function with cycle detection
func (s *System) unifyTypeVariableWithVisited(
	tv, other types.Type,
	source Constraint,
	visiting set.Set[string],
) error {
	if existing, exists := s.Substitutions[tv.Name]; exists {
		// Type variable already has a substitution
		// If we're in a compatible context with numeric types, we may need to promote
		// BUT: Only promote if both are CONCRETE types. If either is a type variable,
		// just recursively unify without promotion.
		if source.Kind == kindCompatible &&
			existing.Kind != types.KindVariable &&
			other.Kind != types.KindVariable &&
			existing.IsNumeric() && other.IsNumeric() && !types.Equal(existing, other) {
			// Compute the promoted type
			promoted := promoteNumericTypes(existing, other)
			// Always update to promoted type (even if same as existing)
			s.Substitutions[tv.Name] = promoted
			return s.unifyTypesWithVisited(promoted, other, source, visiting)
		}
		return s.unifyTypesWithVisited(existing, other, source, visiting)
	}

	if other.Kind == types.KindVariable {
		if otherSub, exists := s.Substitutions[other.Name]; exists {
			return s.unifyTypeVariableWithVisited(tv, otherSub, source, visiting)
		}
		// An integer literal makes an empty series numeric, not integer, so on every
		// write path a later float still fits and the element defaults to f64.
		if open, ok := s.openIntegerTarget(tv, other); ok {
			bound := s.OpenNumeric(open)
			s.recordTypeVars(bound)
			s.Substitutions[open.Name] = bound
			return s.unifyTypeVariableWithVisited(tv, other, source, visiting)
		}
		// An integer literal takes a generic numeric variable's type, so a later float
		// still widens it. With no float, the variable defaults to i64.
		if lit, generic, ok := s.integerIntoGeneric(tv, other); ok {
			if _, set := s.Defaults[generic.Name]; !set {
				s.Defaults[generic.Name] = types.I64()
			}
			s.Substitutions[lit.Name] = generic
			return nil
		}
		if tv.Constraint != nil && other.Constraint == nil {
			s.Substitutions[other.Name] = tv
			return nil
		} else if tv.Constraint != nil && other.Constraint != nil &&
			tv.Constraint.Kind == types.KindFloatConstant &&
			(other.Constraint.Kind == types.KindIntegerConstant ||
				other.Constraint.Kind == types.KindExactIntegerFloatConstant) {
			// An integer literal, or a float literal with an integer value such as
			// 2.0, in a float position takes the float's type.
			s.Substitutions[other.Name] = tv
			return nil
		} else if tv.Constraint != nil && other.Constraint != nil &&
			s.numericYields(other, tv) {
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
			return errors.Wrapf(errCyclicType, "%s occurs in %v", tv.Name, other)
		}
	} else if tv.Constraint.Kind == types.KindNumericConstant {
		if !checkType.IsNumeric() {
			return errors.Wrapf(
				errConstraintViolation,
				"%v does not satisfy numeric constraint",
				other,
			)
		}
	} else if tv.Constraint.Kind == types.KindIntegerConstant {
		// Integer constraint: accepts any numeric type (for literal coercion)
		// Integer literals can be coerced to floats: `x f32 := 42` is valid
		if !checkType.IsNumeric() {
			return errors.Wrapf(
				errConstraintViolation,
				"%v does not satisfy integer constraint",
				other,
			)
		}
	} else if tv.Constraint.Kind == types.KindFloatConstant {
		// Float constraint: in compatible context, only accept floats (not concrete
		// integers). This ensures `x := 10; x + 3.2` fails (x is resolved to i64, can't
		// add float literal). But `2 + 3.2` still works because both are type variables
		// that can promote.
		if !checkType.IsFloat() {
			return errors.Wrapf(
				errConstraintViolation,
				"%v does not satisfy float constraint",
				other,
			)
		}
	} else if tv.Constraint.Kind == types.KindExactIntegerFloatConstant {
		if !checkType.IsNumeric() {
			return errors.Wrapf(
				errConstraintViolation,
				"%v does not satisfy exact integer float constraint",
				other,
			)
		}
	}

	// For constraint kinds (IntegerConstant, FloatConstant, NumericConstant,
	// ExactIntegerFloatConstant), we've already validated compatibility above, so skip
	// exact match check
	isConstraintKind := tv.Constraint != nil &&
		(tv.Constraint.Kind == types.KindIntegerConstant ||
			tv.Constraint.Kind == types.KindFloatConstant ||
			tv.Constraint.Kind == types.KindNumericConstant ||
			tv.Constraint.Kind == types.KindExactIntegerFloatConstant)

	if !isConstraintKind && tv.Constraint != nil &&
		!types.Equal(*tv.Constraint, other) {
		if source.Kind != kindCompatible || !tv.Constraint.IsNumeric() ||
			!other.IsNumeric() {
			return errors.Wrapf(
				errConstraintViolation,
				"%v does not satisfy %v constraint",
				other,
				tv.Constraint,
			)
		}
		other = promoteNumericTypes(*tv.Constraint, other)
	}
	s.Substitutions[tv.Name] = other
	return nil
}

func occursIn(lhs, rhs types.Type) bool {
	if rhs.Kind == types.KindVariable {
		return lhs.Name == rhs.Name
	}
	if rhs.Kind == types.KindChan || rhs.Kind == types.KindSeries {
		return occursIn(lhs, rhs.Unwrap())
	}
	return false
}

func defaultTypeForConstraint(constraint types.Type) types.Type {
	switch constraint.Kind {
	case types.KindIntegerConstant:
		return types.I64()
	case types.KindFloatConstant,
		types.KindNumericConstant,
		types.KindExactIntegerFloatConstant:
		return types.F64()
	default:
		return constraint
	}
}

// promoteNumericTypes returns the promoted type when unifying two numeric types.
// Rule 1: Float promotion - if either is float, result is f32 or f64.
// Rule 2: 64-bit integer promotion - mixed signedness promotes to f64.
// Rule 3: 32-bit promotion - smaller types widen to i32 or u32.
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
		if t1.IsSignedInteger() != t2.IsSignedInteger() {
			return types.F64()
		}
		if t1.IsUnsignedInteger() && t2.IsUnsignedInteger() {
			return types.U64()
		}
		return types.F64()
	}

	// Rule 3: 32-bit and Smaller Integer Promotion
	// Mixed signedness at 32-bit or below
	if t1.IsSignedInteger() || t2.IsSignedInteger() {
		return types.I32()
	}
	return types.U32()
}

// String formats the constraint system for debugging with type variables,
// constraints, and substitutions.
func (s *System) String() string {
	var b strings.Builder
	b.WriteString("=== Type Unification ===\n")

	_, _ = fmt.Fprintf(&b, "\nType Variables (%d):\n", len(s.TypeVars))
	for name, tv := range s.TypeVars {
		_, _ = fmt.Fprintf(&b, "  %s", name)
		if tv.Constraint != nil {
			_, _ = fmt.Fprintf(&b, " : %v", tv.Constraint)
		}
		if sub, exists := s.Substitutions[name]; exists {
			_, _ = fmt.Fprintf(&b, " => %v", sub)
		} else {
			b.WriteString(" (unresolved)")
		}
		b.WriteString("\n")
	}

	_, _ = fmt.Fprintf(&b, "\nConstraints (%d):\n", len(s.Constraints))
	for i, c := range s.Constraints {
		kindStr := "≡"
		if c.Kind == kindCompatible {
			kindStr = "~"
		}
		_, _ = fmt.Fprintf(&b, "  [%d] %v %s %v", i, c.Left, kindStr, c.Right)
		if c.Reason != "" {
			_, _ = fmt.Fprintf(&b, " // %s", c.Reason)
		}
		b.WriteString("\n")
	}

	_, _ = fmt.Fprintf(&b, "\nSubstitutions (%d):\n", len(s.Substitutions))
	for name, t := range s.Substitutions {
		_, _ = fmt.Fprintf(&b, "  %s => %v\n", name, t)
	}

	return b.String()
}
