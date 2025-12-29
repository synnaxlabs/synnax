// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package generate_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/jerky/deps"
	"github.com/synnaxlabs/x/jerky/generate"
	"github.com/synnaxlabs/x/jerky/parse"
	"github.com/synnaxlabs/x/jerky/state"
	"github.com/synnaxlabs/x/jerky/typemap"
)

var _ = Describe("Pure Functions", func() {
	Describe("ToSnakeCase", func() {
		DescribeTable("should convert PascalCase to snake_case",
			func(input, expected string) {
				Expect(generate.ToSnakeCase(input)).To(Equal(expected))
			},
			Entry("simple PascalCase", "UserName", "user_name"),
			Entry("single word", "User", "user"),
			Entry("already lowercase", "user", "user"),
			Entry("multiple capitals", "HTTPServer", "h_t_t_p_server"),
			Entry("acronym at end", "GetUserID", "get_user_i_d"),
			Entry("empty string", "", ""),
			Entry("single letter", "A", "a"),
			Entry("numbers", "User123Name", "user123_name"),
			Entry("long name", "GetUserByFirstNameAndLastName", "get_user_by_first_name_and_last_name"),
			Entry("consecutive capitals", "XMLParser", "x_m_l_parser"),
		)
	})

	Describe("TypeName", func() {
		DescribeTable("should extract type name without package prefix",
			func(input, expected string) {
				Expect(generate.TypeName(input)).To(Equal(expected))
			},
			Entry("simple package.Type", "user.User", "User"),
			Entry("nested package", "pkg.types.user.User", "User"),
			Entry("no package", "User", "User"),
			Entry("full import path", "github.com/synnaxlabs/x/jerky/example.User", "User"),
			Entry("single character type", "pkg.A", "A"),
			Entry("empty string", "", ""),
		)
	})

	Describe("ProtoToGoType", func() {
		DescribeTable("should convert proto types to Go equivalents",
			func(protoType, expected string) {
				Expect(generate.ProtoToGoType(protoType)).To(Equal(expected))
			},
			// String type
			Entry("string", "string", "string"),

			// 32-bit signed integers
			Entry("int32", "int32", "int32"),
			Entry("sint32", "sint32", "int32"),
			Entry("sfixed32", "sfixed32", "int32"),

			// 64-bit signed integers
			Entry("int64", "int64", "int64"),
			Entry("sint64", "sint64", "int64"),
			Entry("sfixed64", "sfixed64", "int64"),

			// 32-bit unsigned integers
			Entry("uint32", "uint32", "uint32"),
			Entry("fixed32", "fixed32", "uint32"),

			// 64-bit unsigned integers
			Entry("uint64", "uint64", "uint64"),
			Entry("fixed64", "fixed64", "uint64"),

			// Floating point
			Entry("double", "double", "float64"),
			Entry("float", "float", "float32"),

			// Boolean
			Entry("bool", "bool", "bool"),

			// Bytes
			Entry("bytes", "bytes", "[]byte"),

			// Unknown types default to string
			Entry("unknown type", "SomeCustomType", "string"),
			Entry("empty string", "", "string"),
		)
	})

	Describe("GetVersionFields", func() {
		var typeState state.TypeState

		BeforeEach(func() {
			typeState = state.TypeState{
				TypeName: "User",
				History: []state.VersionHistory{
					{
						Version: 1,
						Fields: map[string]state.FieldInfo{
							"Name": {FieldNumber: 1, Type: "string"},
							"Age":  {FieldNumber: 2, Type: "int32"},
						},
					},
					{
						Version: 2,
						Fields: map[string]state.FieldInfo{
							"Name":  {FieldNumber: 1, Type: "string"},
							"Age":   {FieldNumber: 2, Type: "int32"},
							"Email": {FieldNumber: 3, Type: "string"},
						},
					},
					{
						Version: 3,
						Fields: map[string]state.FieldInfo{
							"Name":  {FieldNumber: 1, Type: "string"},
							"Email": {FieldNumber: 3, Type: "string"},
						},
					},
				},
			}
		})

		It("should return fields for version 1", func() {
			fields := generate.GetVersionFields(typeState, 1)
			Expect(fields).To(HaveLen(2))
			Expect(fields).To(HaveKey("Name"))
			Expect(fields).To(HaveKey("Age"))
		})

		It("should return fields for version 2 (with added field)", func() {
			fields := generate.GetVersionFields(typeState, 2)
			Expect(fields).To(HaveLen(3))
			Expect(fields).To(HaveKey("Name"))
			Expect(fields).To(HaveKey("Age"))
			Expect(fields).To(HaveKey("Email"))
		})

		It("should return fields for version 3 (with removed field)", func() {
			fields := generate.GetVersionFields(typeState, 3)
			Expect(fields).To(HaveLen(2))
			Expect(fields).To(HaveKey("Name"))
			Expect(fields).To(HaveKey("Email"))
			Expect(fields).NotTo(HaveKey("Age"))
		})

		It("should return empty map for non-existent version", func() {
			fields := generate.GetVersionFields(typeState, 99)
			Expect(fields).To(BeEmpty())
		})

		It("should return empty map for empty type state", func() {
			emptyState := state.TypeState{TypeName: "Empty", History: []state.VersionHistory{}}
			fields := generate.GetVersionFields(emptyState, 1)
			Expect(fields).To(BeEmpty())
		})
	})
})

