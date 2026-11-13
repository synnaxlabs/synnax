// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package validation provides utilities for parsing @validate domain expressions.
// It extracts validation rules like required, min/max length, numeric bounds, and
// defaults.
package validation

import (
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// Rules contains the parsed validation constraints from a @validate domain.
type Rules struct {
	MinLength      *int64
	MaxLength      *int64
	Min            *Number
	GT             *Number
	Max            *Number
	Pattern        *string
	PatternMessage *string
	Required       bool
	// Skip excludes the field from generated validation entirely, including recursion
	// into it. Used for fields that hold a reference to another entity (e.g. a parent
	// key or label keys) rather than embedded data to validate.
	Skip bool
}

// Number represents a numeric constraint value that can be int or float.
type Number struct {
	Int   int64
	Float float64
	IsInt bool
}

func newNumber(v resolution.ExpressionValue) *Number {
	return &Number{
		Int:   v.IntValue,
		Float: v.FloatValue,
		IsInt: v.Kind == resolution.ValueKindInt,
	}
}

// Parse extracts validation rules from a @validate domain entry.
// Returns nil if the domain is nil.
func Parse(domain resolution.Domain) *Rules {
	rules := &Rules{}
	for _, expr := range domain.Expressions {
		if len(expr.Values) == 0 {
			switch expr.Name {
			case "required":
				rules.Required = true
			case "skip":
				rules.Skip = true
			}
			continue
		}
		v := expr.Values[0]
		switch expr.Name {
		case "min_length":
			rules.MinLength = &v.IntValue
		case "max_length":
			rules.MaxLength = &v.IntValue
		case "min":
			rules.Min = newNumber(v)
		case "gt":
			rules.GT = newNumber(v)
		case "max":
			rules.Max = newNumber(v)
		case "pattern":
			rules.Pattern = &v.StringValue
			if len(expr.Values) > 1 {
				rules.PatternMessage = &expr.Values[1].StringValue
			}
		}
	}
	return rules
}

// FieldRules returns the rules that apply to field: the rules of each type in its alias
// and distinct chain, merged with the field's own. Merged rules keep the tighter bound,
// so a field rule can only add constraints. Skip comes from the field alone.
func FieldRules(field resolution.Field, table *resolution.Table) *Rules {
	rules := Parse(field.Domains["validate"])
	ref := field.Type
	visited := set.New[string]()
	for !ref.IsTypeParam() && !resolution.IsPrimitive(ref.Name) &&
		!visited.Contains(ref.Name) {
		visited.Add(ref.Name)
		t, ok := table.Get(ref.Name)
		if !ok {
			break
		}
		if domain, ok := t.Domains["validate"]; ok {
			typeRules := Parse(domain)
			typeRules.Skip = false
			rules = merge(rules, typeRules)
		}
		switch form := t.Form.(type) {
		case resolution.AliasForm:
			ref = form.Target
		case resolution.DistinctForm:
			ref = form.Base
		default:
			return rules
		}
	}
	return rules
}

// merge returns the rules that hold when both a and b hold. A bound that another bound
// implies is dropped. The analyzer rejects a field pattern whose type sets one, so at
// most one of a and b sets a pattern.
func merge(a, b *Rules) *Rules {
	out := *a
	out.Required = a.Required || b.Required
	out.Skip = a.Skip || b.Skip
	out.MinLength = tighterLength(a.MinLength, b.MinLength, true)
	out.MaxLength = tighterLength(a.MaxLength, b.MaxLength, false)
	out.Min = tighterNumber(a.Min, b.Min, true)
	out.GT = tighterNumber(a.GT, b.GT, true)
	out.Max = tighterNumber(a.Max, b.Max, false)
	if out.Min != nil && out.GT != nil {
		if out.GT.value() >= out.Min.value() {
			out.Min = nil
		} else {
			out.GT = nil
		}
	}
	if out.Pattern == nil {
		out.Pattern, out.PatternMessage = b.Pattern, b.PatternMessage
	}
	return &out
}

func tighterLength(a, b *int64, lower bool) *int64 {
	if a == nil || (b != nil && (*b > *a) == lower && *b != *a) {
		return b
	}
	return a
}

func tighterNumber(a, b *Number, lower bool) *Number {
	if a == nil || (b != nil && (b.value() > a.value()) == lower &&
		b.value() != a.value()) {
		return b
	}
	return a
}

func (n Number) value() float64 {
	if n.IsInt {
		return float64(n.Int)
	}
	return n.Float
}

// IsEmpty returns true if the rules have no validation constraints set.
func IsEmpty(r *Rules) bool {
	if r == nil {
		return true
	}
	return !r.Required && r.MinLength == nil && r.MaxLength == nil &&
		r.Min == nil && r.GT == nil && r.Max == nil && r.Pattern == nil
}
