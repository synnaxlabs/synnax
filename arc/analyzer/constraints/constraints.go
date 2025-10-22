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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/types"
)

type Kind int

const (
	KindEquality Kind = iota
	KindCompatible
)

type Constraint struct {
	Kind   Kind
	Left   types.Type
	Right  types.Type
	Source antlr.ParserRuleContext
	Reason string
}

type System struct {
	constraints   []Constraint
	substitutions map[string]types.Type
	typeVars      map[string]types.Type
}

func New() *System {
	return &System{
		constraints:   make([]Constraint, 0),
		substitutions: make(map[string]types.Type),
		typeVars:      make(map[string]types.Type),
	}
}

func (s *System) AddEquality(
	left, right types.Type,
	source antlr.ParserRuleContext,
	reason string,
) {
	s.recordTypeVars(left, right)
	s.constraints = append(s.constraints, Constraint{
		Kind:   KindEquality,
		Left:   left,
		Right:  right,
		Source: source,
		Reason: reason,
	})
}

func (s *System) AddCompatible(
	left, right types.Type,
	source antlr.ParserRuleContext,
	reason string,
) {
	s.recordTypeVars(left, right)
	s.constraints = append(s.constraints, Constraint{
		Kind:   KindCompatible,
		Left:   left,
		Right:  right,
		Source: source,
		Reason: reason,
	})
}

func (s *System) recordTypeVars(toRecord ...types.Type) {
	for _, t := range toRecord {
		if t.Kind == types.KindTypeVariable {
			if _, exists := s.typeVars[t.Name]; !exists {
				s.typeVars[t.Name] = t
			}
		}
		if t.Kind == types.KindChan || t.Kind == types.KindSeries {
			s.recordTypeVars(t.Unwrap())
		}
	}
}

func (s *System) GetSubstitution(name string) (types.Type, bool) {
	t, ok := s.substitutions[name]
	return t, ok
}

func (s *System) SetSubstitution(name string, t types.Type) {
	s.substitutions[name] = t
}

func (s *System) Constraints() []Constraint {
	return s.constraints
}

func (s *System) TypeVariables() map[string]types.Type {
	return s.typeVars
}

func (s *System) HasTypeVariables() bool {
	return len(s.typeVars) > 0
}

func (s *System) ApplySubstitutions(t types.Type) types.Type {
	return s.applySubstitutionsWithVisited(t, make(map[string]bool))
}

func (s *System) applySubstitutionsWithVisited(t types.Type, visited map[string]bool) types.Type {
	if t.Kind == types.KindTypeVariable {
		if visited[t.Name] {
			return t
		}
		if sub, exists := s.substitutions[t.Name]; exists {
			visited[t.Name] = true
			result := s.applySubstitutionsWithVisited(sub, visited)
			visited[t.Name] = false
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
		props := types.NewFunctionProperties()
		for k, v := range t.Inputs.Iter() {
			props.Inputs.Put(k, s.applySubstitutionsWithVisited(v, visited))
		}
		for k, v := range t.Outputs.Iter() {
			props.Outputs.Put(k, s.applySubstitutionsWithVisited(v, visited))
		}
		for k, v := range t.Config.Iter() {
			props.Config.Put(k, s.applySubstitutionsWithVisited(v, visited))
		}
		return types.Function(props)
	}
	return t
}
