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

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/maps"
)

// Kind represents the type of constraint
type Kind int

const (
	// KindEquality means two types must be equal
	KindEquality Kind = iota
	// KindCompatible means types must be compatible (e.g., for numeric promotion)
	KindCompatible
)

// Constraint represents a type constraint between two types
type Constraint struct {
	Kind   Kind
	Left   ir.Type
	Right  ir.Type
	Source antlr.ParserRuleContext // Where this constraint came from (for error reporting)
	Reason string                  // Human-readable reason for this constraint
}

// System manages type constraints and their resolution
type System struct {
	constraints []Constraint
	// Substitutions maps type variable names to resolved types
	substitutions map[string]ir.Type
	// Track which type variables we've seen
	typeVars map[string]*ir.TypeVariable
}

// New creates a new constraint system
func New() *System {
	return &System{
		constraints:   make([]Constraint, 0),
		substitutions: make(map[string]ir.Type),
		typeVars:      make(map[string]*ir.TypeVariable),
	}
}

// AddEquality adds an equality constraint between two types
func (s *System) AddEquality(left, right ir.Type, source antlr.ParserRuleContext, reason string) {
	s.recordTypeVars(left, right)
	s.constraints = append(s.constraints, Constraint{
		Kind:   KindEquality,
		Left:   left,
		Right:  right,
		Source: source,
		Reason: reason,
	})
}

// AddCompatible adds a compatibility constraint (types must be compatible for operations)
func (s *System) AddCompatible(left, right ir.Type, source antlr.ParserRuleContext, reason string) {
	s.recordTypeVars(left, right)
	s.constraints = append(s.constraints, Constraint{
		Kind:   KindCompatible,
		Left:   left,
		Right:  right,
		Source: source,
		Reason: reason,
	})
}

// recordTypeVars tracks any type variables we see
func (s *System) recordTypeVars(types ...ir.Type) {
	for _, t := range types {
		if tv, ok := t.(ir.TypeVariable); ok {
			if _, exists := s.typeVars[tv.Name]; !exists {
				s.typeVars[tv.Name] = &tv
			}
		}
		// Also check inside compound types like Chan or Series
		if ch, ok := t.(ir.Chan); ok {
			s.recordTypeVars(ch.ValueType)
		}
		if series, ok := t.(ir.Series); ok {
			s.recordTypeVars(series.ValueType)
		}
	}
}

// GetSubstitution returns the resolved type for a type variable, if any
func (s *System) GetSubstitution(name string) (ir.Type, bool) {
	t, ok := s.substitutions[name]
	return t, ok
}

// SetSubstitution records a resolved type for a type variable
func (s *System) SetSubstitution(name string, t ir.Type) {
	s.substitutions[name] = t
}

// Constraints returns all collected constraints
func (s *System) Constraints() []Constraint {
	return s.constraints
}

// TypeVariables returns all type variables seen by the system
func (s *System) TypeVariables() map[string]*ir.TypeVariable {
	return s.typeVars
}

// HasTypeVariables returns true if the system has encountered any type variables
func (s *System) HasTypeVariables() bool {
	return len(s.typeVars) > 0
}

// ApplySubstitutions replaces all type variables in a type with their resolved types
func (s *System) ApplySubstitutions(t ir.Type) ir.Type {
	visited := make(map[string]bool)
	return s.applySubstitutionsWithVisited(t, visited)
}

// applySubstitutionsWithVisited is the internal recursive function with cycle detection
func (s *System) applySubstitutionsWithVisited(t ir.Type, visited map[string]bool) ir.Type {
	if t == nil {
		return nil
	}

	// If it's a type variable, substitute it
	if tv, ok := t.(ir.TypeVariable); ok {
		// Check for cycles
		if visited[tv.Name] {
			// We've seen this type variable before in this recursion - stop to avoid infinite loop
			return t
		}

		if sub, exists := s.substitutions[tv.Name]; exists {
			// Mark as visited before recursing
			visited[tv.Name] = true
			// Recursively apply substitutions in case the substitution contains type variables
			result := s.applySubstitutionsWithVisited(sub, visited)
			// Unmark after recursion
			visited[tv.Name] = false
			return result
		}
		// No substitution found, return as-is (will be caught as error later)
		return t
	}

	// Handle compound types
	if ch, ok := t.(ir.Chan); ok {
		return ir.Chan{ValueType: s.applySubstitutionsWithVisited(ch.ValueType, visited)}
	}
	if series, ok := t.(ir.Series); ok {
		return ir.Series{ValueType: s.applySubstitutionsWithVisited(series.ValueType, visited)}
	}

	// Handle Stage type - resolve type variables in params, config, and return
	if stage, ok := t.(ir.Stage); ok {
		// Create new maps for config and params with resolved types
		newConfig := &maps.Ordered[string, ir.Type]{}
		for k, v := range stage.Config.Iter() {
			newConfig.Put(k, s.applySubstitutionsWithVisited(v, visited))
		}

		newParams := &maps.Ordered[string, ir.Type]{}
		for k, v := range stage.Params.Iter() {
			newParams.Put(k, s.applySubstitutionsWithVisited(v, visited))
		}

		return ir.Stage{
			Key:               stage.Key,
			Config:            *newConfig,
			Params:            *newParams,
			Return:            s.applySubstitutionsWithVisited(stage.Return, visited),
			StatefulVariables: stage.StatefulVariables,
			Channels:          stage.Channels,
			Body:              stage.Body,
		}
	}

	// Handle Function type - resolve type variables in params and return
	if fn, ok := t.(ir.Function); ok {
		newParams := &maps.Ordered[string, ir.Type]{}
		for k, v := range fn.Params.Iter() {
			newParams.Put(k, s.applySubstitutionsWithVisited(v, visited))
		}

		return ir.Function{
			Key:    fn.Key,
			Params: *newParams,
			Return: s.applySubstitutionsWithVisited(fn.Return, visited),
			Body:   fn.Body,
		}
	}

	// Concrete type, return as-is
	return t
}

// String provides a debug representation of the constraint system
func (s *System) String() string {
	result := "Constraint System:\n"
	result += fmt.Sprintf("  Type Variables: %d\n", len(s.typeVars))
	for name, tv := range s.typeVars {
		result += fmt.Sprintf("    %s: %v\n", name, tv.Constraint)
	}
	result += fmt.Sprintf("  Constraints: %d\n", len(s.constraints))
	for _, c := range s.constraints {
		kindStr := "="
		if c.Kind == KindCompatible {
			kindStr = "~"
		}
		result += fmt.Sprintf("    %v %s %v (%s)\n", c.Left, kindStr, c.Right, c.Reason)
	}
	result += fmt.Sprintf("  Substitutions: %d\n", len(s.substitutions))
	for name, t := range s.substitutions {
		result += fmt.Sprintf("    %s -> %v\n", name, t)
	}
	return result
}
