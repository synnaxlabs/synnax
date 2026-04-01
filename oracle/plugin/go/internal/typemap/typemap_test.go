// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package typemap_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("PrimitiveGoType", func() {
	DescribeTable("should map supported primitives",
		func(input, expected string) {
			goType := MustBeOk(typemap.PrimitiveGoType(input))
			Expect(goType).To(Equal(expected))
		},
		Entry("string", "string", "string"),
		Entry("bool", "bool", "bool"),
		Entry("int8", "int8", "int8"),
		Entry("int16", "int16", "int16"),
		Entry("int32", "int32", "int32"),
		Entry("int64", "int64", "int64"),
		Entry("uint8", "uint8", "uint8"),
		Entry("uint12 maps to uint16", "uint12", "uint16"),
		Entry("uint16", "uint16", "uint16"),
		Entry("uint20 maps to uint32", "uint20", "uint32"),
		Entry("uint32", "uint32", "uint32"),
		Entry("uint64", "uint64", "uint64"),
		Entry("float32", "float32", "float32"),
		Entry("float64", "float64", "float64"),
		Entry("uuid", "uuid", "uuid.UUID"),
		Entry("bytes", "bytes", "[]byte"),
		Entry("record", "record", "interface{}"),
		Entry("any", "any", "interface{}"),
	)

	It("should return false for unsupported primitives", func() {
		_, ok := typemap.PrimitiveGoType("unknown_type")
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("IsUUID", func() {
	It("should return true for uuid", func() {
		Expect(typemap.IsUUID("uuid")).To(BeTrue())
	})

	It("should return false for other types", func() {
		Expect(typemap.IsUUID("string")).To(BeFalse())
		Expect(typemap.IsUUID("int32")).To(BeFalse())
	})
})

var _ = Describe("UnwrapType", func() {
	It("should return the type itself for a struct", func() {
		table := resolution.NewTable()
		structType := resolution.Type{
			Name:          "User",
			QualifiedName: "test.User",
			Form:          resolution.StructForm{},
		}
		Expect(table.Add(structType)).To(Succeed())
		result := typemap.UnwrapType(structType, table)
		Expect(result.QualifiedName).To(Equal("test.User"))
	})

	It("should resolve through an alias", func() {
		table := resolution.NewTable()
		baseType := resolution.Type{
			Name:          "Base",
			QualifiedName: "test.Base",
			Form:          resolution.StructForm{},
		}
		aliasType := resolution.Type{
			Name:          "Alias",
			QualifiedName: "test.Alias",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "test.Base"},
			},
		}
		Expect(table.Add(baseType)).To(Succeed())
		Expect(table.Add(aliasType)).To(Succeed())
		result := typemap.UnwrapType(aliasType, table)
		Expect(result.QualifiedName).To(Equal("test.Base"))
	})

	It("should resolve through a distinct type", func() {
		table := resolution.NewTable()
		baseType := resolution.Type{
			Name:          "Base",
			QualifiedName: "test.Base",
			Form:          resolution.PrimitiveForm{Name: "string"},
		}
		distinctType := resolution.Type{
			Name:          "MyString",
			QualifiedName: "test.MyString",
			Form: resolution.DistinctForm{
				Base: resolution.TypeRef{Name: "test.Base"},
			},
		}
		Expect(table.Add(baseType)).To(Succeed())
		Expect(table.Add(distinctType)).To(Succeed())
		result := typemap.UnwrapType(distinctType, table)
		Expect(result.QualifiedName).To(Equal("test.Base"))
	})

	It("should resolve through chained aliases", func() {
		table := resolution.NewTable()
		base := resolution.Type{
			Name:          "Base",
			QualifiedName: "test.Base",
			Form:          resolution.StructForm{},
		}
		mid := resolution.Type{
			Name:          "Mid",
			QualifiedName: "test.Mid",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "test.Base"},
			},
		}
		top := resolution.Type{
			Name:          "Top",
			QualifiedName: "test.Top",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "test.Mid"},
			},
		}
		Expect(table.Add(base)).To(Succeed())
		Expect(table.Add(mid)).To(Succeed())
		Expect(table.Add(top)).To(Succeed())
		result := typemap.UnwrapType(top, table)
		Expect(result.QualifiedName).To(Equal("test.Base"))
	})

	It("should return the type when alias target is unresolvable", func() {
		table := resolution.NewTable()
		aliasType := resolution.Type{
			Name:          "Broken",
			QualifiedName: "test.Broken",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "nonexistent"},
			},
		}
		Expect(table.Add(aliasType)).To(Succeed())
		result := typemap.UnwrapType(aliasType, table)
		Expect(result.QualifiedName).To(Equal("test.Broken"))
	})
})

