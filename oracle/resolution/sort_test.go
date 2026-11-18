// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/x/testutil"
)

// sorted returns a field of type ref marked `@index sorted`, naming each of subs.
func sorted(name string, ref resolution.TypeRef, subs ...string) resolution.Field {
	expr := resolution.Expression{Name: "sorted"}
	for _, sub := range subs {
		expr.Values = append(expr.Values, resolution.ExpressionValue{
			Kind:       resolution.ValueKindIdent,
			IdentValue: sub,
		})
	}
	return resolution.Field{
		Name: name,
		Type: ref,
		Domains: map[string]resolution.Domain{
			"index": {Name: "index", Expressions: resolution.Expressions{expr}},
		},
	}
}

var _ = Describe("Sorts", func() {
	var table *resolution.Table
	BeforeEach(func() {
		table = resolution.NewTable()
		Expect(table.Add(resolution.Type{
			Name:          "Span",
			QualifiedName: "test.Span",
			Namespace:     "test",
			Form: resolution.StructForm{Fields: []resolution.Field{
				{Name: "start", Type: resolution.TypeRef{Name: "int64"}},
				{Name: "end", Type: resolution.TypeRef{Name: "int64"}},
				{Name: "label", Type: resolution.TypeRef{Name: "string"}},
			}},
		})).To(Succeed())
	})

	It("Should return a sort for each marked field, in field order", func() {
		sorts := MustSucceed(resolution.Sorts([]resolution.Field{
			sorted("name", resolution.TypeRef{Name: "string"}),
			{Name: "count", Type: resolution.TypeRef{Name: "int32"}},
			sorted("span", resolution.TypeRef{Name: "test.Span"}, "start"),
		}, table))
		Expect(sorts).To(HaveLen(2))
		Expect(sorts[0].Name).To(Equal("name"))
		Expect(sorts[0].Textual).To(BeTrue())
		Expect(sorts[0].Leaf().Name).To(Equal("name"))
		Expect(sorts[1].Name).To(Equal("span_start"))
		Expect(sorts[1].Textual).To(BeFalse())
		Expect(sorts[1].Type).To(Equal(resolution.TypeRef{Name: "int64"}))
		Expect(sorts[1].Leaf().Name).To(Equal("start"))
		Expect(sorts[1].Path).To(HaveLen(2))
	})

	It("Should return a sort for each sub-field that a mark names", func() {
		sorts := MustSucceed(resolution.Sorts([]resolution.Field{
			sorted("span", resolution.TypeRef{Name: "test.Span"}, "start", "label"),
		}, table))
		Expect(sorts).To(HaveLen(2))
		Expect(sorts[0].Name).To(Equal("span_start"))
		Expect(sorts[0].Textual).To(BeFalse())
		Expect(sorts[1].Name).To(Equal("span_label"))
		Expect(sorts[1].Textual).To(BeTrue())
		Expect(sorts[1].Leaf().Name).To(Equal("label"))
	})

	It("Should return no sorts for unmarked fields", func() {
		Expect(resolution.Sorts([]resolution.Field{
			{Name: "count", Type: resolution.TypeRef{Name: "int32"}},
		}, table)).To(BeEmpty())
	})

	DescribeTable("Should reject a mark",
		func(field resolution.Field, message string) {
			Expect(resolution.Sorts([]resolution.Field{field}, table)).
				Error().To(MatchError(message))
		},
		Entry(
			"on an optional field",
			func() resolution.Field {
				f := sorted("name", resolution.TypeRef{Name: "string"})
				f.Optional = true
				return f
			}(),
			"name: an optional field cannot be sorted",
		),
		Entry(
			"on a value that is not a string or a number",
			sorted("flag", resolution.TypeRef{Name: "bool"}),
			"flag: only a string or a number can be sorted",
		),
		Entry(
			"that names a sub-field of a value that is not a struct",
			sorted("name", resolution.TypeRef{Name: "string"}, "start"),
			"name: @index sorted start needs a struct",
		),
		Entry(
			"that names no sub-field",
			sorted("span", resolution.TypeRef{Name: "test.Span"}, "start", "stop"),
			"span: Span has no field stop",
		),
		Entry(
			"that shares its field with a lookup mark",
			func() resolution.Field {
				f := sorted("name", resolution.TypeRef{Name: "string"})
				dom := f.Domains["index"]
				dom.Expressions = append(dom.Expressions, resolution.Expression{
					Name: "lookup",
				})
				f.Domains["index"] = dom
				return f
			}(),
			"name: a sorted index answers exact matches, so drop @index lookup",
		),
	)
})
