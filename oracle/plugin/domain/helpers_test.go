// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
)

func makeDomains(entries ...struct {
	domain string
	exprs  []resolution.Expression
}) map[string]resolution.Domain {
	result := make(map[string]resolution.Domain)
	for _, e := range entries {
		result[e.domain] = resolution.Domain{
			Name:        e.domain,
			Expressions: e.exprs,
		}
	}
	return result
}

func makeType(name string, domains map[string]resolution.Domain) resolution.Type {
	return resolution.Type{
		Name:    name,
		Domains: domains,
		Form:    resolution.StructForm{},
	}
}

func makeField(name string, domains map[string]resolution.Domain) resolution.Field {
	return resolution.Field{
		Name:    name,
		Domains: domains,
	}
}

var _ = Describe("GetString", func() {
	It("should return the string value from a domain expression", func() {
		t := makeType("User", makeDomains(struct {
			domain string
			exprs  []resolution.Expression
		}{
			domain: "go",
			exprs: []resolution.Expression{
				{Name: "output", Values: []resolution.ExpressionValue{
					{StringValue: "core/pkg/user"},
				}},
			},
		}))
		Expect(domain.GetStringFromType(t, "go", "output")).To(Equal("core/pkg/user"))
	})

	It("should return ident value when string value is empty", func() {
		t := makeType("User", makeDomains(struct {
			domain string
			exprs  []resolution.Expression
		}{
			domain: "go",
			exprs: []resolution.Expression{
				{Name: "type", Values: []resolution.ExpressionValue{
					{IdentValue: "MyCustomType"},
				}},
			},
		}))
		Expect(domain.GetStringFromType(t, "go", "type")).To(Equal("MyCustomType"))
	})

	It("should return empty string for missing domain", func() {
		t := makeType("User", map[string]resolution.Domain{})
		Expect(domain.GetStringFromType(t, "go", "output")).To(BeEmpty())
	})

	It("should return empty string for missing expression", func() {
		t := makeType("User", makeDomains(struct {
			domain string
			exprs  []resolution.Expression
		}{
			domain: "go",
			exprs:  []resolution.Expression{{Name: "other"}},
		}))
		Expect(domain.GetStringFromType(t, "go", "output")).To(BeEmpty())
	})

	It("should return empty string when expression has no values", func() {
		t := makeType("User", makeDomains(struct {
			domain string
			exprs  []resolution.Expression
		}{
			domain: "go",
			exprs:  []resolution.Expression{{Name: "output", Values: nil}},
		}))
		Expect(domain.GetStringFromType(t, "go", "output")).To(BeEmpty())
	})
})

var _ = Describe("GetStringFromField", func() {
	It("should return the string value from a field's domain expression", func() {
		f := makeField("name", makeDomains(struct {
			domain string
			exprs  []resolution.Expression
		}{
			domain: "validate",
			exprs: []resolution.Expression{
				{Name: "pattern", Values: []resolution.ExpressionValue{
					{StringValue: "^[a-z]+$"},
				}},
			},
		}))
		Expect(domain.GetStringFromField(f, "validate", "pattern")).To(Equal("^[a-z]+$"))
	})
})

var _ = Describe("GetAllStrings", func() {
	It("should collect values from all expressions with the same name", func() {
		t := makeType("User", map[string]resolution.Domain{
			"go": {
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "field", Values: []resolution.ExpressionValue{
						{StringValue: "Name"},
					}},
					{Name: "field", Values: []resolution.ExpressionValue{
						{StringValue: "Age"},
					}},
				},
			},
		})
		Expect(domain.GetAllStringsFromType(t, "go", "field")).To(Equal([]string{
			"Name", "Age",
		}))
	})

	It("should return nil for missing domain", func() {
		t := makeType("User", map[string]resolution.Domain{})
		Expect(domain.GetAllStringsFromType(t, "go", "field")).To(BeNil())
	})

	It("should skip expressions with different names", func() {
		t := makeType("User", map[string]resolution.Domain{
			"go": {
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "output", Values: []resolution.ExpressionValue{
						{StringValue: "core/pkg/user"},
					}},
					{Name: "field", Values: []resolution.ExpressionValue{
						{StringValue: "Name"},
					}},
				},
			},
		})
		Expect(domain.GetAllStringsFromType(t, "go", "field")).To(Equal([]string{"Name"}))
	})

	It("should collect ident values when string values are empty", func() {
		t := makeType("User", map[string]resolution.Domain{
			"go": {
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "field", Values: []resolution.ExpressionValue{
						{IdentValue: "SomeIdent"},
					}},
				},
			},
		})
		Expect(domain.GetAllStringsFromType(t, "go", "field")).To(Equal([]string{"SomeIdent"}))
	})
})

var _ = Describe("HasExpr", func() {
	It("should return true when expression exists", func() {
		t := makeType("User", map[string]resolution.Domain{
			"validate": {
				Name: "validate",
				Expressions: resolution.Expressions{
					{Name: "required"},
				},
			},
		})
		Expect(domain.HasExprFromType(t, "validate", "required")).To(BeTrue())
	})

	It("should return false when domain is missing", func() {
		t := makeType("User", map[string]resolution.Domain{})
		Expect(domain.HasExprFromType(t, "validate", "required")).To(BeFalse())
	})

	It("should return false when expression is missing", func() {
		t := makeType("User", map[string]resolution.Domain{
			"validate": {
				Name:        "validate",
				Expressions: resolution.Expressions{{Name: "min"}},
			},
		})
		Expect(domain.HasExprFromType(t, "validate", "required")).To(BeFalse())
	})
})

var _ = Describe("GetName", func() {
	It("should return domain name override when present", func() {
		t := makeType("User", map[string]resolution.Domain{
			"go": {
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "name", Values: []resolution.ExpressionValue{
						{StringValue: "UserModel"},
					}},
				},
			},
		})
		Expect(domain.GetName(t, "go")).To(Equal("UserModel"))
	})

	It("should fall back to type name when no override", func() {
		t := makeType("User", map[string]resolution.Domain{})
		Expect(domain.GetName(t, "go")).To(Equal("User"))
	})
})

var _ = Describe("GetType", func() {
	It("should return type override from domain", func() {
		t := makeType("User", map[string]resolution.Domain{
			"go": {
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "type", Values: []resolution.ExpressionValue{
						{IdentValue: "CustomType"},
					}},
				},
			},
		})
		Expect(domain.GetType(t, "go")).To(Equal("CustomType"))
	})
})

var _ = Describe("GetFieldName", func() {
	It("should return domain name override when present", func() {
		f := makeField("user_name", map[string]resolution.Domain{
			"go": {
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "name", Values: []resolution.ExpressionValue{
						{StringValue: "UserName"},
					}},
				},
			},
		})
		Expect(domain.GetFieldName(f, "go")).To(Equal("UserName"))
	})

	It("should fall back to field name when no override", func() {
		f := makeField("user_name", map[string]resolution.Domain{})
		Expect(domain.GetFieldName(f, "go")).To(Equal("user_name"))
	})
})

var _ = Describe("GetFieldType", func() {
	It("should return field type override from domain", func() {
		f := makeField("data", map[string]resolution.Domain{
			"go": {
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "type", Values: []resolution.ExpressionValue{
						{IdentValue: "json.RawMessage"},
					}},
				},
			},
		})
		Expect(domain.GetFieldType(f, "go")).To(Equal("json.RawMessage"))
	})
})
