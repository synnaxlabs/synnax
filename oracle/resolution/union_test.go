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
