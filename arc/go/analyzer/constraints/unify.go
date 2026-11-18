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

	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Sentinel errors for type unification failures.
var (
	// errConstraints is the base constraints error type.
	errConstraints = errors.New("constraints")
	// errCyclicType indicates a type variable occurs within its own substitution.
	errCyclicType = codes.UnresolvedType.Mark(
		errors.Wrap(errConstraints, "cyclic type dependency"),
	)
	// errConstraintViolation indicates a type does not satisfy a type variable's
	// constraint.
	errConstraintViolation = codes.TypeMismatch.Mark(
		errors.Wrap(errConstraints, "constraint violation"),
	)
	// errUnresolvable indicates two types cannot be unified.
	errUnresolvable = codes.TypeMismatch.Mark(
		errors.Wrap(errConstraints, "types are not unifiable"),
	)
	// ErrUnresolvedVariable indicates a type variable could not be resolved to a
	// concrete type.
	ErrUnresolvedVariable = codes.UnresolvedType.Mark(
		errors.Wrap(errConstraints, "unresolved type: type variable"),
	)
	// errConvergence indicates the unification algorithm did not converge within
	// iteration limit.
	errConvergence = codes.UnresolvedType.Mark(
		errors.Wrap(errConstraints, "unification did not converge"),
	)
)

// BoolHint is the hint for a numeric value where a bool is expected. Arc has no cast
// to bool.
const BoolHint = "hint: compare the value to get a bool, such as value != 0"

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
				"type mismatch: %v is not compatible with %v in %s",
				right,
				left,
				c.Reason,
			)
		}
		var hint string
		if left.IsNumeric() && right.IsNumeric() {
			hintType := concreteTypeForHint(left)
			hint = fmt.Sprintf("hint: use %s(value) to convert", hintType)
		} else if c.Kind == kindEquality && left.IsBool() && right.IsNumeric() {
			hint = BoolHint
		}
		return codes.TypeMismatch.Mark(&UnificationError{
			Constraint: &c,
			Left:       left,
			Right:      right,
			Message:    msg,
			Hint:       hint,
		})
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
		case types.KindInvalid, types.KindU8, types.KindU16, types.KindU32,
			types.KindU64, types.KindI8, types.KindI16, types.KindI32, types.KindI64,
			types.KindF32, types.KindF64, types.KindString, types.KindChan,
			types.KindSeries, types.KindVariable, types.KindFunction,
			types.KindSequence, types.KindStage, types.KindVarRef, types.KindBool:
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

// unifyTypeVariableWithVisited is the internal recursive function with cycle detection.
func (s *System) unifyTypeVariableWithVisited(
	tv, other types.Type,
	source Constraint,
	visiting set.Set[string],
) error {
	if existing, exists := s.Substitutions[tv.Name]; exists {
		return s.unifySubstituted(tv, existing, other, source, visiting)
	}
	if other.Kind == types.KindVariable {
		return s.unifyVariables(tv, other, source, visiting)
	}
	if err := checkConstraintBound(tv, other); err != nil {
		return err
	}
	if !isConstraintKind(tv.Constraint) && tv.Constraint != nil &&
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

// unifySubstituted unifies other with the existing substitution of tv. In a compatible
// context, two different concrete numeric types promote, and the promoted type replaces
// the substitution. A type variable on either side never promotes.
func (s *System) unifySubstituted(
	tv, existing, other types.Type,
	source Constraint,
	visiting set.Set[string],
) error {
	if source.Kind == kindCompatible &&
		existing.Kind != types.KindVariable &&
		other.Kind != types.KindVariable &&
		existing.IsNumeric() && other.IsNumeric() && !types.Equal(existing, other) {
		promoted := promoteNumericTypes(existing, other)
		s.Substitutions[tv.Name] = promoted
		return s.unifyTypesWithVisited(promoted, other, source, visiting)
	}
	return s.unifyTypesWithVisited(existing, other, source, visiting)
}

func (s *System) unifyVariables(
	tv, other types.Type,
	source Constraint,
	visiting set.Set[string],
) error {
	if otherSub, exists := s.Substitutions[other.Name]; exists {
		return s.unifyTypeVariableWithVisited(tv, otherSub, source, visiting)
	}
	// An integer literal makes an empty series numeric, not integer, so on every write
	// path a later float still fits and the element defaults to f64.
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
	if s.otherYieldsTo(tv, other) {
		s.Substitutions[other.Name] = tv
		return nil
	}
	if (other.Constraint != nil && tv.Constraint == nil) || tv.Name != other.Name {
		s.Substitutions[tv.Name] = other
	}
	return nil
}

// otherYieldsTo reports whether the unbound variable other takes the type of tv.
func (s *System) otherYieldsTo(tv, other types.Type) bool {
	if tv.Constraint == nil {
		return false
	}
	if other.Constraint == nil {
		return true
	}
	// An integer literal, or a float literal with an integer value such as 2.0, in a
	// float position takes the float's type.
	if tv.Constraint.Kind == types.KindFloatConstant &&
		(other.Constraint.Kind == types.KindIntegerConstant ||
			other.Constraint.Kind == types.KindExactIntegerFloatConstant) {
		return true
	}
	return s.numericYields(other, tv)
}

// checkConstraintBound checks that other, unwrapped from any channel, can bind to tv.
func checkConstraintBound(tv, other types.Type) error {
	if tv.Constraint == nil {
		if occursIn(tv, other) {
			return errors.Wrapf(errCyclicType, "%s occurs in %v", tv.Name, other)
		}
		return nil
	}
	checkType := other.Unwrap()
	kind := tv.Constraint.Kind
	// A float constraint takes only floats, so `x := 10; x + 3.2` fails once x is i64.
	// `2 + 3.2` still works because both sides are type variables.
	if kind == types.KindFloatConstant && !checkType.IsFloat() {
		return constraintViolation(other, "float")
	}
	// Integer literals coerce to floats, so `x f32 := 42` is valid.
	name := numericConstraintName(kind)
	if name != "" && !checkType.IsNumeric() {
		return constraintViolation(other, name)
	}
	return nil
}

func numericConstraintName(kind types.Kind) string {
	if kind == types.KindNumericConstant {
		return "numeric"
	}
	if kind == types.KindIntegerConstant {
		return "integer"
	}
	if kind == types.KindExactIntegerFloatConstant {
		return "exact integer float"
	}
	return ""
}

func constraintViolation(other types.Type, name string) error {
	return errors.Wrapf(
		errConstraintViolation,
		"%v does not satisfy %s constraint",
		other,
		name,
	)
}

// isConstraintKind reports whether constraint is a literal constraint, which
// checkConstraintBound already checked, so it skips the exact match check.
func isConstraintKind(constraint *types.Type) bool {
	return constraint != nil &&
		(constraint.Kind == types.KindIntegerConstant ||
			constraint.Kind == types.KindFloatConstant ||
			constraint.Kind == types.KindNumericConstant ||
			constraint.Kind == types.KindExactIntegerFloatConstant)
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
	case types.KindInvalid, types.KindU8, types.KindU16, types.KindU32, types.KindU64,
		types.KindI8, types.KindI16, types.KindI32, types.KindI64, types.KindF32,
		types.KindF64, types.KindString, types.KindChan, types.KindSeries,
		types.KindVariable, types.KindFunction, types.KindSequence, types.KindStage,
		types.KindVarRef, types.KindBool:
	}
	return constraint
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
