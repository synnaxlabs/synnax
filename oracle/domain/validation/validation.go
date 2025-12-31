// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validation

import "github.com/synnaxlabs/oracle/resolution"

type Rules struct {
	Required  bool
	MinLength *int64
	MaxLength *int64
	Pattern   *string
	Min       *Number
	Max       *Number
	Email     bool
	URL       bool
	Default   *resolution.ExpressionValue
}

type Number struct {
	Int   int64
	Float float64
	IsInt bool
}

func Parse(domain *resolution.DomainEntry) *Rules {
	if domain == nil {
		return nil
	}
	rules := &Rules{}
	for _, expr := range domain.Expressions {
		if len(expr.Values) == 0 {
			switch expr.Name {
			case "required":
				rules.Required = true
			case "email":
				rules.Email = true
			case "url":
				rules.URL = true
			}
			continue
		}
		v := expr.Values[0]
		switch expr.Name {
		case "min_length":
			rules.MinLength = &v.IntValue
		case "max_length":
			rules.MaxLength = &v.IntValue
		case "pattern":
			rules.Pattern = &v.StringValue
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
		}
	}
	return rules
}

func IsEmpty(r *Rules) bool {
	if r == nil {
		return true
	}
	return !r.Required && r.MinLength == nil && r.MaxLength == nil &&
		r.Pattern == nil && r.Min == nil && r.Max == nil &&
		!r.Email && !r.URL && r.Default == nil
}
