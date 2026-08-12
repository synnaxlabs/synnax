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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ResolveUnionVariant", func() {
	var (
		table    *resolution.Table
		unionRef = resolution.TypeRef{Name: "test.CJC"}
	)

	BeforeEach(func() {
		table = resolution.NewTable()
		Expect(table.Add(resolution.Type{
			Name:          "CJC",
			Namespace:     "test",
			QualifiedName: "test.CJC",
			Form: resolution.UnionForm{
				Discriminator: "source",
				Variants: []resolution.UnionVariant{
					{
						Name: "built_in",
						Type: resolution.TypeRef{Name: "test.CJCBuiltIn"},
					},
					{
						Name: "const_val",
						Type: resolution.TypeRef{Name: "test.CJCConstVal"},
					},
				},
			},
		})).To(Succeed())
		Expect(table.Add(resolution.Type{
			Name:          "Mode",
			Namespace:     "test",
			QualifiedName: "test.Mode",
			Form: resolution.EnumForm{
				Values: []resolution.EnumValue{{Name: "fast", Value: "Fast"}},
			},
		})).To(Succeed())
	})

	DescribeTable(
		"Should resolve an identifier naming one of the union's variants",
		func(ident, expected string) {
			uv := MustBeOk(validation.ResolveUnionVariant(ident, unionRef, table))
			Expect(uv.Variant.Name).To(Equal(expected))
			Expect(uv.Union.Discriminator).To(Equal("source"))
			Expect(uv.Type.QualifiedName).To(Equal("test.CJC"))
		},
		Entry("bare variant name", "built_in", "built_in"),
		Entry("namespace qualified", "test.built_in", "built_in"),
		Entry("union-name prefixed", "CJCBuiltIn", "built_in"),
		Entry("union-name prefixed, multi-word", "CJCConstVal", "const_val"),
	)

	DescribeTable(
		"Should not resolve an identifier that names no variant",
		func(ident string, ref resolution.TypeRef) {
			_, ok := validation.ResolveUnionVariant(ident, ref, table)
			Expect(ok).To(BeFalse())
		},
		Entry("unknown variant", "chan", unionRef),
		Entry("prefix without a variant", "CJCMissing", unionRef),
		Entry("enum type, not a union", "fast", resolution.TypeRef{Name: "test.Mode"}),
		Entry("unresolvable type", "built_in", resolution.TypeRef{Name: "test.Absent"}),
		Entry("primitive type", "built_in", resolution.TypeRef{Name: "string"}),
	)
})
