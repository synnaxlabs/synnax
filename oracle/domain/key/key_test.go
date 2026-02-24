// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package key_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Collect", func() {
	It("should return empty for nil input", func() {
		Expect(key.Collect(nil, nil, nil)).To(BeEmpty())
	})

	It("should return empty for structs without key domain", func() {
		types := []resolution.Type{{
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "name",
					Type:    resolution.TypeRef{Name: "string"},
					Domains: map[string]resolution.Domain{},
				}},
			},
		}}
		Expect(key.Collect(types, nil, nil)).To(BeEmpty())
	})

	It("should collect field with key domain", func() {
		types := []resolution.Type{{
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "key",
					Type:    resolution.TypeRef{Name: "uuid"},
					Domains: map[string]resolution.Domain{"key": {}},
				}},
			},
		}}
		result := key.Collect(types, nil, nil)
		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("key"))
		Expect(result[0].Primitive).To(Equal("uuid"))
	})

	It("should deduplicate by field name", func() {
		types := []resolution.Type{
			{Form: resolution.StructForm{Fields: []resolution.Field{{
				Name:    "key",
				Type:    resolution.TypeRef{Name: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}}}},
			{Form: resolution.StructForm{Fields: []resolution.Field{{
				Name:    "key",
				Type:    resolution.TypeRef{Name: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}}}},
		}
		Expect(key.Collect(types, nil, nil)).To(HaveLen(1))
	})

	It("should collect multiple different key fields", func() {
		types := []resolution.Type{{
			Form: resolution.StructForm{
				Fields: []resolution.Field{
					{
						Name:    "key",
						Type:    resolution.TypeRef{Name: "uuid"},
						Domains: map[string]resolution.Domain{"key": {}},
					},
					{
						Name:    "rack",
						Type:    resolution.TypeRef{Name: "uint32"},
						Domains: map[string]resolution.Domain{"key": {}},
					},
				},
			},
		}}
		result := key.Collect(types, nil, nil)
		Expect(result).To(HaveLen(2))
	})

	It("should skip types when skip function returns true", func() {
		types := []resolution.Type{{
			Name: "Skipped",
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "key",
					Type:    resolution.TypeRef{Name: "uuid"},
					Domains: map[string]resolution.Domain{"key": {}},
				}},
			},
		}}
		skip := func(t resolution.Type) bool { return t.Name == "Skipped" }
		Expect(key.Collect(types, nil, skip)).To(BeEmpty())
	})

	It("should not skip when skip function returns false", func() {
		types := []resolution.Type{{
			Name: "NotSkipped",
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "key",
					Type:    resolution.TypeRef{Name: "uuid"},
					Domains: map[string]resolution.Domain{"key": {}},
				}},
			},
		}}
		skip := func(t resolution.Type) bool { return t.Name == "Skipped" }
		Expect(key.Collect(types, nil, skip)).To(HaveLen(1))
	})

	It("should skip non-struct types", func() {
		types := []resolution.Type{{
			Form: resolution.EnumForm{Values: []resolution.EnumValue{{Name: "VALUE"}}},
		}}
		Expect(key.Collect(types, nil, nil)).To(BeEmpty())
	})

	It("should resolve distinct type to primitive", func() {
		table := resolution.NewTable()
		Expect(table.Add(resolution.Type{
			Name:          "UserID",
			QualifiedName: "pkg.UserID",
			Form:          resolution.DistinctForm{Base: resolution.TypeRef{Name: "uuid"}},
		})).To(Succeed())

		types := []resolution.Type{{
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "user_id",
					Type:    resolution.TypeRef{Name: "pkg.UserID"},
					Domains: map[string]resolution.Domain{"key": {}},
				}},
			},
		}}
		result := key.Collect(types, table, nil)
		Expect(result).To(HaveLen(1))
		Expect(result[0].Primitive).To(Equal("uuid"))
	})
})

