// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package constraints implements type constraint collection and unification.
package constraints

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/types"
)

// Kind classifies constraint relationships between types.
type Kind int

const (
	// KindEquality requires exact type equality after substitution.
	KindEquality Kind = iota
	// KindCompatible allows numeric type promotion between compatible types.
	KindCompatible
)

// Constraint represents a type relationship that must hold for successful type checking.
type Constraint struct {
	Kind   Kind
	Left   types.Type
	Right  types.Type
	Source antlr.ParserRuleContext
	Reason string
}

// System accumulates type constraints and computes substitutions via unification.
type System struct {
	Constraints   []Constraint
	Substitutions map[string]types.Type
	TypeVars      map[string]types.Type
}

// New creates an empty constraint system.
func New() *System {
	return &System{
		Constraints:   make([]Constraint, 0),
		Substitutions: make(map[string]types.Type),
		TypeVars:      make(map[string]types.Type),
	}
}

// AddEquality adds an equality constraint requiring left and right to unify to the same type.
func (s *System) AddEquality(
	left, right types.Type,
	source antlr.ParserRuleContext,
	reason string,
) {
	s.recordTypeVars(left, right)
	s.Constraints = append(s.Constraints, Constraint{
		Kind:   KindEquality,
		Left:   left,
		Right:  right,
		Source: source,
		Reason: reason,
	})
}

// AddCompatible adds a compatibility constraint allowing numeric promotion between left and right.
func (s *System) AddCompatible(
	left, right types.Type,
	source antlr.ParserRuleContext,
	reason string,
) {
	s.recordTypeVars(left, right)
	s.Constraints = append(s.Constraints, Constraint{
		Kind:   KindCompatible,
		Left:   left,
		Right:  right,
		Source: source,
		Reason: reason,
	})
}

func (s *System) recordTypeVars(toRecord ...types.Type) {
	for _, t := range toRecord {
		if t.Kind == types.KindVariable {
			if _, exists := s.TypeVars[t.Name]; !exists {
				s.TypeVars[t.Name] = t
			}
		}
		if t.Kind == types.KindChan || t.Kind == types.KindSeries {
			s.recordTypeVars(t.Unwrap())
		}
	}
}

// HasTypeVariables returns true if the system has recorded any type variables.
func (s *System) HasTypeVariables() bool {
	return len(s.TypeVars) > 0
}

// ApplySubstitutions replaces type variables in t with their computed substitutions.
func (s *System) ApplySubstitutions(t types.Type) types.Type {
	return s.applySubstitutionsWithVisited(t, make(map[string]bool))
}

func (s *System) applySubstitutionsWithVisited(t types.Type, visited map[string]bool) types.Type {
	if t.Kind == types.KindVariable {
		if visited[t.Name] {
			return t
		}
		if sub, exists := s.Substitutions[t.Name]; exists {
			visited[t.Name] = true
			result := s.applySubstitutionsWithVisited(sub, visited)
			visited[t.Name] = false
			// Preserve unit from the original type variable if it had one
			if t.Unit != nil && result.Unit == nil {
				result.Unit = t.Unit
			}
			return result
		}
		return t
	}
	if t.Kind == types.KindChan || t.Kind == types.KindSeries {
		freshValue := s.applySubstitutionsWithVisited(t.Unwrap(), visited)
		if t.Kind == types.KindChan {
			return types.Chan(freshValue)
		}
		return types.Series(freshValue)
	}

	if t.Kind == types.KindFunction {
		props := t.Copy()
		for i, p := range t.Inputs {
			props.Inputs[i].Type = s.applySubstitutionsWithVisited(p.Type, visited)
		}
		for i, p := range t.Outputs {
			props.Outputs[i].Type = s.applySubstitutionsWithVisited(p.Type, visited)
		}
		for i, p := range t.Config {
			props.Config[i].Type = s.applySubstitutionsWithVisited(p.Type, visited)
		}
		return types.Function(props)
	}
	return t
}
