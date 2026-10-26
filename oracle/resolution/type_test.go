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

var _ = Describe("RetrieveKey", func() {
	withKey := func(name string) resolution.Type {
		return resolution.Type{
			Namespace: "rack",
			Domains: map[string]resolution.Domain{"retrieve": {
				Name: "retrieve",
				Expressions: resolution.Expressions{{
					Name: "key",
					Values: []resolution.ExpressionValue{{
						Kind:        resolution.ValueKindString,
						StringValue: name,
					}},
				}},
			}},
		}
	}

	It("Should return false for a type with no @retrieve key", func() {
		_, ok := resolution.RetrieveKey(resolution.Type{Namespace: "rack"})
		Expect(ok).To(BeFalse())
	})

	It("Should qualify an unqualified name with the type's namespace", func() {
		Expect(MustBeOk(resolution.RetrieveKey(withKey("Key")))).
			To(Equal(resolution.TypeRef{Name: "rack.Key"}))
	})

	It("Should keep a qualified name as written", func() {
		Expect(MustBeOk(resolution.RetrieveKey(withKey("channel.Key")))).
			To(Equal(resolution.TypeRef{Name: "channel.Key"}))
	})
})

var _ = Describe("InFilterRequest", func() {
	var table *resolution.Table
	BeforeEach(func() {
		table = resolution.NewTable()
		add := func(namespace, name string, filter *resolution.Filter) {
			GinkgoHelper()
			Expect(table.Add(resolution.Type{
				Name:          name,
				QualifiedName: namespace + "." + name,
				Namespace:     namespace,
				Form:          resolution.StructForm{},
				Filter:        filter,
			})).To(Succeed())
		}
		add("t", "Thing", nil)
		add("t", "Where", &resolution.Filter{
			Target:   resolution.TypeRef{Name: "t.Thing"},
			Request:  "t.RetrieveRequest",
			Response: "t.RetrieveResponse",
		})
		add("t", "RetrieveRequest", nil)
		add("t", "RetrieveResponse", nil)
		add(resolution.FilterNamespace, "Set", nil)
	})

	DescribeTable("Should report whether a type is part of a filter request",
		func(name string, in bool) {
			typ := MustBeOk(table.Get(name))
			Expect(resolution.InFilterRequest(typ, table)).To(Equal(in))
		},
		Entry("a filter", "t.Where", true),
		Entry("the request derived from a filter", "t.RetrieveRequest", true),
		Entry("an operator type", resolution.OperatorSet, true),
		Entry("the response derived from a filter", "t.RetrieveResponse", false),
		Entry("a plain struct", "t.Thing", false),
	)
})