var _ = Describe("UnwrapTypeRef", func() {
	It("should preserve type args for a non-alias type", func() {
		table := resolution.NewTable()
		structType := resolution.Type{
			Name:          "User",
			QualifiedName: "test.User",
			Form:          resolution.StructForm{},
		}
		Expect(table.Add(structType)).To(Succeed())
		ref := resolution.TypeRef{
			Name:     "test.User",
			TypeArgs: []resolution.TypeRef{{Name: "string"}},
		}
		result, args := typemap.UnwrapTypeRef(structType, ref, table)
		Expect(result.QualifiedName).To(Equal("test.User"))
		Expect(args).To(HaveLen(1))
	})

	It("should propagate type args through an alias", func() {
		table := resolution.NewTable()
		base := resolution.Type{
			Name:          "Base",
			QualifiedName: "test.Base",
			Form:          resolution.StructForm{},
		}
		alias := resolution.Type{
			Name:          "Alias",
			QualifiedName: "test.Alias",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{
					Name:     "test.Base",
					TypeArgs: []resolution.TypeRef{{Name: "int32"}},
				},
			},
		}
		Expect(table.Add(base)).To(Succeed())
		Expect(table.Add(alias)).To(Succeed())
		ref := resolution.TypeRef{Name: "test.Alias"}
		result, args := typemap.UnwrapTypeRef(alias, ref, table)
		Expect(result.QualifiedName).To(Equal("test.Base"))
		Expect(args).To(HaveLen(1))
		Expect(args[0].Name).To(Equal("int32"))
	})

	It("should propagate type args through a distinct type", func() {
		table := resolution.NewTable()
		base := resolution.Type{
			Name:          "Base",
			QualifiedName: "test.Base",
			Form:          resolution.StructForm{},
		}
		distinct := resolution.Type{
			Name:          "Distinct",
			QualifiedName: "test.Distinct",
			Form: resolution.DistinctForm{
				Base: resolution.TypeRef{
					Name:     "test.Base",
					TypeArgs: []resolution.TypeRef{{Name: "string"}},
				},
			},
		}
		Expect(table.Add(base)).To(Succeed())
		Expect(table.Add(distinct)).To(Succeed())
		ref := resolution.TypeRef{Name: "test.Distinct"}
		result, args := typemap.UnwrapTypeRef(distinct, ref, table)
		Expect(result.QualifiedName).To(Equal("test.Base"))
		Expect(args).To(HaveLen(1))
		Expect(args[0].Name).To(Equal("string"))
	})

	It("should return the type when distinct base is unresolvable", func() {
		table := resolution.NewTable()
		distinct := resolution.Type{
			Name:          "Broken",
			QualifiedName: "test.Broken",
			Form: resolution.DistinctForm{
				Base: resolution.TypeRef{Name: "nonexistent"},
			},
		}
		Expect(table.Add(distinct)).To(Succeed())
		ref := resolution.TypeRef{Name: "test.Broken"}
		result, args := typemap.UnwrapTypeRef(distinct, ref, table)
		Expect(result.QualifiedName).To(Equal("test.Broken"))
		Expect(args).To(BeEmpty())
	})

	It("should return the type when alias target is unresolvable", func() {
		table := resolution.NewTable()
		alias := resolution.Type{
			Name:          "Broken",
			QualifiedName: "test.Broken",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "nonexistent"},
			},
		}
		Expect(table.Add(alias)).To(Succeed())
		ref := resolution.TypeRef{
			Name:     "test.Broken",
			TypeArgs: []resolution.TypeRef{{Name: "int32"}},
		}
		result, args := typemap.UnwrapTypeRef(alias, ref, table)
		Expect(result.QualifiedName).To(Equal("test.Broken"))
		Expect(args).To(HaveLen(1))
	})
})

