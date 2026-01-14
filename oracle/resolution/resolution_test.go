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

var _ = Describe("Table", func() {
	var table *resolution.Table

	BeforeEach(func() {
		table = resolution.NewTable()
	})

	Describe("Built-in Types", func() {
		DescribeTable("registers primitive types",
			func(name string) {
				typ, ok := table.Get(name)
				Expect(ok).To(BeTrue(), "primitive %s should be registered", name)
				_, isPrimitive := typ.Form.(resolution.PrimitiveForm)
				Expect(isPrimitive).To(BeTrue(), "%s should be PrimitiveForm", name)
			},
			Entry("int8", "int8"),
			Entry("int16", "int16"),
			Entry("int32", "int32"),
			Entry("int64", "int64"),
			Entry("uint8", "uint8"),
			Entry("uint12", "uint12"),
			Entry("uint16", "uint16"),
			Entry("uint20", "uint20"),
			Entry("uint32", "uint32"),
			Entry("uint64", "uint64"),
			Entry("float32", "float32"),
			Entry("float64", "float64"),
			Entry("bool", "bool"),
			Entry("string", "string"),
			Entry("uuid", "uuid"),
			Entry("json", "json"),
			Entry("bytes", "bytes"),
			Entry("any", "any"),
		)

		It("registers Array generic builtin with arity 1", func() {
			typ, ok := table.Get("Array")
			Expect(ok).To(BeTrue())
			form, isBuiltinGeneric := typ.Form.(resolution.BuiltinGenericForm)
			Expect(isBuiltinGeneric).To(BeTrue())
			Expect(form.Name).To(Equal("Array"))
			Expect(form.Arity).To(Equal(1))
		})

		It("registers Map generic builtin with arity 2", func() {
			typ, ok := table.Get("Map")
			Expect(ok).To(BeTrue())
			form, isBuiltinGeneric := typ.Form.(resolution.BuiltinGenericForm)
			Expect(isBuiltinGeneric).To(BeTrue())
			Expect(form.Name).To(Equal("Map"))
			Expect(form.Arity).To(Equal(2))
		})
	})

	Describe("Add", func() {
		It("adds new type successfully", func() {
			typ := resolution.Type{
				Name:          "User",
				QualifiedName: "auth.User",
				Namespace:     "auth",
				Form:          resolution.StructForm{Fields: []resolution.Field{}},
			}
			err := table.Add(typ)
			Expect(err).To(Succeed())

			retrieved, ok := table.Get("auth.User")
			Expect(ok).To(BeTrue())
			Expect(retrieved.Name).To(Equal("User"))
			Expect(retrieved.Namespace).To(Equal("auth"))
		})

		It("rejects duplicate qualified names", func() {
			typ := resolution.Type{
				Name:          "User",
				QualifiedName: "auth.User",
				Namespace:     "auth",
				Form:          resolution.StructForm{},
			}
			Expect(table.Add(typ)).To(Succeed())
			err := table.Add(typ)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate type"))
		})
	})

	Describe("Get", func() {
		It("returns false for non-existent type", func() {
			_, ok := table.Get("nonexistent.Type")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("MustGet", func() {
		It("panics for non-existent type", func() {
			Expect(func() {
				table.MustGet("nonexistent.Type")
			}).To(Panic())
		})

		It("returns type when it exists", func() {
			Expect(table.Add(resolution.Type{
				Name:          "User",
				QualifiedName: "auth.User",
				Form:          resolution.StructForm{},
			})).To(Succeed())
			typ := table.MustGet("auth.User")
			Expect(typ.Name).To(Equal("User"))
		})
	})

	Describe("Lookup", func() {
		BeforeEach(func() {
			Expect(table.Add(resolution.Type{
				Name:          "User",
				QualifiedName: "auth.User",
				Namespace:     "auth",
				Form:          resolution.StructForm{},
			})).To(Succeed())
		})

		It("finds type by qualified namespace.name", func() {
			typ, ok := table.Lookup("auth", "User")
			Expect(ok).To(BeTrue())
			Expect(typ.QualifiedName).To(Equal("auth.User"))
		})

		It("falls back to unqualified search for primitives", func() {
			typ, ok := table.Lookup("auth", "string")
			Expect(ok).To(BeTrue())
			Expect(typ.Name).To(Equal("string"))
		})

		It("returns false for non-existent types", func() {
			_, ok := table.Lookup("auth", "NonExistent")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Type Filters", func() {
		BeforeEach(func() {
			Expect(table.Add(resolution.Type{
				Name: "User", QualifiedName: "auth.User", Namespace: "auth",
				Form: resolution.StructForm{},
			})).To(Succeed())
			Expect(table.Add(resolution.Type{
				Name: "Status", QualifiedName: "auth.Status", Namespace: "auth",
				Form: resolution.EnumForm{Values: []resolution.EnumValue{}},
			})).To(Succeed())
			Expect(table.Add(resolution.Type{
				Name: "UserID", QualifiedName: "auth.UserID", Namespace: "auth",
				Form: resolution.DistinctForm{Base: resolution.TypeRef{Name: "uuid"}},
			})).To(Succeed())
			Expect(table.Add(resolution.Type{
				Name: "StringAlias", QualifiedName: "auth.StringAlias", Namespace: "auth",
				Form: resolution.AliasForm{Target: resolution.TypeRef{Name: "string"}},
			})).To(Succeed())
			Expect(table.Add(resolution.Type{
				Name: "Other", QualifiedName: "other.Other", Namespace: "other",
				Form: resolution.StructForm{},
			})).To(Succeed())
		})

		It("filters struct types", func() {
			structs := table.StructTypes()
			Expect(structs).To(HaveLen(2))
			names := []string{structs[0].Name, structs[1].Name}
			Expect(names).To(ContainElements("User", "Other"))
		})

		It("filters enum types", func() {
			enums := table.EnumTypes()
			Expect(enums).To(HaveLen(1))
			Expect(enums[0].Name).To(Equal("Status"))
		})

		It("filters distinct types", func() {
			distincts := table.DistinctTypes()
			Expect(distincts).To(HaveLen(1))
			Expect(distincts[0].Name).To(Equal("UserID"))
		})

		It("filters alias types", func() {
			aliases := table.AliasTypes()
			Expect(aliases).To(HaveLen(1))
			Expect(aliases[0].Name).To(Equal("StringAlias"))
		})

		It("filters types by namespace", func() {
			types := table.TypesInNamespace("auth")
			Expect(types).To(HaveLen(4))
		})

		It("filters structs in namespace", func() {
			structs := table.StructsInNamespace("auth")
			Expect(structs).To(HaveLen(1))
			Expect(structs[0].Name).To(Equal("User"))
		})

		It("filters enums in namespace", func() {
			enums := table.EnumsInNamespace("auth")
			Expect(enums).To(HaveLen(1))
			Expect(enums[0].Name).To(Equal("Status"))
		})

		It("filters types with specific domain", func() {
			Expect(table.Add(resolution.Type{
				Name: "WithGo", QualifiedName: "pkg.WithGo", Namespace: "pkg",
				Form:    resolution.StructForm{},
				Domains: map[string]resolution.Domain{"go": {}},
			})).To(Succeed())
			types := table.TypesWithDomain("go")
			Expect(types).To(HaveLen(1))
			Expect(types[0].Name).To(Equal("WithGo"))
		})
	})

	Describe("Primitive Checks", func() {
		DescribeTable("IsPrimitiveType",
			func(name string, expected bool) {
				Expect(table.IsPrimitiveType(name)).To(Equal(expected))
			},
			Entry("string is primitive", "string", true),
			Entry("int32 is primitive", "int32", true),
			Entry("uuid is primitive", "uuid", true),
			Entry("non-existent is not primitive", "auth.User", false),
			Entry("Array is not primitive", "Array", false),
		)

		DescribeTable("IsStringPrimitiveType",
			func(name string, expected bool) {
				Expect(table.IsStringPrimitiveType(name)).To(Equal(expected))
			},
			Entry("string", "string", true),
			Entry("uuid", "uuid", true),
			Entry("int32", "int32", false),
			Entry("bool", "bool", false),
		)

		DescribeTable("IsNumberPrimitiveType",
			func(name string, expected bool) {
				Expect(table.IsNumberPrimitiveType(name)).To(Equal(expected))
			},
			Entry("int8", "int8", true),
			Entry("int32", "int32", true),
			Entry("int64", "int64", true),
			Entry("uint8", "uint8", true),
			Entry("uint32", "uint32", true),
			Entry("float32", "float32", true),
			Entry("float64", "float64", true),
			Entry("string", "string", false),
			Entry("bool", "bool", false),
		)
	})

	Describe("Imports", func() {
		It("marks and checks imported paths", func() {
			Expect(table.IsImported("github.com/foo/bar")).To(BeFalse())
			table.MarkImported("github.com/foo/bar")
			Expect(table.IsImported("github.com/foo/bar")).To(BeTrue())
			Expect(table.IsImported("github.com/baz/qux")).To(BeFalse())
		})
	})

	Describe("TopologicalSort", func() {
		It("returns empty slice unchanged", func() {
			sorted := table.TopologicalSort([]resolution.Type{})
			Expect(sorted).To(BeEmpty())
		})

		It("returns single element unchanged", func() {
			types := []resolution.Type{{Name: "A", QualifiedName: "ns.A", Form: resolution.StructForm{}}}
			sorted := table.TopologicalSort(types)
			Expect(sorted).To(HaveLen(1))
			Expect(sorted[0].Name).To(Equal("A"))
		})

		It("preserves original order when no dependencies", func() {
			types := []resolution.Type{
				{Name: "A", QualifiedName: "ns.A", Form: resolution.StructForm{}},
				{Name: "B", QualifiedName: "ns.B", Form: resolution.StructForm{}},
				{Name: "C", QualifiedName: "ns.C", Form: resolution.StructForm{}},
			}
			sorted := table.TopologicalSort(types)
			Expect(sorted).To(HaveLen(3))
			Expect(sorted[0].Name).To(Equal("A"))
			Expect(sorted[1].Name).To(Equal("B"))
			Expect(sorted[2].Name).To(Equal("C"))
		})

		It("places dependencies before dependents", func() {
			base := resolution.Type{
				Name: "Base", QualifiedName: "ns.Base", Namespace: "ns",
				Form: resolution.StructForm{},
			}
			derived := resolution.Type{
				Name: "Derived", QualifiedName: "ns.Derived", Namespace: "ns",
				Form: resolution.StructForm{
					Fields: []resolution.Field{
						{Name: "base", Type: resolution.TypeRef{Name: "ns.Base"}},
					},
				},
			}
			Expect(table.Add(base)).To(Succeed())
			Expect(table.Add(derived)).To(Succeed())

			// Pass in reverse order - Derived first, then Base
			types := []resolution.Type{table.MustGet("ns.Derived"), table.MustGet("ns.Base")}
			sorted := table.TopologicalSort(types)
			Expect(sorted[0].Name).To(Equal("Base"))
			Expect(sorted[1].Name).To(Equal("Derived"))
		})

		It("handles transitive dependencies", func() {
			a := resolution.Type{
				Name: "A", QualifiedName: "ns.A", Namespace: "ns",
				Form: resolution.StructForm{},
			}
			b := resolution.Type{
				Name: "B", QualifiedName: "ns.B", Namespace: "ns",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{Type: resolution.TypeRef{Name: "ns.A"}}},
				},
			}
			c := resolution.Type{
				Name: "C", QualifiedName: "ns.C", Namespace: "ns",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{Type: resolution.TypeRef{Name: "ns.B"}}},
				},
			}
			Expect(table.Add(a)).To(Succeed())
			Expect(table.Add(b)).To(Succeed())
			Expect(table.Add(c)).To(Succeed())

			types := []resolution.Type{table.MustGet("ns.C"), table.MustGet("ns.A"), table.MustGet("ns.B")}
			sorted := table.TopologicalSort(types)
			// A must come before B, B must come before C
			var aIdx, bIdx, cIdx int
			for i, t := range sorted {
				switch t.Name {
				case "A":
					aIdx = i
				case "B":
					bIdx = i
				case "C":
					cIdx = i
				}
			}
			Expect(aIdx).To(BeNumerically("<", bIdx))
			Expect(bIdx).To(BeNumerically("<", cIdx))
		})

		It("handles cyclic dependencies gracefully", func() {
			a := resolution.Type{
				Name: "A", QualifiedName: "ns.A", Namespace: "ns",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{Type: resolution.TypeRef{Name: "ns.B"}}},
				},
			}
			b := resolution.Type{
				Name: "B", QualifiedName: "ns.B", Namespace: "ns",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{Type: resolution.TypeRef{Name: "ns.A"}}},
				},
			}
			Expect(table.Add(a)).To(Succeed())
			Expect(table.Add(b)).To(Succeed())

			types := []resolution.Type{table.MustGet("ns.A"), table.MustGet("ns.B")}
			sorted := table.TopologicalSort(types)
			// Both types should be present despite the cycle
			Expect(sorted).To(HaveLen(2))
			names := []string{sorted[0].Name, sorted[1].Name}
			Expect(names).To(ContainElements("A", "B"))
		})

		It("handles alias dependencies", func() {
			base := resolution.Type{
				Name: "Base", QualifiedName: "ns.Base", Namespace: "ns",
				Form: resolution.StructForm{},
			}
			alias := resolution.Type{
				Name: "BaseAlias", QualifiedName: "ns.BaseAlias", Namespace: "ns",
				Form: resolution.AliasForm{Target: resolution.TypeRef{Name: "ns.Base"}},
			}
			Expect(table.Add(base)).To(Succeed())
			Expect(table.Add(alias)).To(Succeed())

			types := []resolution.Type{table.MustGet("ns.BaseAlias"), table.MustGet("ns.Base")}
			sorted := table.TopologicalSort(types)
			Expect(sorted[0].Name).To(Equal("Base"))
			Expect(sorted[1].Name).To(Equal("BaseAlias"))
		})

		It("handles distinct dependencies", func() {
			base := resolution.Type{
				Name: "Base", QualifiedName: "ns.Base", Namespace: "ns",
				Form: resolution.StructForm{},
			}
			distinct := resolution.Type{
				Name: "BaseDistinct", QualifiedName: "ns.BaseDistinct", Namespace: "ns",
				Form: resolution.DistinctForm{Base: resolution.TypeRef{Name: "ns.Base"}},
			}
			Expect(table.Add(base)).To(Succeed())
			Expect(table.Add(distinct)).To(Succeed())

			types := []resolution.Type{table.MustGet("ns.BaseDistinct"), table.MustGet("ns.Base")}
			sorted := table.TopologicalSort(types)
			Expect(sorted[0].Name).To(Equal("Base"))
			Expect(sorted[1].Name).To(Equal("BaseDistinct"))
		})

		It("handles extends dependencies", func() {
			parent := resolution.Type{
				Name: "Parent", QualifiedName: "ns.Parent", Namespace: "ns",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{Name: "id", Type: resolution.TypeRef{Name: "uuid"}}},
				},
			}
			child := resolution.Type{
				Name: "Child", QualifiedName: "ns.Child", Namespace: "ns",
				Form: resolution.StructForm{
					Extends: []resolution.TypeRef{{Name: "ns.Parent"}},
					Fields:  []resolution.Field{{Name: "name", Type: resolution.TypeRef{Name: "string"}}},
				},
			}
			Expect(table.Add(parent)).To(Succeed())
			Expect(table.Add(child)).To(Succeed())

			types := []resolution.Type{table.MustGet("ns.Child"), table.MustGet("ns.Parent")}
			sorted := table.TopologicalSort(types)
			Expect(sorted[0].Name).To(Equal("Parent"))
			Expect(sorted[1].Name).To(Equal("Child"))
		})
	})
})