var _ = Describe("OperatorMembers", func() {
	DescribeTable("Should return the members of an operator type",
		func(op string, members []string) {
			Expect(MustBeOk(resolution.OperatorMembers(op))).To(Equal(members))
		},
		Entry("a set", resolution.OperatorSet, []string{"eq", "in"}),
		Entry("a string", resolution.OperatorString, []string{"eq", "in", "regex"}),
	)

	It("Should return false for a type that is not an operator", func() {
		_, ok := resolution.OperatorMembers("filter.Thing")
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("CollectionKind", func() {
	var table *resolution.Table
	ref := func(name string) resolution.TypeRef { return resolution.TypeRef{Name: name} }
	param := func(tp *resolution.TypeParam) resolution.TypeRef {
		return resolution.TypeRef{Name: "T", TypeParam: tp}
	}

	BeforeEach(func() {
		table = resolution.NewTable()
		add := func(name string, form resolution.TypeForm) {
			GinkgoHelper()
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

	DescribeTable(
		"reports whether ref resolves to a collection",
		func(makeRef func() resolution.TypeRef, wantKind string, wantOK bool) {
			kind, ok := resolution.CollectionKind(makeRef(), table)
			Expect(ok).To(Equal(wantOK))
			Expect(kind).To(Equal(wantKind))
		},
		Entry(
			"direct Array",
			func() resolution.TypeRef { return ref("Array") },
			"Array",
			true,
		),
		Entry(
			"direct Map",
			func() resolution.TypeRef { return ref("Map") },
			"Map",
			true,
		),
		Entry(
			"direct record",
			func() resolution.TypeRef { return ref("record") },
			"record",
			true,
		),
		Entry(
			"distinct over Array",
			func() resolution.TypeRef { return ref("t.DistinctArray") },
			"Array",
			true,
		),
		Entry(
			"alias to Map",
			func() resolution.TypeRef { return ref("t.AliasMap") },
			"Map",
			true,
		),
		Entry(
			"alias chain to Array",
			func() resolution.TypeRef { return ref("t.AliasToAlias") },
			"Array",
			true,
		),
		Entry(
			"distinct over alias to Array",
			func() resolution.TypeRef { return ref("t.DistinctOverAlias") },
			"Array",
			true,
		),
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
		Entry(
			"primitive",
			func() resolution.TypeRef { return ref("string") },
			"",
			false,
		),
		Entry(
			"distinct over scalar",
			func() resolution.TypeRef { return ref("t.DistinctScalar") },
			"",
			false,
		),
		Entry("struct", func() resolution.TypeRef { return ref("t.User") }, "", false),
		Entry("enum", func() resolution.TypeRef { return ref("t.Color") }, "", false),
		Entry(
			"bare type param",
			func() resolution.TypeRef { return param(&resolution.TypeParam{}) },
			"",
			false,
		),
		Entry(
			"unresolved name",
			func() resolution.TypeRef { return ref("t.Missing") },
			"",
			false,
		),
		Entry(
			"cyclic alias chain",
			func() resolution.TypeRef { return ref("t.CycleA") },
			"",
			false,
		),
		Entry(
			"self-referential distinct",
			func() resolution.TypeRef { return ref("t.SelfDistinct") },
			"",
			false,
		),
	)
})

var _ = Describe("Dealias", func() {
	var table *resolution.Table
	ref := func(name string, args ...resolution.TypeRef) resolution.TypeRef {
		return resolution.TypeRef{Name: name, TypeArgs: args}
	}
	tp := resolution.TypeParam{Name: "T"}

	BeforeEach(func() {
		table = resolution.NewTable()
		add := func(name string, form resolution.TypeForm) {
			GinkgoHelper()
			Expect(table.Add(resolution.Type{
				Name:          name,
				QualifiedName: "t." + name,
				Namespace:     "t",
				Form:          form,
			})).To(Succeed())
		}
		add("User", resolution.StructForm{})
		add("Box", resolution.StructForm{TypeParams: []resolution.TypeParam{tp}})
		add("AliasUser", resolution.AliasForm{Target: ref("t.User")})
		add("AliasToAlias", resolution.AliasForm{Target: ref("t.AliasUser")})
		add("AliasUUID", resolution.AliasForm{Target: ref("uuid")})
		add("BoxOfUUID", resolution.AliasForm{Target: ref("t.Box", ref("uuid"))})
		add("AliasBox", resolution.AliasForm{
			Target:     ref("t.Box", resolution.TypeRef{TypeParam: &tp}),
			TypeParams: []resolution.TypeParam{tp},
		})
		add("DistinctUser", resolution.DistinctForm{Base: ref("t.User")})
		add("AliasDistinct", resolution.AliasForm{Target: ref("t.DistinctUser")})
		add("AliasMissing", resolution.AliasForm{Target: ref("t.Missing")})
		add("CycleA", resolution.AliasForm{Target: ref("t.CycleB")})
		add("CycleB", resolution.AliasForm{Target: ref("t.CycleA")})
	})

	DescribeTable(
		"Should follow alias declarations to the type they stand for",
		func(in, wantRef resolution.TypeRef) {
			typ, out, ok := resolution.Dealias(in, table)
			Expect(ok).To(BeTrue())
			Expect(out).To(Equal(wantRef))
			Expect(typ.QualifiedName).To(Equal(wantRef.Name))
		},
		Entry("a struct", ref("t.User"), ref("t.User")),
		Entry("an alias of a struct", ref("t.AliasUser"), ref("t.User")),
		Entry("an alias chain", ref("t.AliasToAlias"), ref("t.User")),
		Entry("an alias of a primitive", ref("t.AliasUUID"), ref("uuid")),
		Entry(
			"an alias that fixes a type argument",
			ref("t.BoxOfUUID"),
			ref("t.Box", ref("uuid")),
		),
		Entry(
			"a generic alias, carrying its type argument to the target",
			ref("t.AliasBox", ref("string")),
			ref("t.Box", ref("string")),
		),
		Entry(
			"an alias of a distinct type, stopping at the distinct type",
			ref("t.AliasDistinct"),
			ref("t.DistinctUser"),
		),
	)

	DescribeTable(
		"Should report a reference it cannot follow",
		func(in resolution.TypeRef) {
			_, _, ok := resolution.Dealias(in, table)
			Expect(ok).To(BeFalse())
		},
		Entry("an unresolved name", ref("t.Missing")),
		Entry("an alias of an unresolved name", ref("t.AliasMissing")),
		Entry("a cyclic alias chain", ref("t.CycleA")),
		Entry("a bare type parameter", resolution.TypeRef{TypeParam: &tp}),
	)

	DescribeTable(
		"Should report a distinct type through its aliases",
		func(in resolution.TypeRef, want bool) {
			Expect(resolution.IsDistinct(in, table)).To(Equal(want))
		},
		Entry("a distinct type", ref("t.DistinctUser"), true),
		Entry("an alias of a distinct type", ref("t.AliasDistinct"), true),
		Entry("a struct", ref("t.User"), false),
		Entry("a primitive", ref("uuid"), false),
		Entry("an alias of a primitive", ref("t.AliasUUID"), false),
		Entry("an unresolved name", ref("t.Missing"), false),
	)
})

var _ = Describe("Unified fields through aliases", func() {
	var table *resolution.Table
	ref := func(name string) resolution.TypeRef {
		return resolution.TypeRef{Name: name}
	}
	field := func(name string) resolution.Field {
		return resolution.Field{Name: name, Type: ref("string")}
	}
	names := func(fields []resolution.Field) []string {
		out := make([]string, len(fields))
		for i, f := range fields {
			out[i] = f.Name
		}
		return out
	}

	BeforeEach(func() {
		table = resolution.NewTable()
		add := func(name string, form resolution.TypeForm) {
			GinkgoHelper()
			Expect(table.Add(resolution.Type{
				Name:          name,
				QualifiedName: "t." + name,
				Namespace:     "t",
				Form:          form,
			})).To(Succeed())
		}
		add("Base", resolution.StructForm{
			Fields: []resolution.Field{field("label"), field("hidden")},
		})
		add("AliasBase", resolution.AliasForm{Target: ref("t.Base")})
		add("Payload", resolution.StructForm{
			Fields: []resolution.Field{field("on")},
		})
		add("AliasPayload", resolution.AliasForm{Target: ref("t.Payload")})
		add("Panel", resolution.StructForm{
			Extends: []resolution.TypeRef{ref("t.AliasBase")},
			Fields:  []resolution.Field{field("width")},
		})
	})

	It("Should inherit the fields of a base named through an alias", func() {
		Expect(names(resolution.UnifiedFields(table.MustGet("t.Panel"), table))).To(
			Equal([]string{"label", "hidden", "width"}),
		)
	})

	It("Should flatten a variant and a union base named through aliases", func() {
		union := resolution.Type{
			Name:          "Node",
			QualifiedName: "t.Node",
			Namespace:     "t",
			Form: resolution.UnionForm{
				Discriminator: "variant",
				Extends:       []resolution.TypeRef{ref("t.AliasBase")},
			},
		}
		variant := resolution.UnionVariant{Name: "valve", Type: ref("t.AliasPayload")}
		Expect(names(resolution.UnifiedVariantFields(union, variant, table))).To(
			Equal([]string{"label", "hidden", "on"}),
		)
	})
})

var _ = Describe("InheritedFields", func() {
	var table *resolution.Table
	ref := func(name string, args ...resolution.TypeRef) resolution.TypeRef {
		return resolution.TypeRef{Name: name, TypeArgs: args}
	}
	field := func(name string) resolution.Field {
		return resolution.Field{Name: name, Type: ref("string")}
	}
	tp := resolution.TypeParam{Name: "T"}

	BeforeEach(func() {
		table = resolution.NewTable()
		add := func(name string, form resolution.TypeForm) {
			GinkgoHelper()
			Expect(table.Add(resolution.Type{
				Name:          name,
				QualifiedName: "t." + name,
				Namespace:     "t",
				Form:          form,
			})).To(Succeed())
		}
		add("Root", resolution.StructForm{Fields: []resolution.Field{field("key")}})
		add("Middle", resolution.StructForm{
			Extends: []resolution.TypeRef{ref("t.Root")},
			Fields:  []resolution.Field{field("port")},
		})
		add("Other", resolution.StructForm{
			Fields: []resolution.Field{
				{Name: "port", Type: ref("int32")},
				field("scale"),
			},
		})
		add("Box", resolution.StructForm{
			TypeParams: []resolution.TypeParam{tp},
			Fields: []resolution.Field{
				{Name: "value", Type: resolution.TypeRef{Name: "T", TypeParam: &tp}},
			},
		})
		add("AliasMiddle", resolution.AliasForm{Target: ref("t.Middle")})
		add("AliasBox", resolution.AliasForm{
			Target:     ref("t.Box", resolution.TypeRef{Name: "T", TypeParam: &tp}),
			TypeParams: []resolution.TypeParam{tp},
		})
	})

	It("Should return the fields of every ancestor", func() {
		Expect(
			resolution.InheritedFields([]resolution.TypeRef{ref("t.Middle")}, table),
		).
			To(SatisfyAll(HaveKey("key"), HaveKey("port"), HaveLen(2)))
	})

	It("Should keep the leftmost parent's field on a name collision", func() {
		inherited := resolution.InheritedFields(
			[]resolution.TypeRef{ref("t.Middle"), ref("t.Other")}, table,
		)
		Expect(inherited).To(HaveLen(3))
		Expect(inherited["port"].Type.Name).To(Equal("string"))
	})

	It("Should substitute a generic parent's type arguments", func() {
		inherited := resolution.InheritedFields(
			[]resolution.TypeRef{ref("t.Box", ref("uuid"))}, table,
		)
		Expect(inherited["value"].Type.Name).To(Equal("uuid"))
	})

	It("Should return the fields of a parent named through an alias", func() {
		Expect(resolution.InheritedFields(
			[]resolution.TypeRef{ref("t.AliasMiddle")}, table,
		)).To(SatisfyAll(HaveKey("key"), HaveKey("port")))
	})

	It("Should substitute a type argument passed through a generic alias", func() {
		inherited := resolution.InheritedFields(
			[]resolution.TypeRef{ref("t.AliasBox", ref("uuid"))}, table,
		)
		Expect(inherited["value"].Type.Name).To(Equal("uuid"))
	})

	It("Should return no fields without parents", func() {
		Expect(resolution.InheritedFields(nil, table)).To(BeEmpty())
	})
})

var _ = Describe("TypeArgMap", func() {
	params := []resolution.TypeParam{{Name: "K"}, {Name: "V"}}

	DescribeTable("Should map each parameter to the argument at its position",
		func(args []resolution.TypeRef, expected map[string]resolution.TypeRef) {
			Expect(resolution.TypeArgMap(params, args)).To(Equal(expected))
		},
		Entry("one argument per parameter",
			[]resolution.TypeRef{{Name: "string"}, {Name: "uuid"}},
			map[string]resolution.TypeRef{"K": {Name: "string"}, "V": {Name: "uuid"}},
		),
		Entry("a missing argument",
			[]resolution.TypeRef{{Name: "string"}},
			map[string]resolution.TypeRef{"K": {Name: "string"}},
		),
		Entry("no arguments", nil, map[string]resolution.TypeRef{}),
	)
})

var _ = Describe("MergeOverrideField", func() {
	doc := func(value string) resolution.Domain {
		return resolution.Domain{
			Name: "doc",
			Expressions: resolution.Expressions{{
				Name:   "value",
				Values: []resolution.ExpressionValue{{StringValue: value}},
			}},
		}
	}
	marshal := resolution.Domain{
		Name:        "go",
		Expressions: resolution.Expressions{{Name: "marshal"}},
	}
	parent := resolution.Field{
		Name:     "x",
		Type:     resolution.TypeRef{Name: "int32"},
		Optional: true,
		Default:  &resolution.ExpressionValue{IntValue: 4},
		Domains:  map[string]resolution.Domain{"doc": doc("parent"), "go": marshal},
	}

	It("Should inherit the type, optionality, and default of a typeless child", func() {
		merged := resolution.MergeOverrideField(parent, resolution.Field{Name: "x"})
		Expect(merged.Type.Name).To(Equal("int32"))
		Expect(merged.Optional).To(BeTrue())
		Expect(merged.Default).To(Equal(&resolution.ExpressionValue{IntValue: 4}))
	})

	It("Should keep the optional marker and default of a typeless child", func() {
		merged := resolution.MergeOverrideField(
			resolution.Field{Name: "x", Type: resolution.TypeRef{Name: "int32"}},
			resolution.Field{
				Name:     "x",
				Optional: true,
				Default:  &resolution.ExpressionValue{IntValue: 7},
			},
		)
		Expect(merged.Optional).To(BeTrue())
		Expect(merged.Default).To(Equal(&resolution.ExpressionValue{IntValue: 7}))
	})

	It("Should keep the type, optionality, and default of a typed child", func() {
		merged := resolution.MergeOverrideField(parent, resolution.Field{
			Name: "x",
			Type: resolution.TypeRef{Name: "int64"},
		})
		Expect(merged.Type.Name).To(Equal("int64"))
		Expect(merged.Optional).To(BeFalse())
		Expect(merged.Default).To(BeNil())
	})

	It("Should merge domains, with the child's expressions winning", func() {
		merged := resolution.MergeOverrideField(parent, resolution.Field{
			Name:    "x",
			Domains: map[string]resolution.Domain{"doc": doc("child")},
		})
		Expect(merged.Domains).To(HaveLen(2))
		Expect(merged.Domains["doc"]).To(Equal(doc("child")))
		Expect(merged.Domains["go"]).To(Equal(marshal))
	})

	It("Should drop the domains the child removes", func() {
		merged := resolution.MergeOverrideField(parent, resolution.Field{
			Name:           "x",
			OmittedDomains: []string{"doc"},
		})
		Expect(merged.Domains).To(HaveKey("go"))
		Expect(merged.Domains).ToNot(HaveKey("doc"))
	})
})
