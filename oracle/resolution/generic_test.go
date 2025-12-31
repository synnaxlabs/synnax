// Copyright 2025 Synnax Labs, Inc.
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

func makeStruct(ns, name string, fields ...resolution.Field) resolution.Struct {
	return resolution.Struct{
		Namespace:     ns,
		Name:          name,
		QualifiedName: ns + "." + name,
		Fields:        fields,
		Domains:       make(map[string]resolution.Domain),
	}
}

func makeField(name string, tr *resolution.TypeRef) resolution.Field {
	return resolution.Field{
		Name:    name,
		TypeRef: tr,
		Domains: make(map[string]resolution.Domain),
	}
}

func makePrimitiveRef(name string) *resolution.TypeRef {
	return &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: name}
}

func makeTypeParamRef(name string) *resolution.TypeRef {
	return &resolution.TypeRef{
		Kind:         resolution.TypeKindTypeParam,
		TypeParamRef: &resolution.TypeParam{Name: name},
	}
}

func makeEnum(ns, name string, values ...string) resolution.Enum {
	e := resolution.Enum{
		Namespace:     ns,
		Name:          name,
		QualifiedName: ns + "." + name,
		Domains:       make(map[string]resolution.Domain),
	}
	for _, v := range values {
		ev := resolution.EnumEntry{Name: v}
		e.Values = append(e.Values, ev)
	}
	return e
}