var _ = Describe("Type Forms", func() {
	Describe("StructForm", func() {
		It("finds field by name", func() {
			form := resolution.StructForm{
				Fields: []resolution.Field{
					{Name: "id", Type: resolution.TypeRef{Name: "uuid"}},
					{Name: "name", Type: resolution.TypeRef{Name: "string"}},
					{Name: "age", Type: resolution.TypeRef{Name: "int32"}},
				},
			}
			field, ok := form.Field("name")
			Expect(ok).To(BeTrue())
			Expect(field.Name).To(Equal("name"))
			Expect(field.Type.Name).To(Equal("string"))
		})

		It("returns false for non-existent field", func() {
			form := resolution.StructForm{
				Fields: []resolution.Field{{Name: "id"}},
			}
			_, ok := form.Field("nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("detects generic structs", func() {
			form := resolution.StructForm{
				TypeParams: []resolution.TypeParam{{Name: "T"}},
			}
			Expect(form.IsGeneric()).To(BeTrue())
		})

		It("detects non-generic structs", func() {
			form := resolution.StructForm{}
			Expect(form.IsGeneric()).To(BeFalse())
		})

		It("finds type parameter by name", func() {
			form := resolution.StructForm{
				TypeParams: []resolution.TypeParam{
					{Name: "T"},
					{Name: "U", Constraint: &resolution.TypeRef{Name: "Comparable"}},
				},
			}
			tp, ok := form.TypeParam("U")
			Expect(ok).To(BeTrue())
			Expect(tp.Name).To(Equal("U"))
			Expect(tp.Constraint).NotTo(BeNil())
		})

		It("returns false for non-existent type parameter", func() {
			form := resolution.StructForm{
				TypeParams: []resolution.TypeParam{{Name: "T"}},
			}
			_, ok := form.TypeParam("V")
			Expect(ok).To(BeFalse())
		})

		It("checks if field is omitted", func() {
			form := resolution.StructForm{
				OmittedFields: []string{"internal", "private"},
			}
			Expect(form.IsFieldOmitted("internal")).To(BeTrue())
			Expect(form.IsFieldOmitted("private")).To(BeTrue())
			Expect(form.IsFieldOmitted("public")).To(BeFalse())
		})
	})

	Describe("AliasForm", func() {
		It("detects generic aliases", func() {
			form := resolution.AliasForm{
				TypeParams: []resolution.TypeParam{{Name: "T"}},
				Target:     resolution.TypeRef{Name: "Array", TypeArgs: []resolution.TypeRef{{Name: "T"}}},
			}
			Expect(form.IsGeneric()).To(BeTrue())
		})

		It("finds type parameter by name", func() {
			form := resolution.AliasForm{
				TypeParams: []resolution.TypeParam{{Name: "K"}, {Name: "V"}},
			}
			tp, ok := form.TypeParam("V")
			Expect(ok).To(BeTrue())
			Expect(tp.Name).To(Equal("V"))
		})
	})

	Describe("TypeRef", func() {
		It("detects type parameters", func() {
			ref := resolution.TypeRef{
				TypeParam: &resolution.TypeParam{Name: "T"},
			}
			Expect(ref.IsTypeParam()).To(BeTrue())
		})

		It("detects non-type-parameters", func() {
			ref := resolution.TypeRef{Name: "string"}
			Expect(ref.IsTypeParam()).To(BeFalse())
		})

		It("resolves type from table", func() {
			table := resolution.NewTable()
			Expect(table.Add(resolution.Type{
				Name:          "User",
				QualifiedName: "auth.User",
				Form:          resolution.StructForm{},
			})).To(Succeed())

			ref := resolution.TypeRef{Name: "auth.User"}
			typ, ok := ref.Resolve(table)
			Expect(ok).To(BeTrue())
			Expect(typ.Name).To(Equal("User"))
		})

		It("returns false when type not found", func() {
			table := resolution.NewTable()
			ref := resolution.TypeRef{Name: "nonexistent.Type"}
			_, ok := ref.Resolve(table)
			Expect(ok).To(BeFalse())
		})

		It("MustResolve panics when type not found", func() {
			table := resolution.NewTable()
			ref := resolution.TypeRef{Name: "nonexistent.Type"}
			Expect(func() {
				ref.MustResolve(table)
			}).To(Panic())
		})
	})

	Describe("TypeParam", func() {
		It("detects default value presence", func() {
			withDefault := resolution.TypeParam{
				Name:    "T",
				Default: &resolution.TypeRef{Name: "string"},
			}
			Expect(withDefault.HasDefault()).To(BeTrue())

			withoutDefault := resolution.TypeParam{Name: "T"}
			Expect(withoutDefault.HasDefault()).To(BeFalse())
		})
	})

	Describe("EnumValue", func() {
		It("extracts string value", func() {
			val := resolution.EnumValue{Name: "ACTIVE", Value: "active"}
			Expect(val.StringValue()).To(Equal("active"))
		})

		It("returns empty string for non-string value", func() {
			val := resolution.EnumValue{Name: "ONE", Value: int64(1)}
			Expect(val.StringValue()).To(Equal(""))
		})

		It("extracts int value", func() {
			val := resolution.EnumValue{Name: "ONE", Value: int64(42)}
			Expect(val.IntValue()).To(Equal(int64(42)))
		})

		It("returns zero for non-int value", func() {
			val := resolution.EnumValue{Name: "ACTIVE", Value: "active"}
			Expect(val.IntValue()).To(Equal(int64(0)))
		})
	})
})

var _ = Describe("Domain", func() {
	Describe("Merge", func() {
		It("merges expressions from parent and child", func() {
			parent := resolution.Domain{
				Name: "validation",
				Expressions: resolution.Expressions{
					{Name: "required"},
					{Name: "min", Values: []resolution.ExpressionValue{{IntValue: 0}}},
				},
			}
			child := resolution.Domain{
				Name: "validation",
				Expressions: resolution.Expressions{
					{Name: "min", Values: []resolution.ExpressionValue{{IntValue: 5}}},
					{Name: "max", Values: []resolution.ExpressionValue{{IntValue: 100}}},
				},
			}
			merged := child.Merge(parent)
			Expect(merged.Name).To(Equal("validation"))
			Expect(merged.Expressions).To(HaveLen(3))
		})

		It("child expressions override parent", func() {
			parent := resolution.Domain{
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "old/path"}}},
				},
			}
			child := resolution.Domain{
				Name: "go",
				Expressions: resolution.Expressions{
					{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "new/path"}}},
				},
			}
			merged := child.Merge(parent)
			expr, ok := merged.Expressions.Find("output")
			Expect(ok).To(BeTrue())
			Expect(expr.Values[0].StringValue).To(Equal("new/path"))
		})
	})

	Describe("Expressions.Find", func() {
		It("finds expression by name", func() {
			exprs := resolution.Expressions{
				{Name: "required"},
				{Name: "default", Values: []resolution.ExpressionValue{{StringValue: "test"}}},
				{Name: "min_length", Values: []resolution.ExpressionValue{{IntValue: 5}}},
			}
			expr, ok := exprs.Find("default")
			Expect(ok).To(BeTrue())
			Expect(expr.Values[0].StringValue).To(Equal("test"))
		})

		It("returns false for non-existent expression", func() {
			exprs := resolution.Expressions{{Name: "required"}}
			_, ok := exprs.Find("nonexistent")
			Expect(ok).To(BeFalse())
		})
	})
})

