// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Parse", func() {
	It("should parse required expression", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{{Name: "required"}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Required).To(BeTrue())
	})

	It("should parse skip expression", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{{Name: "skip"}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Skip).To(BeTrue())
	})

	It("should parse min_length expression", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{{
				Name:   "min_length",
				Values: []resolution.ExpressionValue{{IntValue: 5}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.MinLength).NotTo(BeNil())
		Expect(*rules.MinLength).To(Equal(int64(5)))
	})

	It("should parse max_length expression", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{{
				Name:   "max_length",
				Values: []resolution.ExpressionValue{{IntValue: 100}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.MaxLength).NotTo(BeNil())
		Expect(*rules.MaxLength).To(Equal(int64(100)))
	})

	It("should parse min expression with int", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{
				{
					Name: "min",
					Values: []resolution.ExpressionValue{
						{Kind: resolution.ValueKindInt, IntValue: 10},
					},
				},
			},
		}
		rules := validation.Parse(domain)
		Expect(rules.Min).NotTo(BeNil())
		Expect(rules.Min.IsInt).To(BeTrue())
		Expect(rules.Min.Int).To(Equal(int64(10)))
	})

	It("should parse gt expression with int", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{{
				Name: "gt",
				Values: []resolution.ExpressionValue{
					{Kind: resolution.ValueKindInt, IntValue: 0},
				},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.GT).NotTo(BeNil())
		Expect(rules.GT.IsInt).To(BeTrue())
		Expect(rules.GT.Int).To(Equal(int64(0)))
		Expect(rules.Min).To(BeNil())
		Expect(validation.IsEmpty(rules)).To(BeFalse())
	})

	It("should parse gt expression with float", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{{
				Name: "gt",
				Values: []resolution.ExpressionValue{
					{Kind: resolution.ValueKindFloat, FloatValue: 0.5},
				},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.GT).NotTo(BeNil())
		Expect(rules.GT.IsInt).To(BeFalse())
		Expect(rules.GT.Float).To(Equal(0.5))
	})

	It("should parse min expression with float", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{
				{
					Name: "min",
					Values: []resolution.ExpressionValue{
						{Kind: resolution.ValueKindFloat, FloatValue: 1.5},
					},
				},
			},
		}
		rules := validation.Parse(domain)
		Expect(rules.Min).NotTo(BeNil())
		Expect(rules.Min.IsInt).To(BeFalse())
		Expect(rules.Min.Float).To(Equal(1.5))
	})

	It("should parse max expression", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{
				{
					Name: "max",
					Values: []resolution.ExpressionValue{
						{Kind: resolution.ValueKindInt, IntValue: 100},
					},
				},
			},
		}
		rules := validation.Parse(domain)
		Expect(rules.Max).NotTo(BeNil())
		Expect(rules.Max.Int).To(Equal(int64(100)))
	})

	It("should parse multiple expressions", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{
				{Name: "required"},
				{
					Name:   "min_length",
					Values: []resolution.ExpressionValue{{IntValue: 1}},
				},
				{
					Name:   "max_length",
					Values: []resolution.ExpressionValue{{IntValue: 50}},
				},
			},
		}
		rules := validation.Parse(domain)
		Expect(rules.Required).To(BeTrue())
		Expect(*rules.MinLength).To(Equal(int64(1)))
		Expect(*rules.MaxLength).To(Equal(int64(50)))
	})

	It("should parse pattern expression without message", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{
				{
					Name: "pattern",
					Values: []resolution.ExpressionValue{
						{
							Kind:        resolution.ValueKindString,
							StringValue: "^[a-zA-Z_][a-zA-Z0-9_]*$",
						},
					},
				},
			},
		}
		rules := validation.Parse(domain)
		Expect(rules.Pattern).NotTo(BeNil())
		Expect(*rules.Pattern).To(Equal("^[a-zA-Z_][a-zA-Z0-9_]*$"))
		Expect(rules.PatternMessage).To(BeNil())
	})

	It("should parse pattern expression with message", func() {
		domain := resolution.Domain{
			Expressions: []resolution.Expression{{
				Name: "pattern",
				Values: []resolution.ExpressionValue{
					{
						Kind:        resolution.ValueKindString,
						StringValue: "^[a-zA-Z_][a-zA-Z0-9_]*$",
					},
					{
						Kind:        resolution.ValueKindString,
						StringValue: "Name must be a valid identifier",
					},
				},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Pattern).NotTo(BeNil())
		Expect(*rules.Pattern).To(Equal("^[a-zA-Z_][a-zA-Z0-9_]*$"))
		Expect(rules.PatternMessage).NotTo(BeNil())
		Expect(*rules.PatternMessage).To(Equal("Name must be a valid identifier"))
	})
})