var _ = Describe("Registry-Dependent Functions", func() {
	var (
		g       *generate.Generator
		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "jerky-generate-test-*")
		Expect(err).ToNot(HaveOccurred())

		g, err = generate.NewGenerator(tempDir, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("GetProtoType", func() {
		Context("with primitive types", func() {
			DescribeTable("should map primitive Go types to proto types",
				func(goTypeName, expectedProto string, expectCanFail bool) {
					goType := parse.GoType{Kind: parse.KindPrimitive, Name: goTypeName}
					protoType, canFail := g.GetProtoType(goType)
					Expect(protoType).To(Equal(expectedProto))
					Expect(canFail).To(Equal(expectCanFail))
				},
				Entry("bool", "bool", "bool", false),
				Entry("string", "string", "string", false),
				Entry("[]byte", "[]byte", "bytes", false),
				Entry("int", "int", "int64", false),
				Entry("int8", "int8", "int32", false),
				Entry("int16", "int16", "int32", false),
				Entry("int32", "int32", "int32", false),
				Entry("int64", "int64", "int64", false),
				Entry("uint", "uint", "uint64", false),
				Entry("uint8", "uint8", "uint32", false),
				Entry("uint16", "uint16", "uint32", false),
				Entry("uint32", "uint32", "uint32", false),
				Entry("uint64", "uint64", "uint64", false),
				Entry("float32", "float32", "float", false),
				Entry("float64", "float64", "double", false),
			)
		})

		Context("with external types", func() {
			It("should map uuid.UUID to string with canFail=true", func() {
				goType := parse.GoType{Kind: parse.KindNamed, Name: "uuid.UUID"}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("string"))
				Expect(canFail).To(BeTrue())
			})

			It("should map time.Time to int64", func() {
				goType := parse.GoType{Kind: parse.KindNamed, Name: "time.Time"}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("int64"))
				Expect(canFail).To(BeFalse())
			})

			It("should map time.Duration to int64", func() {
				goType := parse.GoType{Kind: parse.KindNamed, Name: "time.Duration"}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("int64"))
				Expect(canFail).To(BeFalse())
			})

			It("should map telem.TimeStamp to int64", func() {
				goType := parse.GoType{Kind: parse.KindNamed, Name: "telem.TimeStamp"}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("int64"))
				Expect(canFail).To(BeFalse())
			})
		})

		Context("with slices", func() {
			It("should handle slice of primitive type", func() {
				goType := parse.GoType{
					Kind: parse.KindSlice,
					Name: "[]string",
					Elem: &parse.GoType{Kind: parse.KindPrimitive, Name: "string"},
				}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("repeated string"))
				Expect(canFail).To(BeFalse())
			})

			It("should handle slice of int32", func() {
				goType := parse.GoType{
					Kind: parse.KindSlice,
					Name: "[]int32",
					Elem: &parse.GoType{Kind: parse.KindPrimitive, Name: "int32"},
				}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("repeated int32"))
				Expect(canFail).To(BeFalse())
			})
		})

		Context("with maps", func() {
			It("should handle map[string]int32", func() {
				goType := parse.GoType{
					Kind: parse.KindMap,
					Name: "map[string]int32",
					Key:  &parse.GoType{Kind: parse.KindPrimitive, Name: "string"},
					Elem: &parse.GoType{Kind: parse.KindPrimitive, Name: "int32"},
				}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("map<string, int32>"))
				Expect(canFail).To(BeFalse())
			})

			It("should handle map[int64]string", func() {
				goType := parse.GoType{
					Kind: parse.KindMap,
					Name: "map[int64]string",
					Key:  &parse.GoType{Kind: parse.KindPrimitive, Name: "int64"},
					Elem: &parse.GoType{Kind: parse.KindPrimitive, Name: "string"},
				}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("map<int64, string>"))
				Expect(canFail).To(BeFalse())
			})
		})

		Context("with underlying types", func() {
			It("should use underlying type for named type aliases", func() {
				goType := parse.GoType{
					Kind:       parse.KindNamed,
					Name:       "UserID",
					Underlying: &parse.GoType{Kind: parse.KindPrimitive, Name: "uint32"},
				}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("uint32"))
				Expect(canFail).To(BeFalse())
			})
		})

		Context("with unknown types", func() {
			It("should default to string for unknown types", func() {
				goType := parse.GoType{Kind: parse.KindNamed, Name: "SomeUnknownType"}
				protoType, canFail := g.GetProtoType(goType)
				Expect(protoType).To(Equal("string"))
				Expect(canFail).To(BeFalse())
			})
		})

		Context("with jerky-managed types", func() {
			It("should resolve jerky type from dependency registry", func() {
				// Create generator with dependency registry
				depRegistry := deps.NewRegistry()
				depRegistry.Register(deps.TypeInfo{
					PackagePath:    "example/types/address",
					PackageName:    "example",
					TypeName:       "Address",
					CurrentVersion: 2,
				})

				gWithDeps, err := generate.NewGeneratorWithDeps(tempDir, nil, depRegistry)
				Expect(err).ToNot(HaveOccurred())

				goType := parse.GoType{
					Kind:        parse.KindNamed,
					Name:        "address.Address",
					PackagePath: "example/types/address",
					IsJerky:     true,
				}
				protoType, canFail := gWithDeps.GetProtoType(goType)
				Expect(protoType).To(Equal("example.types.address.V2"))
				Expect(canFail).To(BeFalse())
			})
		})
	})

	Describe("GetTranslationExprs", func() {
		Context("with primitive types", func() {
			It("should generate direct expressions for string", func() {
				field := parse.ParsedField{
					Name:   "Name",
					GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"},
				}
				forward, backward, canFail, imports, _ := g.GetTranslationExprs(field, "", "")
				Expect(forward).To(Equal("m.Name"))
				Expect(backward).To(Equal("pb.Name"))
				Expect(canFail).To(BeFalse())
				Expect(imports).To(BeEmpty())
			})

			It("should generate cast expressions for int", func() {
				field := parse.ParsedField{
					Name:   "Count",
					GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "int"},
				}
				forward, backward, canFail, imports, _ := g.GetTranslationExprs(field, "", "")
				Expect(forward).To(Equal("int64(m.Count)"))
				Expect(backward).To(Equal("int(pb.Count)"))
				Expect(canFail).To(BeFalse())
				Expect(imports).To(BeEmpty())
			})

			It("should generate cast expressions for uint8", func() {
				field := parse.ParsedField{
					Name:   "Flags",
					GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "uint8"},
				}
				forward, backward, canFail, imports, _ := g.GetTranslationExprs(field, "", "")
				Expect(forward).To(Equal("uint32(m.Flags)"))
				Expect(backward).To(Equal("uint8(pb.Flags)"))
				Expect(canFail).To(BeFalse())
				Expect(imports).To(BeEmpty())
			})
		})

		Context("with external types", func() {
			It("should generate expressions for uuid.UUID with canFail=true", func() {
				field := parse.ParsedField{
					Name:   "ID",
					GoType: parse.GoType{Kind: parse.KindNamed, Name: "uuid.UUID"},
				}
				forward, backward, canFail, imports, _ := g.GetTranslationExprs(field, "", "")
				Expect(forward).To(Equal("m.ID.String()"))
				Expect(backward).To(Equal("uuid.Parse(pb.ID)"))
				Expect(canFail).To(BeTrue())
				Expect(imports).To(ContainElement("github.com/google/uuid"))
			})

			It("should generate expressions for time.Time", func() {
				field := parse.ParsedField{
					Name:   "CreatedAt",
					GoType: parse.GoType{Kind: parse.KindNamed, Name: "time.Time"},
				}
				forward, backward, canFail, imports, _ := g.GetTranslationExprs(field, "", "")
				Expect(forward).To(Equal("m.CreatedAt.UnixNano()"))
				Expect(backward).To(Equal("time.Unix(0, pb.CreatedAt)"))
				Expect(canFail).To(BeFalse())
				Expect(imports).To(ContainElement("time"))
			})
		})

		Context("with type aliases", func() {
			It("should cast local type alias correctly", func() {
				field := parse.ParsedField{
					Name: "UserID",
					GoType: parse.GoType{
						Kind:       parse.KindNamed,
						Name:       "UserID",
						Underlying: &parse.GoType{Kind: parse.KindPrimitive, Name: "uint32"},
					},
				}
				forward, backward, canFail, imports, _ := g.GetTranslationExprs(field, "", "")
				Expect(forward).To(Equal("uint32(m.UserID)"))
				Expect(backward).To(Equal("UserID(pb.UserID)"))
				Expect(canFail).To(BeFalse())
				Expect(imports).To(BeEmpty())
			})

			It("should cast external type alias with package prefix", func() {
				field := parse.ParsedField{
					Name: "Key",
					GoType: parse.GoType{
						Kind:        parse.KindNamed,
						Name:        "core.Key",
						PackageName: "core",
						PackagePath: "github.com/example/core",
						Underlying:  &parse.GoType{Kind: parse.KindPrimitive, Name: "uint32"},
					},
				}
				forward, backward, canFail, imports, _ := g.GetTranslationExprs(field, "", "different/package")
				Expect(forward).To(Equal("uint32(m.Key)"))
				Expect(backward).To(Equal("core.Key(pb.Key)"))
				Expect(canFail).To(BeFalse())
				Expect(imports).To(ContainElement("github.com/example/core"))
			})
		})
	})

	Describe("ComputeMigrationFields", func() {
		It("should return only common fields between versions", func() {
			typeState := state.TypeState{
				TypeName:   "User",
				FieldOrder: []string{"Name", "Email"},
				History: []state.VersionHistory{
					{
						Version: 1,
						Fields: map[string]state.FieldInfo{
							"Name": {FieldNumber: 1, Type: "string"},
						},
					},
					{
						Version: 2,
						Fields: map[string]state.FieldInfo{
							"Name":  {FieldNumber: 1, Type: "string"},
							"Email": {FieldNumber: 2, Type: "string"},
						},
					},
				},
			}

			fromVH := &typeState.History[0]
			toVH := &typeState.History[1]

			fields := g.ComputeMigrationFields(typeState, fromVH, toVH)

			// Should only have Name (common field), not Email (added in v2)
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].Name).To(Equal("Name"))
		})

		It("should exclude removed fields", func() {
			typeState := state.TypeState{
				TypeName:   "User",
				FieldOrder: []string{"Name", "TempData"},
				History: []state.VersionHistory{
					{
						Version: 1,
						Fields: map[string]state.FieldInfo{
							"Name":     {FieldNumber: 1, Type: "string"},
							"TempData": {FieldNumber: 2, Type: "string"},
						},
					},
					{
						Version: 2,
						Fields: map[string]state.FieldInfo{
							"Name": {FieldNumber: 1, Type: "string"},
						},
					},
				},
			}

			fromVH := &typeState.History[0]
			toVH := &typeState.History[1]

			fields := g.ComputeMigrationFields(typeState, fromVH, toVH)

			// Should only have Name field (TempData is removed in v2)
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].Name).To(Equal("Name"))
		})

		It("should track jerky nested type version changes", func() {
			typeState := state.TypeState{
				TypeName:   "User",
				FieldOrder: []string{"Address"},
				History: []state.VersionHistory{
					{
						Version: 1,
						Fields: map[string]state.FieldInfo{
							"Address": {FieldNumber: 1, Type: "Address"},
						},
						NestedTypeVersions: map[string]state.NestedTypeInfo{
							"Address": {TypeName: "Address", Version: 1},
						},
					},
					{
						Version: 2,
						Fields: map[string]state.FieldInfo{
							"Address": {FieldNumber: 1, Type: "Address"},
						},
						NestedTypeVersions: map[string]state.NestedTypeInfo{
							"Address": {TypeName: "Address", Version: 2},
						},
					},
				},
			}

			fromVH := &typeState.History[0]
			toVH := &typeState.History[1]

			fields := g.ComputeMigrationFields(typeState, fromVH, toVH)

			Expect(fields).To(HaveLen(1))
			Expect(fields[0].Name).To(Equal("Address"))
			Expect(fields[0].IsJerky).To(BeTrue())
			Expect(fields[0].JerkyTypeName).To(Equal("Address"))
			Expect(fields[0].FromJerkyVersion).To(Equal(1))
			Expect(fields[0].ToJerkyVersion).To(Equal(2))
			Expect(fields[0].VersionChanged).To(BeTrue())
		})

		It("should handle slice of jerky type", func() {
			typeState := state.TypeState{
				TypeName:   "User",
				FieldOrder: []string{"Addresses"},
				History: []state.VersionHistory{
					{
						Version: 1,
						Fields: map[string]state.FieldInfo{
							"Addresses": {FieldNumber: 1, Type: "[]Address"},
						},
						NestedTypeVersions: map[string]state.NestedTypeInfo{
							"Addresses": {TypeName: "Address", Version: 1, IsSlice: true},
						},
					},
					{
						Version: 2,
						Fields: map[string]state.FieldInfo{
							"Addresses": {FieldNumber: 1, Type: "[]Address"},
						},
						NestedTypeVersions: map[string]state.NestedTypeInfo{
							"Addresses": {TypeName: "Address", Version: 1, IsSlice: true},
						},
					},
				},
			}

			fromVH := &typeState.History[0]
			toVH := &typeState.History[1]

			fields := g.ComputeMigrationFields(typeState, fromVH, toVH)

			Expect(fields).To(HaveLen(1))
			Expect(fields[0].IsJerky).To(BeTrue())
			Expect(fields[0].IsSlice).To(BeTrue())
			Expect(fields[0].VersionChanged).To(BeFalse())
		})
	})
})

