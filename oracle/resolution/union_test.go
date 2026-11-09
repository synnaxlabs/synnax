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

var _ = Describe("UnionForm", func() {
	form := resolution.UnionForm{
		Discriminator: "type",
		Variants: []resolution.UnionVariant{
			{Name: "linear", Type: resolution.TypeRef{Name: "LinearScaleFields"}},
			{Name: "none", Type: resolution.TypeRef{Name: "NoneScaleFields"}},
		},
	}

	Describe("BorrowsEnum", func() {
		It("Should be false when no enum is named", func() {
			Expect(form.BorrowsEnum()).To(BeFalse())
		})
		It("Should be true when the discriminator names an enum", func() {
			borrowed := form
			borrowed.Enum = resolution.TypeRef{Name: "test.Method"}
			Expect(borrowed.BorrowsEnum()).To(BeTrue())
		})
	})

	Describe("Variant", func() {
		It("Should return the variant matching the discriminator value", func() {
			v := MustBeOk(form.Variant("linear"))
			Expect(v.Type.Name).To(Equal("LinearScaleFields"))
		})

		It("Should report a missing discriminator value", func() {
			_, ok := form.Variant("table")
			Expect(ok).To(BeFalse())
		})
	})
})

var _ = Describe("Table.UnionTypes", func() {
	It("Should return only union-form types", func() {
		table := resolution.NewTable()
		Expect(table.Add(resolution.Type{
			Name:          "Payload",
			QualifiedName: "test.Payload",
			Namespace:     "test",
			Form:          resolution.StructForm{},
		})).To(Succeed())
		Expect(table.Add(resolution.Type{
			Name:          "Status",
			QualifiedName: "test.Status",
			Namespace:     "test",
			Form:          resolution.EnumForm{},
		})).To(Succeed())
		Expect(table.Add(resolution.Type{
			Name:          "Scale",
			QualifiedName: "test.Scale",
			Namespace:     "test",
			Form:          resolution.UnionForm{Discriminator: "type"},
		})).To(Succeed())

		unions := table.UnionTypes()
		Expect(unions).To(HaveLen(1))
		Expect(unions[0].Name).To(Equal("Scale"))
	})
})

var _ = Describe("RecordFields", func() {
	var table *resolution.Table
	BeforeEach(func() {
		table = resolution.NewTable()
		Expect(table.Add(resolution.Type{
			Name:          "Base",
			QualifiedName: "test.Base",
			Namespace:     "test",
			Form: resolution.StructForm{Fields: []resolution.Field{
				{Name: "name", Type: resolution.TypeRef{Name: "string"}},
			}},
		})).To(Succeed())
	})
	union := func(enum string) resolution.Type {
		return resolution.Type{
			Name: "View",
			Form: resolution.UnionForm{
				Discriminator: "type",
				Enum:          resolution.TypeRef{Name: enum},
				Extends:       []resolution.TypeRef{{Name: "test.Base"}},
			},
		}
	}

	It("Should hold the tag of a union that borrows its enum", func() {
		fields := MustBeOk(resolution.RecordFields(union("test.Type"), table))
		Expect(fields).To(Equal([]resolution.Field{
			{Name: "name", Type: resolution.TypeRef{Name: "string"}},
			{Name: "type", Type: resolution.TypeRef{Name: "test.Type"}},
		}))
	})

	It("Should hold the fields of a struct with defaults for its parameters", func() {
		def := resolution.TypeRef{Name: "int32"}
		thing := resolution.Type{
			Name: "Thing",
			Form: resolution.StructForm{
				TypeParams: []resolution.TypeParam{{Name: "T", Default: &def}},
				Fields: []resolution.Field{
					{Name: "key", Type: resolution.TypeRef{Name: "uuid"}},
					{Name: "value", Type: resolution.TypeRef{
						TypeParam: &resolution.TypeParam{Name: "T"},
					}},
				},
			},
		}
		fields := MustBeOk(resolution.RecordFields(thing, table))
		Expect(fields).To(Equal([]resolution.Field{
			{Name: "key", Type: resolution.TypeRef{Name: "uuid"}},
			{Name: "value", Type: def},
		}))
	})

	It("Should report a type that is neither a struct nor a union", func() {
		_, ok := resolution.RecordFields(
			resolution.Type{Name: "Kind", Form: resolution.EnumForm{}},
			table,
		)
		Expect(ok).To(BeFalse())
	})

	It("Should leave out the tag of an unnamed enum", func() {
		fields := MustBeOk(resolution.RecordFields(union(""), table))
		Expect(fields).To(Equal([]resolution.Field{
			{Name: "name", Type: resolution.TypeRef{Name: "string"}},
		}))
	})
})