var _ = Describe("IsEmpty", func() {
	It("should return true for nil", func() {
		Expect(validation.IsEmpty(nil)).To(BeTrue())
	})

	It("should return true for empty rules", func() {
		Expect(validation.IsEmpty(&validation.Rules{})).To(BeTrue())
	})

	It("should return false when required is set", func() {
		Expect(validation.IsEmpty(&validation.Rules{Required: true})).To(BeFalse())
	})

	It("should return false when any field is set", func() {
		minLen := int64(1)
		Expect(validation.IsEmpty(&validation.Rules{MinLength: &minLen})).To(BeFalse())
	})

	It("should return false when pattern is set", func() {
		pattern := "^[a-z]+$"
		Expect(validation.IsEmpty(&validation.Rules{Pattern: &pattern})).To(BeFalse())
	})
})

func rule(name string, values ...resolution.ExpressionValue) resolution.Domain {
	return resolution.Domain{
		Expressions: []resolution.Expression{{Name: name, Values: values}},
	}
}

func intValue(v int64) resolution.ExpressionValue {
	return resolution.ExpressionValue{Kind: resolution.ValueKindInt, IntValue: v}
}

func floatValue(v float64) resolution.ExpressionValue {
	return resolution.ExpressionValue{Kind: resolution.ValueKindFloat, FloatValue: v}
}

func parse(name string, values ...resolution.ExpressionValue) *validation.Rules {
	return validation.Parse(rule(name, values...))
}

// newChainTable returns a table holding Rate (distinct over float64, min 0), Hz (alias
// of Rate, max 100), and a cycle between Loop and Back.
func newChainTable() *resolution.Table {
	table := resolution.NewTable()
	types := []resolution.Type{
		{
			Name:          "Rate",
			QualifiedName: "Rate",
			Form: resolution.DistinctForm{
				Base: resolution.TypeRef{Name: "float64"},
			},
			Domains: map[string]resolution.Domain{
				"validate": rule("min", intValue(0)),
			},
		},
		{
			Name:          "Hz",
			QualifiedName: "Hz",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "Rate"},
			},
			Domains: map[string]resolution.Domain{
				"validate": rule("max", intValue(100)),
			},
		},
		{
			Name:          "Loop",
			QualifiedName: "Loop",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "Back"},
			},
		},
		{
			Name:          "Back",
			QualifiedName: "Back",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "Loop"},
			},
		},
	}
	for _, t := range types {
		Expect(table.Add(t)).To(Succeed())
	}
	return table
}

func names(types []resolution.Type) []string {
	out := make([]string, len(types))
	for i, t := range types {
		out[i] = t.Name
	}
	return out
}

var _ = Describe("TypeChain", func() {
	It("should follow aliases and distinct bases to a primitive", func() {
		chain := validation.TypeChain(resolution.TypeRef{Name: "Hz"}, newChainTable())
		Expect(names(chain)).To(Equal([]string{"Hz", "Rate"}))
	})

	It("should return nothing for a primitive", func() {
		chain := validation.TypeChain(
			resolution.TypeRef{Name: "float64"},
			newChainTable(),
		)
		Expect(chain).To(BeEmpty())
	})

	It("should return nothing for a type parameter", func() {
		ref := resolution.TypeRef{TypeParam: &resolution.TypeParam{Name: "T"}}
		Expect(validation.TypeChain(ref, newChainTable())).To(BeEmpty())
	})

	It("should stop at a cycle", func() {
		chain := validation.TypeChain(resolution.TypeRef{Name: "Loop"}, newChainTable())
		Expect(names(chain)).To(Equal([]string{"Loop", "Back"}))
	})
})