var _ = Describe("Table", func() {
	Describe("NewTable", func() {
		It("initializes empty slices and maps", func() {
			t := resolution.NewTable()
			Expect(t.Structs).To(BeEmpty())
			Expect(t.Enums).To(BeEmpty())
			Expect(t.Imports).To(BeEmpty())
			Expect(t.Namespaces).To(BeEmpty())
		})
	})
	Describe("AddStruct/AddEnum", func() {
		It("appends to the respective slices", func() {
			t := resolution.NewTable()
			s := makeStruct("pkg", "Foo")
			e := makeEnum("pkg", "Bar")
			t.AddStruct(s)
			t.AddEnum(e)
			Expect(t.Structs).To(HaveLen(1))
			Expect(t.Enums).To(HaveLen(1))
			Expect(t.Structs[0]).To(Equal(s))
			Expect(t.Enums[0]).To(Equal(e))
		})
	})
	Describe("AllStructs/AllEnums", func() {
		It("returns all entries", func() {
			t := resolution.NewTable()
			s1 := makeStruct("a", "One")
			s2 := makeStruct("b", "Two")
			e1 := makeEnum("a", "E1")
			t.AddStruct(s1)
			t.AddStruct(s2)
			t.AddEnum(e1)
			Expect(t.AllStructs()).To(HaveLen(2))
			Expect(t.AllEnums()).To(HaveLen(1))
		})
	})
	Describe("GetStruct", func() {
		DescribeTable("qualified name lookup",
			func(qname string, shouldFind bool) {
				t := resolution.NewTable()
				t.AddStruct(makeStruct("pkg", "Foo"))
				s, ok := t.GetStruct(qname)
				Expect(ok).To(Equal(shouldFind))
				if shouldFind {
					Expect(s.Name).To(Equal("Foo"))
				}
			},
			Entry("exact match", "pkg.Foo", true),
			Entry("wrong namespace", "other.Foo", false),
			Entry("wrong name", "pkg.Bar", false),
			Entry("unqualified name does not match", "Foo", false),
		)
	})
	Describe("MustGetStruct", func() {
		It("returns struct when found", func() {
			t := resolution.NewTable()
			t.AddStruct(makeStruct("pkg", "Foo"))
			s := t.MustGetStruct("pkg.Foo")
			Expect(s.Name).To(Equal("Foo"))
		})
		It("panics when not found", func() {
			t := resolution.NewTable()
			Expect(func() { t.MustGetStruct("missing.Struct") }).To(Panic())
		})
	})
	Describe("GetEnum", func() {
		DescribeTable("qualified name lookup",
			func(qname string, shouldFind bool) {
				t := resolution.NewTable()
				t.AddEnum(makeEnum("pkg", "Status", "Active", "Inactive"))
				e, ok := t.GetEnum(qname)
				Expect(ok).To(Equal(shouldFind))
				if shouldFind {
					Expect(e.Name).To(Equal("Status"))
				}
			},
			Entry("exact match", "pkg.Status", true),
			Entry("wrong namespace", "other.Status", false),
			Entry("wrong name", "pkg.State", false),
		)
	})
	Describe("MustGetEnum", func() {
		It("returns enum when found", func() {
			t := resolution.NewTable()
			t.AddEnum(makeEnum("pkg", "Status"))
			e := t.MustGetEnum("pkg.Status")
			Expect(e.Name).To(Equal("Status"))
		})
		It("panics when not found", func() {
			t := resolution.NewTable()
			Expect(func() { t.MustGetEnum("missing.Enum") }).To(Panic())
		})
	})
	Describe("LookupStruct", func() {
		DescribeTable("two-pass resolution",
			func(ns, name string, expectedQName string, shouldFind bool) {
				t := resolution.NewTable()
				t.AddStruct(makeStruct("schema", "User"))
				t.AddStruct(makeStruct("schema", "Group"))
				s, ok := t.LookupStruct(ns, name)
				Expect(ok).To(Equal(shouldFind))
				if shouldFind {
					Expect(s.QualifiedName).To(Equal(expectedQName))
				}
			},
			Entry("exact qualified match", "schema", "User", "schema.User", true),
			Entry("unqualified fallback from different namespace", "other", "User", "schema.User", true),
			Entry("not found", "schema", "Missing", "", false),
		)
		It("prefers qualified match over unqualified", func() {
			t := resolution.NewTable()
			t.AddStruct(makeStruct("a", "Foo"))
			t.AddStruct(makeStruct("b", "Foo"))
			s, ok := t.LookupStruct("b", "Foo")
			Expect(ok).To(BeTrue())
			Expect(s.QualifiedName).To(Equal("b.Foo"))
		})
	})
	Describe("LookupEnum", func() {
		DescribeTable("two-pass resolution",
			func(ns, name string, expectedQName string, shouldFind bool) {
				t := resolution.NewTable()
				t.AddEnum(makeEnum("schema", "Status"))
				e, ok := t.LookupEnum(ns, name)
				Expect(ok).To(Equal(shouldFind))
				if shouldFind {
					Expect(e.QualifiedName).To(Equal(expectedQName))
				}
			},
			Entry("exact qualified match", "schema", "Status", "schema.Status", true),
			Entry("unqualified fallback", "other", "Status", "schema.Status", true),
			Entry("not found", "schema", "Missing", "", false),
		)
	})
	Describe("StructsInNamespace/EnumsInNamespace", func() {
		It("filters by namespace", func() {
			t := resolution.NewTable()
			t.AddStruct(makeStruct("a", "One"))
			t.AddStruct(makeStruct("a", "Two"))
			t.AddStruct(makeStruct("b", "Three"))
			t.AddEnum(makeEnum("a", "E1"))
			t.AddEnum(makeEnum("b", "E2"))
			Expect(t.StructsInNamespace("a")).To(HaveLen(2))
			Expect(t.StructsInNamespace("b")).To(HaveLen(1))
			Expect(t.StructsInNamespace("c")).To(BeEmpty())
			Expect(t.EnumsInNamespace("a")).To(HaveLen(1))
			Expect(t.EnumsInNamespace("b")).To(HaveLen(1))
			Expect(t.EnumsInNamespace("c")).To(BeEmpty())
		})
	})
	Describe("Import tracking", func() {
		DescribeTable("MarkImported/IsImported",
			func(markPath string, checkPath string, expected bool) {
				t := resolution.NewTable()
				if markPath != "" {
					t.MarkImported(markPath)
				}
				Expect(t.IsImported(checkPath)).To(Equal(expected))
			},
			Entry("marked path returns true", "schema/user", "schema/user", true),
			Entry("unmarked path returns false", "", "schema/user", false),
			Entry("different path returns false", "schema/user", "schema/group", false),
		)
	})
})