var _ = Describe("ResolveLeafPrimitive", func() {
	goTypeName := func(t resolution.Type) (string, error) {
		return t.Name, nil
	}

	It("should resolve a primitive directly", func() {
		table := resolution.NewTable()
		primType := resolution.Type{
			Name:          "string",
			QualifiedName: "string",
			Form:          resolution.PrimitiveForm{Name: "string"},
		}
		prim, cast := MustSucceed2(typemap.ResolveLeafPrimitive(primType, table, goTypeName))
		Expect(prim).To(Equal("string"))
		Expect(cast).To(BeEmpty())
	})

	It("should resolve a distinct type with a cast", func() {
		table := resolution.NewTable()
		baseType := table.MustGet("string")
		distinctType := resolution.Type{
			Name:          "MyString",
			QualifiedName: "test.MyString",
			Form: resolution.DistinctForm{
				Base: resolution.TypeRef{Name: "string"},
			},
		}
		_ = baseType
		Expect(table.Add(distinctType)).To(Succeed())
		prim, cast := MustSucceed2(typemap.ResolveLeafPrimitive(distinctType, table, goTypeName))
		Expect(prim).To(Equal("string"))
		Expect(cast).To(Equal("MyString"))
	})

	It("should resolve an integer enum", func() {
		table := resolution.NewTable()
		enumType := resolution.Type{
			Name:          "Status",
			QualifiedName: "test.Status",
			Form:          resolution.EnumForm{IsIntEnum: true},
		}
		Expect(table.Add(enumType)).To(Succeed())
		prim, cast := MustSucceed2(typemap.ResolveLeafPrimitive(enumType, table, goTypeName))
		Expect(prim).To(Equal("int64"))
		Expect(cast).To(Equal("Status"))
	})

	It("should resolve a string enum", func() {
		table := resolution.NewTable()
		enumType := resolution.Type{
			Name:          "Priority",
			QualifiedName: "test.Priority",
			Form:          resolution.EnumForm{IsIntEnum: false},
		}
		Expect(table.Add(enumType)).To(Succeed())
		prim, cast := MustSucceed2(typemap.ResolveLeafPrimitive(enumType, table, goTypeName))
		Expect(prim).To(Equal("string"))
		Expect(cast).To(Equal("Priority"))
	})

	It("should resolve an alias to a primitive without a cast", func() {
		table := resolution.NewTable()
		aliasType := resolution.Type{
			Name:          "Name",
			QualifiedName: "test.Name",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "string"},
			},
		}
		Expect(table.Add(aliasType)).To(Succeed())
		prim, cast := MustSucceed2(typemap.ResolveLeafPrimitive(aliasType, table, goTypeName))
		Expect(prim).To(Equal("string"))
		Expect(cast).To(BeEmpty())
	})

	It("should error on unsupported type forms", func() {
		table := resolution.NewTable()
		structType := resolution.Type{
			Name:          "User",
			QualifiedName: "test.User",
			Form:          resolution.StructForm{},
		}
		Expect(table.Add(structType)).To(Succeed())
		_, _, err := typemap.ResolveLeafPrimitive(structType, table, goTypeName)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("unsupported type form")))
	})

	It("should error when distinct base is unresolvable", func() {
		table := resolution.NewTable()
		distinct := resolution.Type{
			Name:          "Broken",
			QualifiedName: "test.Broken",
			Form: resolution.DistinctForm{
				Base: resolution.TypeRef{Name: "nonexistent"},
			},
		}
		Expect(table.Add(distinct)).To(Succeed())
		_, _, err := typemap.ResolveLeafPrimitive(distinct, table, goTypeName)
		Expect(err).To(MatchError(ContainSubstring("cannot resolve distinct base")))
	})

	It("should error when alias target is unresolvable", func() {
		table := resolution.NewTable()
		alias := resolution.Type{
			Name:          "Broken",
			QualifiedName: "test.Broken",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "nonexistent"},
			},
		}
		Expect(table.Add(alias)).To(Succeed())
		_, _, err := typemap.ResolveLeafPrimitive(alias, table, goTypeName)
		Expect(err).To(MatchError(ContainSubstring("cannot resolve alias target")))
	})

	It("should resolve an alias to a distinct type with a cast", func() {
		table := resolution.NewTable()
		distinct := resolution.Type{
			Name:          "UserID",
			QualifiedName: "test.UserID",
			Form: resolution.DistinctForm{
				Base: resolution.TypeRef{Name: "uuid"},
			},
		}
		alias := resolution.Type{
			Name:          "Key",
			QualifiedName: "test.Key",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "test.UserID"},
			},
		}
		Expect(table.Add(distinct)).To(Succeed())
		Expect(table.Add(alias)).To(Succeed())
		prim, cast := MustSucceed2(typemap.ResolveLeafPrimitive(alias, table, goTypeName))
		Expect(prim).To(Equal("uuid"))
		Expect(cast).To(Equal("Key"))
	})
})

