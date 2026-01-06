// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package validation provides utilities for parsing @validate domain expressions.
// It extracts validation rules like required, min/max length, and defaults.
package validation

import "github.com/synnaxlabs/oracle/resolution"

// Rules contains the parsed validation constraints from a @validate domain.
type Rules struct {
	Required       bool
	MinLength      *int64
	MaxLength      *int64
	Min            *Number
	Max            *Number
	Default        *resolution.ExpressionValue
	Pattern        *string
	PatternMessage *string
}

// Number represents a numeric constraint value that can be int or float.
type Number struct {
	Int   int64
	Float float64
	IsInt bool
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
			rules.Min = &Number{
				Int:   v.IntValue,
				Float: v.FloatValue,
				IsInt: v.Kind == resolution.ValueKindInt,
			}
		case "max":
			rules.Max = &Number{
				Int:   v.IntValue,
				Float: v.FloatValue,
				IsInt: v.Kind == resolution.ValueKindInt,
			}
		case "default":
			rules.Default = &v
		case "pattern":
			rules.Pattern = &v.StringValue
			if len(expr.Values) > 1 {
				rules.PatternMessage = &expr.Values[1].StringValue
			}
		}
	}
	return rules
}

// IsEmpty returns true if the rules have no validation constraints set.
func IsEmpty(r *Rules) bool {
	if r == nil {
		return true
	}
	return !r.Required && r.MinLength == nil && r.MaxLength == nil &&
		r.Min == nil && r.Max == nil && r.Default == nil && r.Pattern == nil
}
