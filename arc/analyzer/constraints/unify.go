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

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// Unify solves all constraints in the system and produces type substitutions
func (s *System) Unify() error {
	for _, c := range s.constraints {
		if err := s.unifyTypes(c.Left, c.Right, c); err != nil {
			return errors.Wrapf(err, "failed to unify %v and %v: %s", c.Left, c.Right, c.Reason)
		}
	}
	for name, tv := range s.typeVars {
		if _, resolved := s.substitutions[name]; !resolved {
			if tv.Constraint != nil {
				s.substitutions[name] = defaultTypeForConstraint(*tv.Constraint)
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
	t1 = s.ApplySubstitutions(t1)
	t2 = s.ApplySubstitutions(t2)
	if types.Equal(t1, t2) {
		return nil
	}
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

	if t1.Kind == types.KindChan || t1.Kind == types.KindSeries {
		if t2.Kind == types.KindChan || t2.Kind == types.KindSeries {
			return s.unifyTypesWithVisited(*t1.ValueType, *t2.ValueType, source, visiting)
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
	if existing, exists := s.substitutions[tv.Name]; exists {
		return s.unifyTypesWithVisited(existing, other, source, visiting)
	}

	if other.Kind == types.KindTypeVariable {
		if otherSub, exists := s.substitutions[other.Name]; exists {
			return s.unifyTypeVariableWithVisited(tv, otherSub, source, visiting)
		}
		if tv.Constraint != nil && other.Constraint == nil {
			s.substitutions[other.Name] = tv
			return nil
		} else if other.Constraint != nil && tv.Constraint == nil {
			s.substitutions[tv.Name] = other
			return nil
		} else if tv.Name != other.Name {
			s.substitutions[tv.Name] = other
			return nil
		}
		return nil
	}

	// Check if the type satisfies the variable's constraint
	if tv.Constraint != nil {
		if tv.Constraint.Kind == types.KindNumericConstant {
			if !other.IsNumeric() {
				return errors.Newf("type %v does not satisfy constraint %v", other, tv)
			}
		} else if !types.Equal(*tv.Constraint, other) {
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
	}

	if occursIn(tv, other) {
		return errors.Newf("cyclic type: %s occurs in %v", tv.Name, other)
	}
	s.substitutions[tv.Name] = other
	return nil
}

// occursIn checks if a type variable occurs in another type (prevents infinite types)
func occursIn(lhs, rhs types.Type) bool {
	if rhs.Kind == types.KindTypeVariable {
		return lhs.Name == rhs.Name
	}
	if rhs.Kind == types.KindChan || rhs.Kind == types.KindSeries {
		return occursIn(lhs, *rhs.ValueType)
	}
	return false
}

// defaultTypeForConstraint returns a default concrete type for a constraint
func defaultTypeForConstraint(constraint types.Type) types.Type {
	if constraint.Kind == types.KindNumericConstant {
		return types.F64()
	}
	return constraint
}

func (s *System) String() string {
	result := "=== Type Unification ===\n"
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
	result += fmt.Sprintf("\nSubstitutions (%d):\n", len(s.substitutions))
	for name, t := range s.substitutions {
		result += fmt.Sprintf("  %s => %v\n", name, t)
	}
	return result
}
