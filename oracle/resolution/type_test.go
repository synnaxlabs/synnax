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
)

var _ = Describe("CollectionKind", func() {
	var table *resolution.Table
	ref := func(name string) resolution.TypeRef { return resolution.TypeRef{Name: name} }
	param := func(tp *resolution.TypeParam) resolution.TypeRef {
		return resolution.TypeRef{Name: "T", TypeParam: tp}
	}

	BeforeEach(func() {
		table = resolution.NewTable()
		add := func(name string, form resolution.TypeForm) {
			Expect(table.Add(resolution.Type{
				Name:          name,
				QualifiedName: "t." + name,
				Namespace:     "t",
				Form:          form,
			})).To(Succeed())
		}
		add("DistinctArray", resolution.DistinctForm{Base: ref("Array")})
		add("AliasMap", resolution.AliasForm{Target: ref("Map")})
		add("AliasArray", resolution.AliasForm{Target: ref("Array")})
		add("AliasToAlias", resolution.AliasForm{Target: ref("t.AliasArray")})
		add("DistinctOverAlias", resolution.DistinctForm{Base: ref("t.AliasArray")})
		add("DistinctScalar", resolution.DistinctForm{Base: ref("uuid")})
		add("User", resolution.StructForm{})
		add("Color", resolution.EnumForm{Values: []resolution.EnumValue{}})
		add("CycleA", resolution.AliasForm{Target: ref("t.CycleB")})
		add("CycleB", resolution.AliasForm{Target: ref("t.CycleA")})
		add("SelfDistinct", resolution.DistinctForm{Base: ref("t.SelfDistinct")})
	})

	DescribeTable("reports whether ref resolves to a collection",
		func(makeRef func() resolution.TypeRef, wantKind string, wantOK bool) {
			kind, ok := resolution.CollectionKind(makeRef(), table)
			Expect(ok).To(Equal(wantOK))
			Expect(kind).To(Equal(wantKind))
		},
		Entry("direct Array", func() resolution.TypeRef { return ref("Array") }, "Array", true),
		Entry("direct Map", func() resolution.TypeRef { return ref("Map") }, "Map", true),
		Entry("direct record", func() resolution.TypeRef { return ref("record") }, "record", true),
		Entry("distinct over Array", func() resolution.TypeRef { return ref("t.DistinctArray") }, "Array", true),
		Entry("alias to Map", func() resolution.TypeRef { return ref("t.AliasMap") }, "Map", true),
		Entry("alias chain to Array", func() resolution.TypeRef { return ref("t.AliasToAlias") }, "Array", true),
		Entry("distinct over alias to Array", func() resolution.TypeRef { return ref("t.DistinctOverAlias") }, "Array", true),
		Entry("type param constrained to Array",
			func() resolution.TypeRef {
				c := ref("Array")
				return param(&resolution.TypeParam{Constraint: &c})
			}, "Array", true),
		Entry("type param defaulting to Map",
			func() resolution.TypeRef {
				d := ref("Map")
				return param(&resolution.TypeParam{Default: &d})
			}, "Map", true),
		Entry("type param prefers constraint over default",
			func() resolution.TypeRef {
				c, d := ref("Array"), ref("record")
				return param(&resolution.TypeParam{Constraint: &c, Default: &d})
			}, "Array", true),
		Entry("primitive", func() resolution.TypeRef { return ref("string") }, "", false),
		Entry("distinct over scalar", func() resolution.TypeRef { return ref("t.DistinctScalar") }, "", false),
		Entry("struct", func() resolution.TypeRef { return ref("t.User") }, "", false),
		Entry("enum", func() resolution.TypeRef { return ref("t.Color") }, "", false),
		Entry("bare type param", func() resolution.TypeRef { return param(&resolution.TypeParam{}) }, "", false),
		Entry("unresolved name", func() resolution.TypeRef { return ref("t.Missing") }, "", false),
		Entry("cyclic alias chain", func() resolution.TypeRef { return ref("t.CycleA") }, "", false),
		Entry("self-referential distinct", func() resolution.TypeRef { return ref("t.SelfDistinct") }, "", false),
	)
})
