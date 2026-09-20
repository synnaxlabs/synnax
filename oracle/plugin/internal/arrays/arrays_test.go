// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arrays_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arrays", func() {
	var table *resolution.Table

	arrayOf := func(elem string, size *int64) resolution.TypeRef {
		return resolution.TypeRef{
			Name:      "Array",
			TypeArgs:  []resolution.TypeRef{{Name: elem}},
			ArraySize: size,
		}
	}

	BeforeEach(func() {
		table = resolution.NewTable()
		for _, typ := range []resolution.Type{
			{
				Name:          "Label",
				QualifiedName: "test.Label",
				Form:          resolution.StructForm{},
			},
			{
				Name:          "Names",
				QualifiedName: "test.Names",
				Form:          resolution.AliasForm{Target: arrayOf("string", nil)},
			},
			{
				Name:          "MAC",
				QualifiedName: "test.MAC",
				Form: resolution.AliasForm{
					Target: arrayOf("uint8", new(int64(6))),
				},
			},
			{
				Name:          "MACAddr",
				QualifiedName: "test.MACAddr",
				Form: resolution.DistinctForm{
					Base: resolution.TypeRef{Name: "test.MAC"},
				},
			},
			{
				Name:          "Tags",
				QualifiedName: "test.Tags",
				Form: resolution.DistinctForm{
					Base: resolution.TypeRef{Name: "test.Names"},
				},
			},
			{
				Name:          "Matrix",
				QualifiedName: "test.Matrix",
				Form: resolution.AliasForm{
					Target: resolution.TypeRef{
						Name:     "Array",
						TypeArgs: []resolution.TypeRef{arrayOf("float64", nil)},
					},
				},
			},
			{
				Name:          "Arr",
				QualifiedName: "test.Arr",
				Form:          resolution.BuiltinGenericForm{Name: "Array", Arity: 1},
			},
			{
				Name:          "NoElem",
				QualifiedName: "test.NoElem",
				Form: resolution.AliasForm{
					Target: resolution.TypeRef{
						Name:      "Array",
						ArraySize: new(int64(4)),
					},
				},
			},
		} {
			Expect(table.Add(typ)).To(Succeed())
		}
	})

	Describe("IsArray", func() {
		DescribeTable("should detect arrays through aliases and distincts",
			func(ref resolution.TypeRef, expected bool) {
				Expect(arrays.IsArray(ref, table)).To(Equal(expected))
			},
			Entry("direct Array reference", arrayOf("string", nil), true),
			Entry("alias to an array", resolution.TypeRef{Name: "test.Names"}, true),
			Entry(
				"distinct over an array alias",
				resolution.TypeRef{Name: "test.Tags"},
				true,
			),
			Entry(
				"named builtin generic array",
				resolution.TypeRef{Name: "test.Arr"},
				true,
			),
			Entry("Map builtin generic", resolution.TypeRef{Name: "Map"}, false),
			Entry("struct", resolution.TypeRef{Name: "test.Label"}, false),
			Entry("primitive", resolution.TypeRef{Name: "string"}, false),
			Entry("unresolvable", resolution.TypeRef{Name: "nonexistent"}, false),
		)
	})

	Describe("ElementType", func() {
		DescribeTable("should return the element for array references",
			func(ref resolution.TypeRef, expected string) {
				elem := MustBeOk(arrays.ElementType(ref, table))
				Expect(elem.Name).To(Equal(expected))
			},
			Entry("direct Array reference", arrayOf("string", nil), "string"),
			Entry(
				"distinct over an array alias",
				resolution.TypeRef{Name: "test.Tags"},
				"string",
			),
			Entry(
				"named builtin generic with type args",
				resolution.TypeRef{
					Name:     "test.Arr",
					TypeArgs: []resolution.TypeRef{{Name: "int32"}},
				},
				"int32",
			),
		)

		DescribeTable("should report false for non-arrays and missing elements",
			func(ref resolution.TypeRef) {
				_, ok := arrays.ElementType(ref, table)
				Expect(ok).To(BeFalse())
			},
			Entry("bare Array without type args", resolution.TypeRef{Name: "Array"}),
			Entry(
				"named builtin generic without type args",
				resolution.TypeRef{Name: "test.Arr"},
			),
			Entry("Map builtin generic", resolution.TypeRef{
				Name:     "Map",
				TypeArgs: []resolution.TypeRef{{Name: "string"}, {Name: "int32"}},
			}),
			Entry("struct", resolution.TypeRef{Name: "test.Label"}),
			Entry("unresolvable", resolution.TypeRef{Name: "nonexistent"}),
		)
	})

	Describe("Size", func() {
		DescribeTable("should return the declared fixed size",
			func(ref resolution.TypeRef, expected int64) {
				Expect(arrays.Size(ref, table)).To(HaveValue(Equal(expected)))
			},
			Entry(
				"direct fixed-size reference",
				arrayOf("uint8", new(int64(6))),
				int64(6),
			),
			Entry(
				"alias to a fixed-size array",
				resolution.TypeRef{Name: "test.MAC"},
				int64(6),
			),
			Entry(
				"distinct over a fixed-size alias",
				resolution.TypeRef{Name: "test.MACAddr"},
				int64(6),
			),
		)

		DescribeTable("should return nil for dynamic and non-array references",
			func(ref resolution.TypeRef) {
				Expect(arrays.Size(ref, table)).To(BeNil())
			},
			Entry("dynamic array", arrayOf("string", nil)),
			Entry("dynamic array alias", resolution.TypeRef{Name: "test.Names"}),
			Entry("bare Array without a size", resolution.TypeRef{Name: "Array"}),
			Entry("struct", resolution.TypeRef{Name: "test.Label"}),
			Entry("unresolvable", resolution.TypeRef{Name: "nonexistent"}),
		)
	})

	Describe("IsFixedSizeUint8", func() {
		DescribeTable("should detect fixed-size uint8 arrays",
			func(ref resolution.TypeRef, expected bool) {
				Expect(arrays.IsFixedSizeUint8(ref, table)).To(Equal(expected))
			},
			Entry(
				"direct fixed uint8 array",
				arrayOf("uint8", new(int64(6))),
				true,
			),
			Entry(
				"alias to a fixed uint8 array",
				resolution.TypeRef{Name: "test.MAC"},
				true,
			),
			Entry(
				"fixed string array",
				arrayOf("string", new(int64(3))),
				false,
			),
			Entry(
				"fixed struct array",
				arrayOf("test.Label", new(int64(3))),
				false,
			),
			Entry("dynamic uint8 array", arrayOf("uint8", nil), false),
			Entry(
				"fixed array without an element",
				resolution.TypeRef{Name: "test.NoElem"},
				false,
			),
		)

		DescribeTable(
			"should fall back to the element name when it does not resolve",
			func(elem string, expected bool) {
				bare := &resolution.Table{}
				ref := arrayOf(elem, new(int64(4)))
				Expect(arrays.IsFixedSizeUint8(ref, bare)).To(Equal(expected))
			},
			Entry("unresolved uint8 element", "uint8", true),
			Entry("unresolved non-uint8 element", "int8", false),
		)
	})

	Describe("IsNested", func() {
		DescribeTable("should detect arrays of arrays",
			func(ref resolution.TypeRef, expected bool) {
				Expect(arrays.IsNested(ref, table)).To(Equal(expected))
			},
			Entry("alias to a nested array", resolution.TypeRef{Name: "test.Matrix"},
				true),
			Entry(
				"direct nested reference",
				resolution.TypeRef{
					Name:     "Array",
					TypeArgs: []resolution.TypeRef{{Name: "test.Names"}},
				},
				true,
			),
			Entry("flat array", arrayOf("string", nil), false),
			Entry("non-array", resolution.TypeRef{Name: "test.Label"}, false),
			Entry(
				"array without an element",
				resolution.TypeRef{Name: "test.NoElem"},
				false,
			),
		)
	})

	Describe("OptionalWrapperName", func() {
		DescribeTable("should derive the optional list wrapper name",
			func(ref resolution.TypeRef, expected string) {
				Expect(arrays.OptionalWrapperName(ref, table)).To(Equal(expected))
			},
			Entry(
				"array of a named type",
				arrayOf("test.Label", nil),
				"LabelList",
			),
			Entry(
				"non-array reference",
				resolution.TypeRef{Name: "test.Label"},
				"ListWrapper",
			),
		)

		DescribeTable("should fall back when the element does not resolve",
			func(elem, expected string) {
				bare := &resolution.Table{}
				name := arrays.OptionalWrapperName(arrayOf(elem, nil), bare)
				Expect(name).To(Equal(expected))
			},
			Entry("primitive element title-cases", "uint8", "Uint8List"),
			Entry("unknown element", "Mystery", "ListWrapper"),
		)
	})

	Describe("NestedWrapperName", func() {
		DescribeTable("should derive the nested array wrapper name",
			func(ref resolution.TypeRef, expected string) {
				Expect(arrays.NestedWrapperName(ref, table)).To(Equal(expected))
			},
			Entry(
				"array of a named type",
				arrayOf("test.Label", nil),
				"LabelWrapper",
			),
			Entry(
				"non-array reference",
				resolution.TypeRef{Name: "test.Label"},
				"ArrayWrapper",
			),
		)

		DescribeTable("should fall back when the element does not resolve",
			func(elem, expected string) {
				bare := &resolution.Table{}
				name := arrays.NestedWrapperName(arrayOf(elem, nil), bare)
				Expect(name).To(Equal(expected))
			},
			Entry("primitive element title-cases", "float64", "Float64Array"),
			Entry("unknown element", "Mystery", "ArrayWrapper"),
		)
	})
})