var _ = Describe("Primitive Helpers", func() {
	DescribeTable("IsPrimitive",
		func(name string, expected bool) {
			Expect(resolution.IsPrimitive(name)).To(Equal(expected))
		},
		Entry("string", "string", true),
		Entry("uuid", "uuid", true),
		Entry("int32", "int32", true),
		Entry("float64", "float64", true),
		Entry("bool", "bool", true),
		Entry("json", "json", true),
		Entry("bytes", "bytes", true),
		Entry("User", "User", false),
		Entry("Array", "Array", false),
	)

	DescribeTable("IsStringPrimitive",
		func(name string, expected bool) {
			Expect(resolution.IsStringPrimitive(name)).To(Equal(expected))
		},
		Entry("string", "string", true),
		Entry("uuid", "uuid", true),
		Entry("int32", "int32", false),
	)

	DescribeTable("IsNumberPrimitive",
		func(name string, expected bool) {
			Expect(resolution.IsNumberPrimitive(name)).To(Equal(expected))
		},
		Entry("int8", "int8", true),
		Entry("uint64", "uint64", true),
		Entry("float32", "float32", true),
		Entry("string", "string", false),
	)
})

var _ = Describe("UnifiedFields", func() {
	var table *resolution.Table

	BeforeEach(func() {
		table = resolution.NewTable()
	})

	It("returns fields directly for struct without extends", func() {
		typ := resolution.Type{
			Name:          "Simple",
			QualifiedName: "ns.Simple",
			Form: resolution.StructForm{
				Fields: []resolution.Field{
					{Name: "id", Type: resolution.TypeRef{Name: "uuid"}},
					{Name: "name", Type: resolution.TypeRef{Name: "string"}},
				},
			},
		}
		fields := resolution.UnifiedFields(typ, table)
		Expect(fields).To(HaveLen(2))
		Expect(fields[0].Name).To(Equal("id"))
		Expect(fields[1].Name).To(Equal("name"))
	})

	It("returns nil for non-struct types", func() {
		typ := resolution.Type{
			Name:          "Status",
			QualifiedName: "ns.Status",
			Form:          resolution.EnumForm{},
		}
		fields := resolution.UnifiedFields(typ, table)
		Expect(fields).To(BeNil())
	})

	It("includes parent fields", func() {
		parent := resolution.Type{
			Name:          "Parent",
			QualifiedName: "ns.Parent",
			Namespace:     "ns",
			Form: resolution.StructForm{
				Fields: []resolution.Field{
					{Name: "id", Type: resolution.TypeRef{Name: "uuid"}},
				},
			},
		}
		child := resolution.Type{
			Name:          "Child",
			QualifiedName: "ns.Child",
			Namespace:     "ns",
			Form: resolution.StructForm{
				Extends: []resolution.TypeRef{{Name: "ns.Parent"}},
				Fields: []resolution.Field{
					{Name: "name", Type: resolution.TypeRef{Name: "string"}},
				},
			},
		}
		Expect(table.Add(parent)).To(Succeed())
		Expect(table.Add(child)).To(Succeed())

		fields := resolution.UnifiedFields(child, table)
		Expect(fields).To(HaveLen(2))
		names := []string{fields[0].Name, fields[1].Name}
		Expect(names).To(ContainElements("id", "name"))
	})

	It("child fields override parent fields", func() {
		parent := resolution.Type{
			Name:          "Parent",
			QualifiedName: "ns.Parent",
			Namespace:     "ns",
			Form: resolution.StructForm{
				Fields: []resolution.Field{
					{Name: "name", Type: resolution.TypeRef{Name: "string"}},
				},
			},
		}
		child := resolution.Type{
			Name:          "Child",
			QualifiedName: "ns.Child",
			Namespace:     "ns",
			Form: resolution.StructForm{
				Extends: []resolution.TypeRef{{Name: "ns.Parent"}},
				Fields: []resolution.Field{
					{Name: "name", Type: resolution.TypeRef{Name: "uuid"}}, // override with different type
				},
			},
		}
		Expect(table.Add(parent)).To(Succeed())
		Expect(table.Add(child)).To(Succeed())

		fields := resolution.UnifiedFields(child, table)
		Expect(fields).To(HaveLen(1))
		Expect(fields[0].Name).To(Equal("name"))
		Expect(fields[0].Type.Name).To(Equal("uuid"))
	})

	It("respects omitted fields from parent", func() {
		parent := resolution.Type{
			Name:          "Parent",
			QualifiedName: "ns.Parent",
			Namespace:     "ns",
			Form: resolution.StructForm{
				Fields: []resolution.Field{
					{Name: "id", Type: resolution.TypeRef{Name: "uuid"}},
					{Name: "internal", Type: resolution.TypeRef{Name: "string"}},
				},
			},
		}
		child := resolution.Type{
			Name:          "Child",
			QualifiedName: "ns.Child",
			Namespace:     "ns",
			Form: resolution.StructForm{
				Extends:       []resolution.TypeRef{{Name: "ns.Parent"}},
				OmittedFields: []string{"internal"},
				Fields: []resolution.Field{
					{Name: "name", Type: resolution.TypeRef{Name: "string"}},
				},
			},
		}
		Expect(table.Add(parent)).To(Succeed())
		Expect(table.Add(child)).To(Succeed())

		fields := resolution.UnifiedFields(child, table)
		names := make([]string, len(fields))
		for i, f := range fields {
			names[i] = f.Name
		}
		Expect(names).To(ContainElements("id", "name"))
		Expect(names).NotTo(ContainElement("internal"))
	})
})