var _ = Describe("StructEntry", func() {
	Describe("Field", func() {
		DescribeTable("lookup by name",
			func(fieldName string, shouldFind bool) {
				s := makeStruct("pkg", "User",
					makeField("id", makePrimitiveRef("uuid")),
					makeField("name", makePrimitiveRef("string")),
				)
				f, found := s.Field(fieldName)
				Expect(found).To(Equal(shouldFind))
				if shouldFind {
					Expect(f.Name).To(Equal(fieldName))
				}
			},
			Entry("existing field", "id", true),
			Entry("another existing field", "name", true),
			Entry("missing field", "email", false),
		)
	})
	Describe("IsGeneric", func() {
		DescribeTable("checks TypeParams",
			func(typeParams []resolution.TypeParam, expected bool) {
				s := resolution.Struct{TypeParams: typeParams}
				Expect(s.IsGeneric()).To(Equal(expected))
			},
			Entry("with type params", []resolution.TypeParam{{Name: "T"}}, true),
			Entry("empty type params", []resolution.TypeParam{}, false),
			Entry("nil type params", nil, false),
		)
	})
	Describe("IsAlias", func() {
		DescribeTable("checks AliasOf",
			func(aliasOf *resolution.TypeRef, expected bool) {
				s := resolution.Struct{AliasOf: aliasOf}
				Expect(s.IsAlias()).To(Equal(expected))
			},
			Entry("with alias", makePrimitiveRef("string"), true),
			Entry("without alias", nil, false),
		)
	})
	Describe("HasExtends", func() {
		DescribeTable("checks Extends",
			func(extends *resolution.TypeRef, expected bool) {
				s := resolution.Struct{Extends: extends}
				Expect(s.HasExtends()).To(Equal(expected))
			},
			Entry("with extends", &resolution.TypeRef{Kind: resolution.TypeKindStruct}, true),
			Entry("without extends", nil, false),
		)
	})
	Describe("IsFieldOmitted", func() {
		DescribeTable("checks OmittedFields",
			func(omittedFields []string, fieldName string, expected bool) {
				s := resolution.Struct{OmittedFields: omittedFields}
				Expect(s.IsFieldOmitted(fieldName)).To(Equal(expected))
			},
			Entry("field in omitted list", []string{"foo", "bar"}, "foo", true),
			Entry("field not in omitted list", []string{"foo", "bar"}, "baz", false),
			Entry("empty omitted list", []string{}, "foo", false),
			Entry("nil omitted list", nil, "foo", false),
		)
	})
	Describe("TypeParam", func() {
		DescribeTable("lookup by name",
			func(typeParams []resolution.TypeParam, name string, shouldFind bool) {
				s := resolution.Struct{TypeParams: typeParams}
				tp, found := s.TypeParam(name)
				Expect(found).To(Equal(shouldFind))
				if shouldFind {
					Expect(tp.Name).To(Equal(name))
				}
			},
			Entry("existing param", []resolution.TypeParam{{Name: "T"}, {Name: "U"}}, "T", true),
			Entry("missing param", []resolution.TypeParam{{Name: "T"}}, "U", false),
			Entry("empty params", []resolution.TypeParam{}, "T", false),
		)
	})
})

