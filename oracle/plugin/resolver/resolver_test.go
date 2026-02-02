// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolver_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/primitives"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
)

func TestResolver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resolver Suite")
}

// MockImportAdder implements ImportAdder for testing.
type MockImportAdder struct {
	Imports []MockImport
}

type MockImport struct {
	Category string
	Path     string
	Alias    string
}

func (m *MockImportAdder) AddImport(category, path, alias string) {
	m.Imports = append(m.Imports, MockImport{Category: category, Path: path, Alias: alias})
}

// MockTypeFormatter implements TypeFormatter for testing (Go-like syntax).
type MockTypeFormatter struct{}

func (m *MockTypeFormatter) FormatQualified(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return qualifier + "." + typeName
}

func (m *MockTypeFormatter) FormatGeneric(baseName string, typeArgs []string) string {
	if len(typeArgs) == 0 {
		return baseName
	}
	result := baseName + "["
	for i, arg := range typeArgs {
		if i > 0 {
			result += ", "
		}
		result += arg
	}
	return result + "]"
}

func (m *MockTypeFormatter) FormatArray(elemType string) string {
	return "[]" + elemType
}

func (m *MockTypeFormatter) FormatFixedArray(elemType string, size int64) string {
	return fmt.Sprintf("[%d]%s", size, elemType)
}

func (m *MockTypeFormatter) FormatMap(keyType, valType string) string {
	return "map[" + keyType + "]" + valType
}

func (m *MockTypeFormatter) FallbackType() string {
	return "any"
}

// MockImportResolver implements ImportResolver for testing.
type MockImportResolver struct {
	ImportPath   string
	Qualifier    string
	ShouldImport bool
}

func (m *MockImportResolver) ResolveImport(outputPath string, ctx *resolver.Context) (string, string, bool) {
	return m.ImportPath, m.Qualifier, m.ShouldImport
}

// MockPrimitiveMapper implements primitives.Mapper for testing.
type MockPrimitiveMapper struct{}

func (m *MockPrimitiveMapper) Map(name string) primitives.Mapping {
	// Return simple Go-like mappings for testing
	switch name {
	case "string":
		return primitives.Mapping{TargetType: "string"}
	case "int32":
		return primitives.Mapping{TargetType: "int32"}
	case "bool":
		return primitives.Mapping{TargetType: "bool"}
	case "uuid":
		return primitives.Mapping{
			TargetType: "uuid.UUID",
			Imports:    []primitives.Import{{Category: "external", Path: "github.com/google/uuid"}},
		}
	default:
		return primitives.Mapping{TargetType: "any"}
	}
}

