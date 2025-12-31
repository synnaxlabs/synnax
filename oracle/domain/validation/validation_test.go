// Copyright 2025 Synnax Labs, Inc.
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
	It("should return nil for nil domain", func() {
		Expect(validation.Parse(nil)).To(BeNil())
	})

	It("should parse required expression", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{Name: "required"}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Required).To(BeTrue())
	})

	It("should parse email expression", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{Name: "email"}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Email).To(BeTrue())
	})

	It("should parse url expression", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{Name: "url"}},
		}
		rules := validation.Parse(domain)
		Expect(rules.URL).To(BeTrue())
	})

	It("should parse min_length expression", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{
				Name:   "min_length",
				Values: []resolution.ExpressionValue{{IntValue: 5}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.MinLength).NotTo(BeNil())
		Expect(*rules.MinLength).To(Equal(int64(5)))
	})

	It("should parse max_length expression", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{
				Name:   "max_length",
				Values: []resolution.ExpressionValue{{IntValue: 100}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.MaxLength).NotTo(BeNil())
		Expect(*rules.MaxLength).To(Equal(int64(100)))
	})

	It("should parse pattern expression", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{
				Name:   "pattern",
				Values: []resolution.ExpressionValue{{StringValue: "^[a-z]+$"}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Pattern).NotTo(BeNil())
		Expect(*rules.Pattern).To(Equal("^[a-z]+$"))
	})

	It("should parse min expression with int", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{
				Name:   "min",
				Values: []resolution.ExpressionValue{{Kind: resolution.ValueKindInt, IntValue: 10}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Min).NotTo(BeNil())
		Expect(rules.Min.IsInt).To(BeTrue())
		Expect(rules.Min.Int).To(Equal(int64(10)))
	})

	It("should parse min expression with float", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{
				Name:   "min",
				Values: []resolution.ExpressionValue{{Kind: resolution.ValueKindFloat, FloatValue: 1.5}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Min).NotTo(BeNil())
		Expect(rules.Min.IsInt).To(BeFalse())
		Expect(rules.Min.Float).To(Equal(1.5))
	})

	It("should parse max expression", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{
				Name:   "max",
				Values: []resolution.ExpressionValue{{Kind: resolution.ValueKindInt, IntValue: 100}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Max).NotTo(BeNil())
		Expect(rules.Max.Int).To(Equal(int64(100)))
	})

	It("should parse default expression", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{{
				Name:   "default",
				Values: []resolution.ExpressionValue{{Kind: resolution.ValueKindString, StringValue: "test"}},
			}},
		}
		rules := validation.Parse(domain)
		Expect(rules.Default).NotTo(BeNil())
		Expect(rules.Default.StringValue).To(Equal("test"))
	})

	It("should parse multiple expressions", func() {
		domain := &resolution.Domain{
			Expressions: []*resolution.Expression{
				{Name: "required"},
				{Name: "min_length", Values: []resolution.ExpressionValue{{IntValue: 1}}},
				{Name: "max_length", Values: []resolution.ExpressionValue{{IntValue: 50}}},
			},
		}
		rules := validation.Parse(domain)
		Expect(rules.Required).To(BeTrue())
		Expect(*rules.MinLength).To(Equal(int64(1)))
		Expect(*rules.MaxLength).To(Equal(int64(50)))
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
})
