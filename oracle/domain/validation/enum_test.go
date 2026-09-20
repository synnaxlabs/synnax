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

var _ = Describe("ResolveEnumVariant", func() {
	var (
		table   *resolution.Table
		enumRef = resolution.TypeRef{Name: "test.Concurrency"}
	)

	BeforeEach(func() {
		table = resolution.NewTable()
		Expect(table.Add(resolution.Type{
			Name:          "Concurrency",
			Namespace:     "test",
			QualifiedName: "test.Concurrency",
			Form: resolution.EnumForm{
				Values: []resolution.EnumValue{
					{Name: "exclusive", Value: "Exclusive"},
					{Name: "cfg_default", Value: "CfgDefault"},
				},
			},
		})).To(Succeed())
		Expect(table.Add(resolution.Type{
			Name:          "Point",
			Namespace:     "test",
			QualifiedName: "test.Point",
			Form:          resolution.StructForm{},
		})).To(Succeed())
	})

	DescribeTable("Should resolve an identifier naming one of the enum's variants",
		func(ident, expected string) {
			ev := MustBeOk(validation.ResolveEnumVariant(ident, enumRef, table))
			Expect(ev.Variant.Name).To(Equal(expected))
			Expect(ev.Type.QualifiedName).To(Equal("test.Concurrency"))
		},
		Entry("bare variant name", "exclusive", "exclusive"),
		Entry("namespace qualified", "test.exclusive", "exclusive"),
		Entry("multi-word snake_case variant", "cfg_default", "cfg_default"),
		Entry("EnumName-prefixed form", "ConcurrencyExclusive", "exclusive"),
		Entry(
			"namespace-qualified prefixed form",
			"test.ConcurrencyExclusive",
			"exclusive",
		),
	)

	DescribeTable("Should not resolve an identifier that names no variant",
		func(ident string, ref resolution.TypeRef) {
			_, ok := validation.ResolveEnumVariant(ident, ref, table)
			Expect(ok).To(BeFalse())
		},
		Entry("unknown bare name", "shared", enumRef),
		Entry("prefix without a variant", "ConcurrencyMissing", enumRef),
		Entry("struct type, not an enum", "exclusive",
			resolution.TypeRef{Name: "test.Point"}),
		Entry("unresolvable type", "exclusive",
			resolution.TypeRef{Name: "test.Absent"}),
	)
})