var _ = Describe("Resolver", func() {
	var (
		r     *resolver.Resolver
		adder *MockImportAdder
		table *resolution.Table
		ctx   *resolver.Context
	)

	BeforeEach(func() {
		adder = &MockImportAdder{}
		table = resolution.NewTable()
		ctx = &resolver.Context{
			Table:      table,
			OutputPath: "pkg/types.go",
			Namespace:  "test",
			RepoRoot:   "/repo",
			DomainName: "go",
		}
		r = &resolver.Resolver{
			Formatter:       &MockTypeFormatter{},
			ImportResolver:  &MockImportResolver{ShouldImport: false},
			ImportAdder:     adder,
			PrimitiveMapper: &MockPrimitiveMapper{},
		}
	})

	Describe("ResolveTypeRef", func() {
		Describe("Type Parameters", func() {
			It("Should resolve type parameters by name", func() {
				typeRef := resolution.TypeRef{
					Name: "T",
					TypeParam: &resolution.TypeParam{
						Name: "T",
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("T"))
			})

			It("Should substitute defaulted type params when enabled", func() {
				ctx.SubstituteDefaultedTypeParams = true
				typeRef := resolution.TypeRef{
					TypeParam: &resolution.TypeParam{
						Name:    "V",
						Default: &resolution.TypeRef{Name: "string"},
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("string"))
			})

			It("Should not substitute defaulted type params when disabled", func() {
				ctx.SubstituteDefaultedTypeParams = false
				typeRef := resolution.TypeRef{
					TypeParam: &resolution.TypeParam{
						Name:    "V",
						Default: &resolution.TypeRef{Name: "string"},
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("V"))
			})
		})

		Describe("Array Types", func() {
			It("Should resolve Array<T> types", func() {
				typeRef := resolution.TypeRef{
					Name: "Array",
					TypeArgs: []resolution.TypeRef{
						{Name: "string"},
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("[]string"))
			})

			It("Should resolve nested Array<Array<T>> types", func() {
				typeRef := resolution.TypeRef{
					Name: "Array",
					TypeArgs: []resolution.TypeRef{
						{
							Name: "Array",
							TypeArgs: []resolution.TypeRef{
								{Name: "int32"},
							},
						},
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("[][]int32"))
			})

			It("Should resolve fixed-size Array[N] types", func() {
				size := int64(4)
				typeRef := resolution.TypeRef{
					Name: "Array",
					TypeArgs: []resolution.TypeRef{
						{Name: "int32"},
					},
					ArraySize: &size,
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("[4]int32"))
			})
		})

		Describe("Map Types", func() {
			It("Should resolve Map<K, V> types", func() {
				typeRef := resolution.TypeRef{
					Name: "Map",
					TypeArgs: []resolution.TypeRef{
						{Name: "string"},
						{Name: "int32"},
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("map[string]int32"))
			})
		})

		Describe("Primitive Types", func() {
			It("Should resolve primitive types with imports", func() {
				typeRef := resolution.TypeRef{Name: "uuid"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("uuid.UUID"))
				Expect(adder.Imports).To(HaveLen(1))
				Expect(adder.Imports[0].Path).To(Equal("github.com/google/uuid"))
			})

			It("Should resolve simple primitives without imports", func() {
				typeRef := resolution.TypeRef{Name: "string"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("string"))
				Expect(adder.Imports).To(BeEmpty())
			})
		})

		Describe("Named Types", func() {
			BeforeEach(func() {
				// Add a struct type to the table
				Expect(table.Add(resolution.Type{
					Name:          "MyStruct",
					QualifiedName: "test.MyStruct",
					Namespace:     "test",
					Form:          resolution.StructForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
							},
						},
					},
				})).To(Succeed())
			})

			It("Should resolve struct types in the same namespace", func() {
				typeRef := resolution.TypeRef{
					Name: "test.MyStruct",
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("MyStruct"))
			})

			It("Should resolve struct types with type arguments", func() {
				// Add a generic struct
				Expect(table.Add(resolution.Type{
					Name:          "Container",
					QualifiedName: "test.Container",
					Namespace:     "test",
					Form: resolution.StructForm{
						TypeParams: []resolution.TypeParam{{Name: "T"}},
					},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
							},
						},
					},
				})).To(Succeed())

				typeRef := resolution.TypeRef{
					Name: "test.Container",
					TypeArgs: []resolution.TypeRef{
						{Name: "string"},
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("Container[string]"))
			})

			It("Should filter defaulted type params in struct when substitution enabled", func() {
				ctx.SubstituteDefaultedTypeParams = true
				Expect(table.Add(resolution.Type{
					Name:          "DefaultedStruct",
					QualifiedName: "test.DefaultedStruct",
					Namespace:     "test",
					Form: resolution.StructForm{
						TypeParams: []resolution.TypeParam{
							{Name: "T"},
							{Name: "V", Default: &resolution.TypeRef{Name: "string"}},
						},
					},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
							},
						},
					},
				})).To(Succeed())

				typeRef := resolution.TypeRef{
					Name: "test.DefaultedStruct",
					TypeArgs: []resolution.TypeRef{
						{Name: "int32"},
						{Name: "bool"},
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("DefaultedStruct[int32]"))
			})

			It("Should return fallback for struct with empty output path", func() {
				Expect(table.Add(resolution.Type{
					Name:          "NoOutputStruct",
					QualifiedName: "other.NoOutputStruct",
					Namespace:     "other",
					Form:          resolution.StructForm{},
					Domains:       map[string]resolution.Domain{},
				})).To(Succeed())

				typeRef := resolution.TypeRef{Name: "other.NoOutputStruct"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("any"))
			})
		})

		Describe("Cross-Namespace Types", func() {
			BeforeEach(func() {
				// Add a struct in a different namespace
				Expect(table.Add(resolution.Type{
					Name:          "ExternalType",
					QualifiedName: "other.ExternalType",
					Namespace:     "other",
					Form:          resolution.StructForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "external/types.go"}}},
							},
						},
					},
				})).To(Succeed())

				// Configure import resolver to return import info
				r.ImportResolver = &MockImportResolver{
					ImportPath:   "github.com/example/external",
					Qualifier:    "external",
					ShouldImport: true,
				}
			})

			It("Should add import and qualify cross-namespace types", func() {
				ctx.Namespace = "test"
				typeRef := resolution.TypeRef{
					Name: "other.ExternalType",
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("external.ExternalType"))
				Expect(adder.Imports).To(HaveLen(1))
				Expect(adder.Imports[0].Path).To(Equal("github.com/example/external"))
				Expect(adder.Imports[0].Alias).To(Equal("external"))
			})
		})

		Describe("Enum Types", func() {
			BeforeEach(func() {
				Expect(table.Add(resolution.Type{
					Name:          "Status",
					QualifiedName: "test.Status",
					Namespace:     "test",
					Form:          resolution.EnumForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
							},
						},
					},
				})).To(Succeed())
			})

			It("Should resolve enum types in the same namespace", func() {
				typeRef := resolution.TypeRef{
					Name: "test.Status",
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("Status"))
			})

			It("Should add import and qualify cross-namespace enums", func() {
				Expect(table.Add(resolution.Type{
					Name:          "ExternalEnum",
					QualifiedName: "other.ExternalEnum",
					Namespace:     "other",
					Form:          resolution.EnumForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "external/enums.go"}}},
							},
						},
					},
				})).To(Succeed())

				r.ImportResolver = &MockImportResolver{
					ImportPath:   "github.com/example/external",
					Qualifier:    "external",
					ShouldImport: true,
				}
				typeRef := resolution.TypeRef{Name: "other.ExternalEnum"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("external.ExternalEnum"))
				Expect(adder.Imports).To(HaveLen(1))
			})

			It("Should return fallback for enum with empty output path", func() {
				Expect(table.Add(resolution.Type{
					Name:          "NoOutputEnum",
					QualifiedName: "other.NoOutputEnum",
					Namespace:     "other",
					Form:          resolution.EnumForm{},
					Domains:       map[string]resolution.Domain{},
				})).To(Succeed())

				typeRef := resolution.TypeRef{Name: "other.NoOutputEnum"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("any"))
			})
		})

		Describe("Distinct Types", func() {
			BeforeEach(func() {
				Expect(table.Add(resolution.Type{
					Name:          "UserID",
					QualifiedName: "test.UserID",
					Namespace:     "test",
					Form:          resolution.DistinctForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
							},
						},
					},
				})).To(Succeed())
			})

			It("Should resolve distinct types", func() {
				typeRef := resolution.TypeRef{
					Name: "test.UserID",
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("UserID"))
			})

			It("Should add import and qualify cross-namespace distinct types", func() {
				Expect(table.Add(resolution.Type{
					Name:          "ExternalID",
					QualifiedName: "other.ExternalID",
					Namespace:     "other",
					Form:          resolution.DistinctForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "external/types.go"}}},
							},
						},
					},
				})).To(Succeed())

				r.ImportResolver = &MockImportResolver{
					ImportPath:   "github.com/example/external",
					Qualifier:    "external",
					ShouldImport: true,
				}
				typeRef := resolution.TypeRef{Name: "other.ExternalID"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("external.ExternalID"))
				Expect(adder.Imports).To(HaveLen(1))
			})

			It("Should return fallback for distinct with empty output path", func() {
				Expect(table.Add(resolution.Type{
					Name:          "NoOutputID",
					QualifiedName: "other.NoOutputID",
					Namespace:     "other",
					Form:          resolution.DistinctForm{},
					Domains:       map[string]resolution.Domain{},
				})).To(Succeed())

				typeRef := resolution.TypeRef{Name: "other.NoOutputID"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("any"))
			})
		})

		Describe("Alias Types", func() {
			BeforeEach(func() {
				Expect(table.Add(resolution.Type{
					Name:          "StringAlias",
					QualifiedName: "test.StringAlias",
					Namespace:     "test",
					Form:          resolution.AliasForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
							},
						},
					},
				})).To(Succeed())
			})

			It("Should resolve alias types by name (not expanding)", func() {
				typeRef := resolution.TypeRef{
					Name: "test.StringAlias",
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("StringAlias"))
			})

			It("Should resolve generic alias with type arguments", func() {
				Expect(table.Add(resolution.Type{
					Name:          "GenericAlias",
					QualifiedName: "test.GenericAlias",
					Namespace:     "test",
					Form: resolution.AliasForm{
						TypeParams: []resolution.TypeParam{{Name: "T"}},
					},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
							},
						},
					},
				})).To(Succeed())

				typeRef := resolution.TypeRef{
					Name:     "test.GenericAlias",
					TypeArgs: []resolution.TypeRef{{Name: "string"}},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("GenericAlias[string]"))
			})

			It("Should filter defaulted type params in alias when substitution enabled", func() {
				ctx.SubstituteDefaultedTypeParams = true
				Expect(table.Add(resolution.Type{
					Name:          "DefaultedAlias",
					QualifiedName: "test.DefaultedAlias",
					Namespace:     "test",
					Form: resolution.AliasForm{
						TypeParams: []resolution.TypeParam{
							{Name: "T"},
							{Name: "V", Default: &resolution.TypeRef{Name: "string"}},
						},
					},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
							},
						},
					},
				})).To(Succeed())

				typeRef := resolution.TypeRef{
					Name: "test.DefaultedAlias",
					TypeArgs: []resolution.TypeRef{
						{Name: "int32"},
						{Name: "bool"},
					},
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("DefaultedAlias[int32]"))
			})

			It("Should add import and qualify cross-namespace alias", func() {
				Expect(table.Add(resolution.Type{
					Name:          "ExternalAlias",
					QualifiedName: "other.ExternalAlias",
					Namespace:     "other",
					Form:          resolution.AliasForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "external/types.go"}}},
							},
						},
					},
				})).To(Succeed())

				r.ImportResolver = &MockImportResolver{
					ImportPath:   "github.com/example/external",
					Qualifier:    "external",
					ShouldImport: true,
				}
				typeRef := resolution.TypeRef{Name: "other.ExternalAlias"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("external.ExternalAlias"))
				Expect(adder.Imports).To(HaveLen(1))
			})

			It("Should return fallback for alias with empty output path", func() {
				Expect(table.Add(resolution.Type{
					Name:          "NoOutputAlias",
					QualifiedName: "other.NoOutputAlias",
					Namespace:     "other",
					Form:          resolution.AliasForm{},
					Domains:       map[string]resolution.Domain{},
				})).To(Succeed())

				typeRef := resolution.TypeRef{Name: "other.NoOutputAlias"}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("any"))
			})
		})

		Describe("Name Overrides", func() {
			BeforeEach(func() {
				Expect(table.Add(resolution.Type{
					Name:          "OriginalName",
					QualifiedName: "test.OriginalName",
					Namespace:     "test",
					Form:          resolution.StructForm{},
					Domains: map[string]resolution.Domain{
						"go": {
							Expressions: []resolution.Expression{
								{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
								{Name: "name", Values: []resolution.ExpressionValue{{StringValue: "CustomName"}}},
							},
						},
					},
				})).To(Succeed())
			})

			It("Should use language-specific name override", func() {
				typeRef := resolution.TypeRef{
					Name: "test.OriginalName",
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("CustomName"))
			})
		})

		Describe("Unresolvable Types", func() {
			It("Should return fallback type for unknown references", func() {
				typeRef := resolution.TypeRef{
					Name: "unknown.UnknownType",
				}
				result := r.ResolveTypeRef(typeRef, ctx)
				Expect(result).To(Equal("any"))
			})
		})
	})
})

