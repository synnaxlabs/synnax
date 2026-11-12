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
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
)

// Kind classifies constraint relationships between types.
type Kind int

const (
	// kindEquality requires exact type equality after substitution.
	kindEquality Kind = iota
	// kindCompatible allows numeric type promotion between compatible types.
	kindCompatible
)

// TypeOrigin tracks where a type was inferred for error reporting.
type TypeOrigin struct {
	// Source is the AST node where inference occurred.
	Source antlr.ParserRuleContext
	// SymbolName is the variable or parameter name, if applicable.
	SymbolName string
	// InferredAs is a human-readable description of the inferred type.
	InferredAs string
}

// Constraint represents a type relationship that must hold for successful type
// checking.
type Constraint struct {
	// Source is the AST node that generated this constraint for error reporting.
	Source antlr.ParserRuleContext
	// Reason describes why this constraint exists for debugging.
	Reason string
	// Left is the first type in the relationship.
	Left types.Type
	// Right is the second type in the relationship.
	Right types.Type
	// LeftOrigin tracks where the left type was inferred.
	LeftOrigin *TypeOrigin
	// RightOrigin tracks where the right type was inferred.
	RightOrigin *TypeOrigin
	// Kind classifies the constraint as equality or compatibility.
	Kind Kind
}

// System accumulates type constraints and computes substitutions via unification.
type System struct {
	// Substitutions maps type variable names to their resolved types.
	Substitutions map[string]types.Type
	// TypeVars tracks all type variables encountered during constraint collection.
	TypeVars map[string]types.Type
	// Defaults maps a type variable to the type it takes when no constraint binds it,
	// in place of its constraint's default.
	Defaults map[string]types.Type
	// openNumerics holds the numeric type variables that stand for an open series
	// element.
	openNumerics set.Set[string]
	// Constraints holds all type relationships to be solved.
	Constraints []Constraint
}

// New creates an empty constraint system.
func New() *System {
	return &System{
		Constraints:   make([]Constraint, 0),
		Substitutions: make(map[string]types.Type),
		TypeVars:      make(map[string]types.Type),
		Defaults:      make(map[string]types.Type),
		openNumerics:  set.New[string](),
	}
}

// AddDefault records the type tv takes when no constraint binds it.
func (s *System) AddDefault(tv, t types.Type) {
	s.recordTypeVars(tv)
	s.Defaults[tv.Name] = t
}

// OpenNumeric returns a numeric type variable for the open series element elem. An
// integer literal unified with it takes its type, so a later float still fits.
func (s *System) OpenNumeric(elem types.Type) types.Type {
	numeric := types.NumericConstraint()
	tv := types.Variable(elem.Name+"_numeric", &numeric)
	s.openNumerics.Add(tv.Name)
	return tv
}

// AddEquality adds an equality constraint requiring left and right to unify to the
// same type. It immediately attempts unification and returns an error if the types
// are incompatible. If unification fails, the constraint is not added to avoid
// duplicate errors during batch unification.
func (s *System) AddEquality(
	left, right types.Type,
	source antlr.ParserRuleContext,
	reason string,
) error {
	s.recordTypeVars(left, right)
	constraint := Constraint{
		Kind:   kindEquality,
		Left:   left,
		Right:  right,
		Source: source,
		Reason: reason,
	}
	if err := s.UnifyConstraint(constraint); err != nil {
		return err
	}
	s.Constraints = append(s.Constraints, constraint)
	return nil
}

// AddCompatible adds a compatibility constraint allowing numeric promotion between
// left and right. It immediately attempts unification and returns an error if the
// types are incompatible. If unification fails, the constraint is not added to avoid
// duplicate errors during batch unification.
func (s *System) AddCompatible(
	left, right types.Type,
	source antlr.ParserRuleContext,
	reason string,
) error {
	s.recordTypeVars(left, right)
	constraint := Constraint{
		Kind:   kindCompatible,
		Left:   left,
		Right:  right,
		Source: source,
		Reason: reason,
	}
	if err := s.UnifyConstraint(constraint); err != nil {
		return err
	}
	s.Constraints = append(s.Constraints, constraint)
	return nil
}

// AddVariable records the type variables in t, so Unify resolves each one, with its
// default when no other constraint binds it.
func (s *System) AddVariable(t types.Type) {
	s.recordTypeVars(t)
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
	return s.applySubstitutions(t, make(set.Set[string]))
}

func (s *System) applySubstitutionsToParams(
	t types.Params,
	visited set.Set[string],
) types.Params {
	t2 := slices.Clone(t)
	for i, p := range t2 {
		t2[i].Type = s.applySubstitutions(p.Type, visited)
	}
	return t2
}

func (s *System) applySubstitutions(t types.Type, visited set.Set[string]) types.Type {
	if t.Kind == types.KindVariable {
		if visited.Contains(t.Name) {
			return t
		}
		if sub, exists := s.Substitutions[t.Name]; exists {
			visited.Add(t.Name)
			result := s.applySubstitutions(sub, visited)
			visited.Remove(t.Name)
			// A type variable keeps its own unit, which is the unit of its literal.
			if t.Unit != nil {
				result.Unit = t.Unit
			}
			return result
		}
		return t
	}
	if t.Kind == types.KindChan || t.Kind == types.KindSeries {
		freshValue := s.applySubstitutions(t.Unwrap(), visited)
		if t.Kind == types.KindChan {
			fresh := types.Chan(freshValue)
			fresh.ChanDirection = t.ChanDirection
			return fresh
		}
		return types.Series(freshValue)
	}

	if t.Kind == types.KindFunction {
		return types.Function(types.FunctionProperties{
			Inputs:  s.applySubstitutionsToParams(t.Inputs, visited),
			Outputs: s.applySubstitutionsToParams(t.Outputs, visited),
		})
	}
	return t
}