var _ = Describe("SubstituteTypeRef", func() {
	It("substitutes type parameter with concrete type", func() {
		ref := resolution.TypeRef{
			TypeParam: &resolution.TypeParam{Name: "T"},
		}
		typeArgMap := map[string]resolution.TypeRef{
			"T": {Name: "string"},
		}
		result := resolution.SubstituteTypeRef(ref, typeArgMap)
		Expect(result.Name).To(Equal("string"))
		Expect(result.IsTypeParam()).To(BeFalse())
	})

	It("returns original ref when type param not in map", func() {
		ref := resolution.TypeRef{
			TypeParam: &resolution.TypeParam{Name: "U"},
		}
		typeArgMap := map[string]resolution.TypeRef{
			"T": {Name: "string"},
		}
		result := resolution.SubstituteTypeRef(ref, typeArgMap)
		Expect(result.IsTypeParam()).To(BeTrue())
		Expect(result.TypeParam.Name).To(Equal("U"))
	})

	It("returns non-type-param refs unchanged when no type args", func() {
		ref := resolution.TypeRef{Name: "string"}
		result := resolution.SubstituteTypeRef(ref, map[string]resolution.TypeRef{})
		Expect(result.Name).To(Equal("string"))
	})

	It("recursively substitutes type args", func() {
		ref := resolution.TypeRef{
			Name: "Array",
			TypeArgs: []resolution.TypeRef{
				{TypeParam: &resolution.TypeParam{Name: "T"}},
			},
		}
		typeArgMap := map[string]resolution.TypeRef{
			"T": {Name: "int32"},
		}
		result := resolution.SubstituteTypeRef(ref, typeArgMap)
		Expect(result.Name).To(Equal("Array"))
		Expect(result.TypeArgs).To(HaveLen(1))
		Expect(result.TypeArgs[0].Name).To(Equal("int32"))
	})

	It("handles nested generic substitution", func() {
		// Map<K, Array<V>>
		ref := resolution.TypeRef{
			Name: "Map",
			TypeArgs: []resolution.TypeRef{
				{TypeParam: &resolution.TypeParam{Name: "K"}},
				{
					Name: "Array",
					TypeArgs: []resolution.TypeRef{
						{TypeParam: &resolution.TypeParam{Name: "V"}},
					},
				},
			},
		}
		typeArgMap := map[string]resolution.TypeRef{
			"K": {Name: "string"},
			"V": {Name: "uuid"},
		}
		result := resolution.SubstituteTypeRef(ref, typeArgMap)
		Expect(result.Name).To(Equal("Map"))
		Expect(result.TypeArgs[0].Name).To(Equal("string"))
		Expect(result.TypeArgs[1].Name).To(Equal("Array"))
		Expect(result.TypeArgs[1].TypeArgs[0].Name).To(Equal("uuid"))
	})
})