var _ = Describe("Context", func() {
	var (
		table *resolution.Table
		ctx   *resolver.Context
	)

	BeforeEach(func() {
		table = resolution.NewTable()
		ctx = &resolver.Context{
			Table:      table,
			OutputPath: "pkg/types.go",
			Namespace:  "test",
			RepoRoot:   "/repo",
			DomainName: "go",
		}
	})

	Describe("IsSameOutput", func() {
		It("Should return true for same namespace and output path", func() {
			t := resolution.Type{
				Name:      "MyType",
				Namespace: "test",
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
						},
					},
				},
			}
			Expect(ctx.IsSameOutput(t)).To(BeTrue())
		})

		It("Should return false for different namespace", func() {
			t := resolution.Type{
				Name:      "MyType",
				Namespace: "other",
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
						},
					},
				},
			}
			Expect(ctx.IsSameOutput(t)).To(BeFalse())
		})

		It("Should return false for different output path", func() {
			t := resolution.Type{
				Name:      "MyType",
				Namespace: "test",
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "other/types.go"}}},
						},
					},
				},
			}
			Expect(ctx.IsSameOutput(t)).To(BeFalse())
		})

		It("Should return true for same namespace with no output path", func() {
			t := resolution.Type{
				Name:      "MyType",
				Namespace: "test",
				Domains:   map[string]resolution.Domain{},
			}
			Expect(ctx.IsSameOutput(t)).To(BeTrue())
		})
	})

	Describe("GetTypeName", func() {
		It("Should return original name when no override", func() {
			t := resolution.Type{
				Name:    "MyType",
				Domains: map[string]resolution.Domain{},
			}
			Expect(ctx.GetTypeName(t)).To(Equal("MyType"))
		})

		It("Should return override name when present", func() {
			t := resolution.Type{
				Name: "MyType",
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "name", Values: []resolution.ExpressionValue{{StringValue: "OverrideName"}}},
						},
					},
				},
			}
			Expect(ctx.GetTypeName(t)).To(Equal("OverrideName"))
		})

		It("Should ignore overrides for other domains", func() {
			t := resolution.Type{
				Name: "MyType",
				Domains: map[string]resolution.Domain{
					"py": {
						Expressions: []resolution.Expression{
							{Name: "name", Values: []resolution.ExpressionValue{{StringValue: "PythonName"}}},
						},
					},
				},
			}
			Expect(ctx.GetTypeName(t)).To(Equal("MyType"))
		})
	})

	Describe("GetOutputPath", func() {
		It("Should return output path from domain", func() {
			t := resolution.Type{
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
						},
					},
				},
			}
			Expect(ctx.GetOutputPath(t)).To(Equal("pkg/types.go"))
		})

		It("Should return empty for missing domain", func() {
			t := resolution.Type{
				Domains: map[string]resolution.Domain{},
			}
			Expect(ctx.GetOutputPath(t)).To(Equal(""))
		})
	})

	Describe("IsSameOutputEnum", func() {
		It("Should return true for enum in same namespace with matching output", func() {
			// Add a struct in same namespace to provide output path for enum
			Expect(table.Add(resolution.Type{
				Name:          "StructInNamespace",
				QualifiedName: "test.StructInNamespace",
				Namespace:     "test",
				Form:          resolution.StructForm{},
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
						},
					},
				},
			})).To(Succeed())

			enum := resolution.Type{
				Name:      "Status",
				Namespace: "test",
				Form:      resolution.EnumForm{},
			}
			Expect(ctx.IsSameOutputEnum(enum)).To(BeTrue())
		})

		It("Should return false for enum in different namespace", func() {
			enum := resolution.Type{
				Name:      "Status",
				Namespace: "other",
				Form:      resolution.EnumForm{},
			}
			Expect(ctx.IsSameOutputEnum(enum)).To(BeFalse())
		})
	})

	Describe("GetEnumOutputPath", func() {
		It("Should return output path from enum's own domain", func() {
			enum := resolution.Type{
				Name:      "Status",
				Namespace: "test",
				Form:      resolution.EnumForm{},
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/enums.go"}}},
						},
					},
				},
			}
			Expect(ctx.GetEnumOutputPath(enum)).To(Equal("pkg/enums.go"))
		})

		It("Should derive output path from struct in same namespace", func() {
			Expect(table.Add(resolution.Type{
				Name:          "StructInNamespace",
				QualifiedName: "test.StructInNamespace",
				Namespace:     "test",
				Form:          resolution.StructForm{},
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
						},
					},
				},
			})).To(Succeed())

			enum := resolution.Type{
				Name:      "Status",
				Namespace: "test",
				Form:      resolution.EnumForm{},
			}
			Expect(ctx.GetEnumOutputPath(enum)).To(Equal("pkg/types.go"))
		})

		It("Should return empty for orphaned enum", func() {
			enum := resolution.Type{
				Name:      "OrphanedEnum",
				Namespace: "orphan",
				Form:      resolution.EnumForm{},
			}
			Expect(ctx.GetEnumOutputPath(enum)).To(Equal(""))
		})
	})
})
