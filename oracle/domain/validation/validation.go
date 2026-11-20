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
	"cmp"

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
	// Finite refuses NaN and the infinities in a floating point field.
	Finite bool
	// LessThan requires the field to be less than another field of the same struct.
	LessThan *Comparison
	// Skip excludes the field from generated validation entirely, including recursion
	// into it. Used for fields that hold a reference to another entity (e.g. a parent
	// key or label keys) rather than embedded data to validate.
	Skip bool
}

// Comparison is a rule that compares a field to another field of its struct.
type Comparison struct {
	// Field is the schema name of the other field.
	Field string
	// Message replaces the generated error message when it is not empty.
	Message string
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
			case "finite":
				rules.Finite = true
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
		case "lt":
			rules.LessThan = &Comparison{Field: v.IdentValue}
			if len(expr.Values) > 1 {
				rules.LessThan.Message = expr.Values[1].StringValue
			}
		case "pattern":
			rules.Pattern = &v.StringValue
			if len(expr.Values) > 1 {
				rules.PatternMessage = &expr.Values[1].StringValue
			}
		}
	}
	return rules
}

// LessThan returns the lt rule that domains set for lang: one set with `@validate lt`,
// else one set with `@<lang> validate lt`. It returns nil when neither sets one.
func LessThan(domains map[string]resolution.Domain, lang string) *Comparison {
	if lt := Parse(domains["validate"]).LessThan; lt != nil {
		return lt
	}
	expr, ok := domains[lang].Expressions.Find("validate")
	if !ok || len(expr.Values) == 0 {
		return nil
	}
	scoped := resolution.Expression{
		Name:   expr.Values[0].IdentValue,
		Values: expr.Values[1:],
	}
	return Parse(
		resolution.Domain{Expressions: resolution.Expressions{scoped}},
	).LessThan
}

// FieldRules returns the rules that apply to field: its own rules merged with the
// TypeRules of its type chain. Skip comes from the field alone.
func FieldRules(field resolution.Field, table *resolution.Table) *Rules {
	return Merge(
		Parse(field.Domains["validate"]),
		TypeRules(TypeChain(field.Type, table)),
	)
}

// TypeChain returns the type that ref names, followed by each type its aliases and
// distinct bases reach. It stops at a primitive, a type parameter, or a cycle.
func TypeChain(ref resolution.TypeRef, table *resolution.Table) []resolution.Type {
	var chain []resolution.Type
	visited := set.New[string]()
	for !ref.IsTypeParam() && !resolution.IsPrimitive(ref.Name) &&
		!visited.Contains(ref.Name) {
		visited.Add(ref.Name)
		t, ok := table.Get(ref.Name)
		if !ok {
			break
		}
		chain = append(chain, t)
		switch form := t.Form.(type) {
		case resolution.AliasForm:
			ref = form.Target
		case resolution.DistinctForm:
			ref = form.Base
		default:
			return chain
		}
	}
	return chain
}

// TypeRules returns the merged type-level rules of types. Skip and LessThan never apply
// to a type.
func TypeRules(types []resolution.Type) *Rules {
	rules := &Rules{}
	for _, t := range types {
		if domain, ok := t.Domains["validate"]; ok {
			rules = Merge(rules, Parse(domain))
		}
	}
	rules.Skip = false
	rules.LessThan = nil
	return rules
}

// Merge returns the rules that hold when both a and b hold. A merged bound keeps the
// tighter value, and a bound that another bound implies is dropped, so merging can only
// add constraints. The analyzer rejects a second pattern on a field or in a type chain,
// so at most one of a and b sets a pattern.
func Merge(a, b *Rules) *Rules {
	out := *a
	out.Required = a.Required || b.Required
	out.Finite = a.Finite || b.Finite
	out.Skip = a.Skip || b.Skip
	if out.LessThan == nil {
		out.LessThan = b.LessThan
	}
	out.MinLength = tighterLength(a.MinLength, b.MinLength, true)
	out.MaxLength = tighterLength(a.MaxLength, b.MaxLength, false)
	out.Min = tighterNumber(a.Min, b.Min, true)
	out.GT = tighterNumber(a.GT, b.GT, true)
	out.Max = tighterNumber(a.Max, b.Max, false)
	if out.Min != nil && out.GT != nil {
		if out.GT.Compare(*out.Min) >= 0 {
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
	if a == nil || (b != nil && b.Compare(*a) != 0 && (b.Compare(*a) > 0) == lower) {
		return b
	}
	return a
}

// Compare returns -1, 0, or 1 when n is less than, equal to, or greater than o. Two
// integers compare exactly.
func (n Number) Compare(o Number) int {
	if n.IsInt && o.IsInt {
		return cmp.Compare(n.Int, o.Int)
	}
	return cmp.Compare(n.float(), o.float())
}

func (n Number) float() float64 {
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
	return !r.Required && !r.Finite && r.MinLength == nil && r.MaxLength == nil &&
		r.Min == nil && r.GT == nil && r.Max == nil && r.Pattern == nil &&
		r.LessThan == nil
}