var _ = Describe("AllFields", func() {
	Describe("no inheritance", func() {
		DescribeTable("returns own fields",
			func(fields []resolution.Field, expectedLen int) {
				s := resolution.Struct{Fields: fields, Domains: make(map[string]resolution.Domain)}
				Expect(s.UnifiedFields()).To(HaveLen(expectedLen))
			},
			Entry("single field", []resolution.Field{makeField("id", makePrimitiveRef("uuid"))}, 1),
			Entry("multiple fields", []resolution.Field{
				makeField("id", makePrimitiveRef("uuid")),
				makeField("name", makePrimitiveRef("string")),
			}, 2),
			Entry("empty fields", []resolution.Field{}, 0),
			Entry("nil fields", nil, 0),
		)
	})
	Describe("direct inheritance", func() {
		It("includes parent fields before child fields", func() {
			parent := makeStruct("pkg", "Parent",
				makeField("parentField", makePrimitiveRef("string")),
			)
			child := resolution.Struct{
				Name:      "Child",
				Namespace: "pkg",
				Fields:    []resolution.Field{makeField("childField", makePrimitiveRef("int32"))},
				Extends:   &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parent},
				Domains:   make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(2))
			Expect(fields[0].Name).To(Equal("parentField"))
			Expect(fields[1].Name).To(Equal("childField"))
		})
		It("handles multiple parent fields", func() {
			parent := makeStruct("pkg", "Parent",
				makeField("a", makePrimitiveRef("string")),
				makeField("b", makePrimitiveRef("string")),
				makeField("c", makePrimitiveRef("string")),
			)
			child := resolution.Struct{
				Name:    "Child",
				Fields:  []resolution.Field{makeField("d", makePrimitiveRef("int32"))},
				Extends: &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parent},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(4))
			Expect(fields[0].Name).To(Equal("a"))
			Expect(fields[1].Name).To(Equal("b"))
			Expect(fields[2].Name).To(Equal("c"))
			Expect(fields[3].Name).To(Equal("d"))
		})
	})
	Describe("field omission", func() {
		It("excludes omitted fields", func() {
			parent := makeStruct("pkg", "Parent",
				makeField("keep", makePrimitiveRef("string")),
				makeField("omit", makePrimitiveRef("string")),
			)
			child := resolution.Struct{
				Name:          "Child",
				Fields:        []resolution.Field{makeField("own", makePrimitiveRef("int32"))},
				Extends:       &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parent},
				OmittedFields: []string{"omit"},
				Domains:       make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(2))
			Expect(fields[0].Name).To(Equal("keep"))
			Expect(fields[1].Name).To(Equal("own"))
		})
		It("excludes multiple omitted fields", func() {
			parent := makeStruct("pkg", "Parent",
				makeField("a", makePrimitiveRef("string")),
				makeField("b", makePrimitiveRef("string")),
				makeField("c", makePrimitiveRef("string")),
			)
			child := resolution.Struct{
				Name:          "Child",
				Fields:        []resolution.Field{},
				Extends:       &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parent},
				OmittedFields: []string{"a", "c"},
				Domains:       make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].Name).To(Equal("b"))
		})
	})
	Describe("field override", func() {
		It("child field overrides parent field type", func() {
			parent := makeStruct("pkg", "Parent",
				makeField("field", makePrimitiveRef("string")),
			)
			child := resolution.Struct{
				Name:    "Child",
				Fields:  []resolution.Field{makeField("field", makePrimitiveRef("int64"))},
				Extends: &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parent},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].Name).To(Equal("field"))
			Expect(fields[0].TypeRef.Primitive).To(Equal("int64"))
		})
		It("merges domains from parent and child", func() {
			parentField := makeField("field", makePrimitiveRef("string"))
			parentField.Domains["key"] = resolution.Domain{
				Name:        "key",
				Expressions: []resolution.Expression{{Name: "parentExpr"}},
			}
			parent := resolution.Struct{
				Name:          "Parent",
				Namespace:     "pkg",
				QualifiedName: "pkg.Parent",
				Fields:        []resolution.Field{parentField},
				Domains:       make(map[string]resolution.Domain),
			}
			childField := makeField("field", makePrimitiveRef("int64"))
			childField.Domains["ts"] = resolution.Domain{
				Name:        "ts",
				Expressions: []resolution.Expression{{Name: "childExpr"}},
			}
			child := resolution.Struct{
				Name:    "Child",
				Fields:  []resolution.Field{childField},
				Extends: &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parent},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].Domains).To(HaveKey("key"))
			Expect(fields[0].Domains).To(HaveKey("ts"))
		})
		It("child domain expression overrides parent expression with same name", func() {
			parentField := makeField("field", makePrimitiveRef("string"))
			parentField.Domains["doc"] = resolution.Domain{
				Name:        "doc",
				Expressions: []resolution.Expression{{Name: "shared", Values: []resolution.ExpressionValue{{StringValue: "parent"}}}},
			}
			parent := resolution.Struct{
				Name:          "Parent",
				Namespace:     "pkg",
				QualifiedName: "pkg.Parent",
				Fields:        []resolution.Field{parentField},
				Domains:       make(map[string]resolution.Domain),
			}
			childField := makeField("field", makePrimitiveRef("int64"))
			childField.Domains["doc"] = resolution.Domain{
				Name:        "doc",
				Expressions: []resolution.Expression{{Name: "shared", Values: []resolution.ExpressionValue{{StringValue: "child"}}}},
			}
			child := resolution.Struct{
				Name:    "Child",
				Fields:  []resolution.Field{childField},
				Extends: &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parent},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].Domains["doc"].Expressions).To(HaveLen(1))
			Expect(fields[0].Domains["doc"].Expressions[0].Values[0].StringValue).To(Equal("child"))
		})
	})
	Describe("generic parent substitution", func() {
		It("substitutes type parameter with concrete type", func() {
			parent := resolution.Struct{
				Name:          "Parent",
				Namespace:     "pkg",
				QualifiedName: "pkg.Parent",
				Fields:        []resolution.Field{makeField("value", makeTypeParamRef("T"))},
				TypeParams:    []resolution.TypeParam{{Name: "T"}},
				Domains:       make(map[string]resolution.Domain),
			}
			child := resolution.Struct{
				Name:   "Child",
				Fields: []resolution.Field{},
				Extends: &resolution.TypeRef{
					Kind:      resolution.TypeKindStruct,
					StructRef: &parent,
					TypeArgs:  []*resolution.TypeRef{makePrimitiveRef("string")},
				},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].TypeRef.Kind).To(Equal(resolution.TypeKindPrimitive))
			Expect(fields[0].TypeRef.Primitive).To(Equal("string"))
		})
		It("preserves IsArray flag during substitution", func() {
			parent := resolution.Struct{
				Name:          "Parent",
				Namespace:     "pkg",
				QualifiedName: "pkg.Parent",
				Fields: []resolution.Field{
					makeField("values", &resolution.TypeRef{
						Kind:         resolution.TypeKindTypeParam,
						TypeParamRef: &resolution.TypeParam{Name: "T"},
						IsArray:      true,
					}),
				},
				TypeParams: []resolution.TypeParam{{Name: "T"}},
				Domains:    make(map[string]resolution.Domain),
			}
			child := resolution.Struct{
				Name:   "Child",
				Fields: []resolution.Field{},
				Extends: &resolution.TypeRef{
					Kind:      resolution.TypeKindStruct,
					StructRef: &parent,
					TypeArgs:  []*resolution.TypeRef{makePrimitiveRef("int32")},
				},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].TypeRef.IsArray).To(BeTrue())
			Expect(fields[0].TypeRef.Primitive).To(Equal("int32"))
		})
		It("preserves IsOptional flag during substitution", func() {
			parent := resolution.Struct{
				Name:          "Parent",
				Namespace:     "pkg",
				QualifiedName: "pkg.Parent",
				Fields: []resolution.Field{
					makeField("maybe", &resolution.TypeRef{
						Kind:         resolution.TypeKindTypeParam,
						TypeParamRef: &resolution.TypeParam{Name: "T"},
						IsOptional:   true,
					}),
				},
				TypeParams: []resolution.TypeParam{{Name: "T"}},
				Domains:    make(map[string]resolution.Domain),
			}
			child := resolution.Struct{
				Name:   "Child",
				Fields: []resolution.Field{},
				Extends: &resolution.TypeRef{
					Kind:      resolution.TypeKindStruct,
					StructRef: &parent,
					TypeArgs:  []*resolution.TypeRef{makePrimitiveRef("string")},
				},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].TypeRef.IsOptional).To(BeTrue())
		})
		It("substitutes nested type args in generic types", func() {
			listStruct := makeStruct("pkg", "List")
			parent := resolution.Struct{
				Name:          "Parent",
				Namespace:     "pkg",
				QualifiedName: "pkg.Parent",
				Fields: []resolution.Field{
					makeField("items", &resolution.TypeRef{
						Kind:      resolution.TypeKindStruct,
						StructRef: &listStruct,
						TypeArgs:  []*resolution.TypeRef{makeTypeParamRef("T")},
					}),
				},
				TypeParams: []resolution.TypeParam{{Name: "T"}},
				Domains:    make(map[string]resolution.Domain),
			}
			child := resolution.Struct{
				Name:   "Child",
				Fields: []resolution.Field{},
				Extends: &resolution.TypeRef{
					Kind:      resolution.TypeKindStruct,
					StructRef: &parent,
					TypeArgs:  []*resolution.TypeRef{makePrimitiveRef("uuid")},
				},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].TypeRef.TypeArgs).To(HaveLen(1))
			Expect(fields[0].TypeRef.TypeArgs[0].Primitive).To(Equal("uuid"))
		})
		It("substitutes map key and value types", func() {
			parent := resolution.Struct{
				Name:          "Parent",
				Namespace:     "pkg",
				QualifiedName: "pkg.Parent",
				Fields: []resolution.Field{
					makeField("lookup", &resolution.TypeRef{
						Kind:         resolution.TypeKindMap,
						MapKeyType:   makeTypeParamRef("K"),
						MapValueType: makeTypeParamRef("V"),
					}),
				},
				TypeParams: []resolution.TypeParam{{Name: "K"}, {Name: "V"}},
				Domains:    make(map[string]resolution.Domain),
			}
			child := resolution.Struct{
				Name:   "Child",
				Fields: []resolution.Field{},
				Extends: &resolution.TypeRef{
					Kind:      resolution.TypeKindStruct,
					StructRef: &parent,
					TypeArgs:  []*resolution.TypeRef{makePrimitiveRef("string"), makePrimitiveRef("int64")},
				},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].TypeRef.MapKeyType.Primitive).To(Equal("string"))
			Expect(fields[0].TypeRef.MapValueType.Primitive).To(Equal("int64"))
		})
	})
	Describe("deep inheritance", func() {
		It("handles grandparent -> parent -> child chain", func() {
			grandparent := makeStruct("pkg", "Grandparent",
				makeField("gpField", makePrimitiveRef("string")),
			)
			parent := resolution.Struct{
				Name:          "Parent",
				Namespace:     "pkg",
				QualifiedName: "pkg.Parent",
				Fields:        []resolution.Field{makeField("pField", makePrimitiveRef("int32"))},
				Extends:       &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &grandparent},
				Domains:       make(map[string]resolution.Domain),
			}
			child := resolution.Struct{
				Name:    "Child",
				Fields:  []resolution.Field{makeField("cField", makePrimitiveRef("bool"))},
				Extends: &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parent},
				Domains: make(map[string]resolution.Domain),
			}
			fields := child.UnifiedFields()
			Expect(fields).To(HaveLen(3))
			Expect(fields[0].Name).To(Equal("gpField"))
			Expect(fields[1].Name).To(Equal("pField"))
			Expect(fields[2].Name).To(Equal("cField"))
		})
	})
})