var _ = Describe("TypeRules", func() {
	It("should merge the rules of every type", func() {
		table := newChainTable()
		rules := validation.TypeRules(
			validation.TypeChain(resolution.TypeRef{Name: "Hz"}, table),
		)
		Expect(rules.Min.Int).To(Equal(int64(0)))
		Expect(rules.Max.Int).To(Equal(int64(100)))
	})

	It("should never skip", func() {
		skipped := resolution.Type{
			Domains: map[string]resolution.Domain{"validate": rule("skip")},
		}
		Expect(validation.TypeRules([]resolution.Type{skipped}).Skip).To(BeFalse())
	})
})

var _ = Describe("Merge", func() {
	It("should keep the higher lower bound", func() {
		merged := validation.Merge(parse("min", intValue(1)), parse("min", intValue(3)))
		Expect(merged.Min.Int).To(Equal(int64(3)))
	})

	It("should keep the lower upper bound across int and float values", func() {
		merged := validation.Merge(
			parse("max", intValue(1)),
			parse("max", floatValue(0.5)),
		)
		Expect(merged.Max.Float).To(Equal(0.5))
		Expect(merged.Max.IsInt).To(BeFalse())
	})

	It("should keep the first of two equal bounds", func() {
		merged := validation.Merge(
			parse("max", intValue(1)),
			parse("max", floatValue(1)),
		)
		Expect(merged.Max.IsInt).To(BeTrue())
	})

	It("should keep the tighter length bounds", func() {
		merged := validation.Merge(
			validation.Merge(
				parse("min_length", intValue(2)),
				parse("max_length", intValue(10)),
			),
			validation.Merge(
				parse("min_length", intValue(4)),
				parse("max_length", intValue(20)),
			),
		)
		Expect(*merged.MinLength).To(Equal(int64(4)))
		Expect(*merged.MaxLength).To(Equal(int64(10)))
	})

	It("should drop a min that a gt implies", func() {
		merged := validation.Merge(parse("min", intValue(0)), parse("gt", intValue(0)))
		Expect(merged.Min).To(BeNil())
		Expect(merged.GT.Int).To(Equal(int64(0)))
	})

	It("should drop a gt that a min implies", func() {
		merged := validation.Merge(parse("min", intValue(5)), parse("gt", intValue(0)))
		Expect(merged.GT).To(BeNil())
		Expect(merged.Min.Int).To(Equal(int64(5)))
	})

	It("should keep required from either side", func() {
		merged := validation.Merge(&validation.Rules{}, parse("required"))
		Expect(merged.Required).To(BeTrue())
	})

	It("should take a pattern from the side that sets one", func() {
		merged := validation.Merge(&validation.Rules{}, parse(
			"pattern",
			resolution.ExpressionValue{StringValue: "^[a-z]+$"},
		))
		Expect(*merged.Pattern).To(Equal("^[a-z]+$"))
	})
})

var _ = Describe("FieldRules", func() {
	It("should merge the field's rules with its type chain's rules", func() {
		field := resolution.Field{
			Type: resolution.TypeRef{Name: "Hz"},
			Domains: map[string]resolution.Domain{
				"validate": rule("max", intValue(50)),
			},
		}
		rules := validation.FieldRules(field, newChainTable())
		Expect(rules.Min.Int).To(Equal(int64(0)))
		Expect(rules.Max.Int).To(Equal(int64(50)))
	})

	It("should skip only when the field skips", func() {
		field := resolution.Field{
			Type:    resolution.TypeRef{Name: "Hz"},
			Domains: map[string]resolution.Domain{"validate": rule("skip")},
		}
		Expect(validation.FieldRules(field, newChainTable()).Skip).To(BeTrue())
	})
})