var _ = Describe("Generator with Custom Registry", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "jerky-custom-registry-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("should use custom type mappings", func() {
		registry := typemap.NewRegistry()
		registry.Register(typemap.Mapping{
			GoType:       "custom.Type",
			ProtoType:    "bytes",
			ForwardExpr:  "{{.Field}}.Serialize()",
			BackwardExpr: "custom.Deserialize({{.Field}})",
			NeedsImport:  []string{"example.com/custom"},
			CanFail:      true,
		})

		g, err := generate.NewGenerator(tempDir, registry)
		Expect(err).ToNot(HaveOccurred())

		goType := parse.GoType{Kind: parse.KindNamed, Name: "custom.Type"}
		protoType, canFail := g.GetProtoType(goType)
		Expect(protoType).To(Equal("bytes"))
		Expect(canFail).To(BeTrue())

		field := parse.ParsedField{
			Name:   "Data",
			GoType: goType,
		}
		forward, backward, canFail, imports, _ := g.GetTranslationExprs(field, "", "")
		Expect(forward).To(Equal("m.Data.Serialize()"))
		Expect(backward).To(Equal("custom.Deserialize(pb.Data)"))
		Expect(canFail).To(BeTrue())
		Expect(imports).To(ContainElement("example.com/custom"))
	})
})