var _ = Describe("Primitives", func() {
	DescribeTable("IsPrimitive",
		func(name string, expected bool) {
			Expect(resolution.IsPrimitive(name)).To(Equal(expected))
		},
		Entry("uuid", "uuid", true),
		Entry("string", "string", true),
		Entry("bool", "bool", true),
		Entry("int8", "int8", true),
		Entry("int16", "int16", true),
		Entry("int32", "int32", true),
		Entry("int64", "int64", true),
		Entry("uint8", "uint8", true),
		Entry("uint16", "uint16", true),
		Entry("uint32", "uint32", true),
		Entry("uint64", "uint64", true),
		Entry("float32", "float32", true),
		Entry("float64", "float64", true),
		Entry("timestamp", "timestamp", true),
		Entry("timespan", "timespan", true),
		Entry("time_range", "time_range", true),
		Entry("time_range_bounded", "time_range_bounded", true),
		Entry("json", "json", true),
		Entry("bytes", "bytes", true),
		Entry("non-primitive", "Foo", false),
		Entry("empty string", "", false),
	)
	DescribeTable("IsStringPrimitive",
		func(name string, expected bool) {
			Expect(resolution.IsStringPrimitive(name)).To(Equal(expected))
		},
		Entry("string", "string", true),
		Entry("uuid", "uuid", true),
		Entry("int32", "int32", false),
		Entry("bool", "bool", false),
		Entry("non-primitive", "Foo", false),
	)
	DescribeTable("IsNumberPrimitive",
		func(name string, expected bool) {
			Expect(resolution.IsNumberPrimitive(name)).To(Equal(expected))
		},
		Entry("int8", "int8", true),
		Entry("int16", "int16", true),
		Entry("int32", "int32", true),
		Entry("int64", "int64", true),
		Entry("uint8", "uint8", true),
		Entry("uint16", "uint16", true),
		Entry("uint32", "uint32", true),
		Entry("uint64", "uint64", true),
		Entry("float32", "float32", true),
		Entry("float64", "float64", true),
		Entry("string", "string", false),
		Entry("bool", "bool", false),
		Entry("timestamp", "timestamp", false),
	)
})