var _ = Describe("ResolveGoSliceElemType", func() {
	goTypeName := func(t resolution.Type) (string, error) {
		return t.Name, nil
	}

	It("should resolve a concrete type directly", func() {
		table := resolution.NewTable()
		structType := resolution.Type{
			Name:          "User",
			QualifiedName: "test.User",
			Form:          resolution.StructForm{},
		}
		Expect(table.Add(structType)).To(Succeed())
		result := MustSucceed(typemap.ResolveGoSliceElemType(structType, table, goTypeName))
		Expect(result).To(Equal("User"))
	})

	It("should resolve through an alias to a struct", func() {
		table := resolution.NewTable()
		base := resolution.Type{
			Name:          "Base",
			QualifiedName: "test.Base",
			Form:          resolution.StructForm{},
		}
		alias := resolution.Type{
			Name:          "Alias",
			QualifiedName: "test.Alias",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "test.Base"},
			},
		}
		Expect(table.Add(base)).To(Succeed())
		Expect(table.Add(alias)).To(Succeed())
		result := MustSucceed(typemap.ResolveGoSliceElemType(alias, table, goTypeName))
		Expect(result).To(Equal("Base"))
	})

	It("should resolve through a distinct type to a struct", func() {
		table := resolution.NewTable()
		base := resolution.Type{
			Name:          "Base",
			QualifiedName: "test.Base",
			Form:          resolution.StructForm{},
		}
		distinct := resolution.Type{
			Name:          "Distinct",
			QualifiedName: "test.Distinct",
			Form: resolution.DistinctForm{
				Base: resolution.TypeRef{Name: "test.Base"},
			},
		}
		Expect(table.Add(base)).To(Succeed())
		Expect(table.Add(distinct)).To(Succeed())
		result := MustSucceed(typemap.ResolveGoSliceElemType(distinct, table, goTypeName))
		Expect(result).To(Equal("Base"))
	})

	It("should error when alias target is unresolvable", func() {
		table := resolution.NewTable()
		alias := resolution.Type{
			Name:          "Broken",
			QualifiedName: "test.Broken",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{Name: "nonexistent"},
			},
		}
		Expect(table.Add(alias)).To(Succeed())
		_, err := typemap.ResolveGoSliceElemType(alias, table, goTypeName)
		Expect(err).To(MatchError(ContainSubstring("cannot resolve type")))
	})

	It("should handle nested arrays", func() {
		table := resolution.NewTable()
		innerAlias := resolution.Type{
			Name:          "StringArray",
			QualifiedName: "test.StringArray",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{
					Name:     "Array",
					TypeArgs: []resolution.TypeRef{{Name: "string"}},
				},
			},
		}
		Expect(table.Add(innerAlias)).To(Succeed())
		result := MustSucceed(typemap.ResolveGoSliceElemType(innerAlias, table, goTypeName))
		Expect(result).To(Equal("[]string"))
	})
})