var _ = Describe("HasKey", func() {
	It("returns true when field has key domain", func() {
		field := resolution.Field{
			Name:    "id",
			Domains: map[string]resolution.Domain{"key": {}},
		}
		Expect(key.HasKey(field)).To(BeTrue())
	})

	It("returns false when field has no key domain", func() {
		field := resolution.Field{
			Name:    "name",
			Domains: map[string]resolution.Domain{"doc": {}},
		}
		Expect(key.HasKey(field)).To(BeFalse())
	})

	It("returns false when field has no domains", func() {
		field := resolution.Field{Name: "name"}
		Expect(key.HasKey(field)).To(BeFalse())
	})

	It("returns false when domains is empty map", func() {
		field := resolution.Field{
			Name:    "name",
			Domains: map[string]resolution.Domain{},
		}
		Expect(key.HasKey(field)).To(BeFalse())
	})
})

var _ = Describe("HasGenerate", func() {
	It("returns true when key domain has generate expression", func() {
		field := resolution.Field{
			Name: "id",
			Domains: map[string]resolution.Domain{
				"key": {
					Expressions: resolution.Expressions{{Name: "generate"}},
				},
			},
		}
		Expect(key.HasGenerate(field)).To(BeTrue())
	})

	It("returns false when key domain has no generate expression", func() {
		field := resolution.Field{
			Name: "id",
			Domains: map[string]resolution.Domain{
				"key": {
					Expressions: resolution.Expressions{{Name: "primary"}},
				},
			},
		}
		Expect(key.HasGenerate(field)).To(BeFalse())
	})

	It("returns false when field has no key domain", func() {
		field := resolution.Field{
			Name:    "id",
			Domains: map[string]resolution.Domain{},
		}
		Expect(key.HasGenerate(field)).To(BeFalse())
	})

	It("returns false when key domain has empty expressions", func() {
		field := resolution.Field{
			Name: "id",
			Domains: map[string]resolution.Domain{
				"key": {Expressions: resolution.Expressions{}},
			},
		}
		Expect(key.HasGenerate(field)).To(BeFalse())
	})
})

var _ = Describe("ResolvePrimitive", func() {
	var table *resolution.Table

	BeforeEach(func() {
		table = resolution.NewTable()
	})

	It("returns primitive name directly", func() {
		ref := resolution.TypeRef{Name: "uuid"}
		Expect(key.ResolvePrimitive(ref, table)).To(Equal("uuid"))
	})

	It("returns empty for type parameters", func() {
		ref := resolution.TypeRef{TypeParam: &resolution.TypeParam{Name: "T"}}
		Expect(key.ResolvePrimitive(ref, table)).To(BeEmpty())
	})

	It("follows distinct type to base primitive", func() {
		Expect(table.Add(resolution.Type{
			Name:          "UserID",
			QualifiedName: "pkg.UserID",
			Form:          resolution.DistinctForm{Base: resolution.TypeRef{Name: "uuid"}},
		})).To(Succeed())

		ref := resolution.TypeRef{Name: "pkg.UserID"}
		Expect(key.ResolvePrimitive(ref, table)).To(Equal("uuid"))
	})

	It("follows nested distinct types", func() {
		Expect(table.Add(resolution.Type{
			Name:          "BaseID",
			QualifiedName: "pkg.BaseID",
			Form:          resolution.DistinctForm{Base: resolution.TypeRef{Name: "uint64"}},
		})).To(Succeed())
		Expect(table.Add(resolution.Type{
			Name:          "UserID",
			QualifiedName: "pkg.UserID",
			Form:          resolution.DistinctForm{Base: resolution.TypeRef{Name: "pkg.BaseID"}},
		})).To(Succeed())

		ref := resolution.TypeRef{Name: "pkg.UserID"}
		Expect(key.ResolvePrimitive(ref, table)).To(Equal("uint64"))
	})

	It("returns empty for non-primitive non-distinct types", func() {
		Expect(table.Add(resolution.Type{
			Name:          "User",
			QualifiedName: "pkg.User",
			Form:          resolution.StructForm{},
		})).To(Succeed())

		ref := resolution.TypeRef{Name: "pkg.User"}
		Expect(key.ResolvePrimitive(ref, table)).To(BeEmpty())
	})

	It("returns empty for unknown type references", func() {
		ref := resolution.TypeRef{Name: "pkg.Unknown"}
		Expect(key.ResolvePrimitive(ref, table)).To(BeEmpty())
	})
})