var _ = Describe("EnumEntry", func() {
	Describe("QualifiedName", func() {
		It("uses namespace.name format", func() {
			e := makeEnum("schema", "Status")
			Expect(e.QualifiedName).To(Equal("schema.Status"))
		})
	})
})

var _ = Describe("TypeRef", func() {
	DescribeTable("Kind",
		func(tr *resolution.TypeRef, expectedKind resolution.TypeKind) {
			Expect(tr.Kind).To(Equal(expectedKind))
		},
		Entry("primitive", makePrimitiveRef("string"), resolution.TypeKindPrimitive),
		Entry("struct", &resolution.TypeRef{Kind: resolution.TypeKindStruct}, resolution.TypeKindStruct),
		Entry("enum", &resolution.TypeRef{Kind: resolution.TypeKindEnum}, resolution.TypeKindEnum),
		Entry("type param", makeTypeParamRef("T"), resolution.TypeKindTypeParam),
		Entry("map", &resolution.TypeRef{Kind: resolution.TypeKindMap}, resolution.TypeKindMap),
		Entry("unresolved", &resolution.TypeRef{Kind: resolution.TypeKindUnresolved}, resolution.TypeKindUnresolved),
	)
	Describe("modifiers", func() {
		It("can have IsArray set", func() {
			tr := &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "int32", IsArray: true}
			Expect(tr.IsArray).To(BeTrue())
		})
		It("can have IsOptional set", func() {
			tr := &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string", IsOptional: true}
			Expect(tr.IsOptional).To(BeTrue())
		})
		It("can have IsHardOptional set", func() {
			tr := &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string", IsHardOptional: true}
			Expect(tr.IsHardOptional).To(BeTrue())
		})
	})
})

var _ = Describe("ValueKind", func() {
	DescribeTable("constants",
		func(kind resolution.ValueKind, expectedValue int) {
			Expect(int(kind)).To(Equal(expectedValue))
		},
		Entry("String", resolution.ValueKindString, 0),
		Entry("Int", resolution.ValueKindInt, 1),
		Entry("Float", resolution.ValueKindFloat, 2),
		Entry("Bool", resolution.ValueKindBool, 3),
		Entry("Ident", resolution.ValueKindIdent, 4),
	)
})

var _ = Describe("TypeKind", func() {
	DescribeTable("constants",
		func(kind resolution.TypeKind, expectedValue int) {
			Expect(int(kind)).To(Equal(expectedValue))
		},
		Entry("Primitive", resolution.TypeKindPrimitive, 0),
		Entry("Struct", resolution.TypeKindStruct, 1),
		Entry("Enum", resolution.TypeKindEnum, 2),
		Entry("TypeParam", resolution.TypeKindTypeParam, 3),
		Entry("Map", resolution.TypeKindMap, 4),
		Entry("Unresolved", resolution.TypeKindUnresolved, 5),
	)
})
