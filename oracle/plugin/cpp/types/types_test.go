// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"context"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/cpp/types"
	"github.com/synnaxlabs/oracle/testutil"
)

func TestCppTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "C++ Types Plugin Suite")
}

var _ = Describe("CppFormatter", func() {
	f := types.CppFormatter

	Describe("FormatQualified", func() {
		It("Should format qualified names with :: separator", func() {
			Expect(f.FormatQualified("synnax", "Type")).To(Equal("synnax::Type"))
		})

		It("Should return type name when qualifier is empty", func() {
			Expect(f.FormatQualified("", "Type")).To(Equal("Type"))
		})
	})

	Describe("FormatGeneric", func() {
		It("Should format generic types with angle brackets", func() {
			Expect(f.FormatGeneric("Container", []string{"int", "string"})).To(Equal("Container<int, string>"))
		})

		It("Should return base name when no type args", func() {
			Expect(f.FormatGeneric("Container", nil)).To(Equal("Container"))
		})
	})

	Describe("FormatArray", func() {
		It("Should format as std::vector", func() {
			Expect(f.FormatArray("int")).To(Equal("std::vector<int>"))
		})
	})

	Describe("FormatMap", func() {
		It("Should format as std::unordered_map", func() {
			Expect(f.FormatMap("string", "int")).To(Equal("std::unordered_map<string, int>"))
		})
	})

	Describe("FallbackType", func() {
		It("Should return void", func() {
			Expect(f.FallbackType()).To(Equal("void"))
		})
	})
})

var _ = Describe("CppImportResolver", func() {
	var r *types.CppImportResolver

	BeforeEach(func() {
		r = &types.CppImportResolver{FilePattern: "types.gen.h"}
	})

	Describe("ResolveImport", func() {
		It("Should return include path with file pattern", func() {
			importPath, qualifier, shouldImport := r.ResolveImport("client/cpp/user", nil)
			Expect(importPath).To(Equal("client/cpp/user/types.gen.h"))
			Expect(qualifier).To(Equal(""))
			Expect(shouldImport).To(BeTrue())
		})
	})
})

var _ = Describe("C++ Types Plugin", func() {
	var (
		ctx       context.Context
		loader    *testutil.MockFileLoader
		cppPlugin *types.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
		cppPlugin = types.New(types.DefaultOptions())
	})

	Describe("Check", func() {
		It("Should return nil (no validation required)", func() {
			Expect(cppPlugin.Check(&plugin.Request{})).To(BeNil())
		})
	})

	Describe("Namespace Derivation", func() {
		DescribeTable("should derive correct namespace from output path",
			func(outputPath, expectedNamespace string) {
				source := `
					@cpp output "` + outputPath + `"

					TestType struct {
						key uint32
					}
				`
				resp := testutil.MustGenerate(ctx, source, "test", loader, cppPlugin)
				testutil.ExpectContent(resp, "types.gen.h").
					ToContain("namespace " + expectedNamespace + " {")
			},
			// arc/cpp paths should use arc:: namespace
			Entry("arc/cpp/types", "arc/cpp/types", "arc::types"),
			Entry("arc/cpp/ir", "arc/cpp/ir", "arc::ir"),
			Entry("arc/cpp/graph", "arc/cpp/graph", "arc::graph"),
			// x/cpp paths should use x:: namespace
			Entry("x/cpp/telem", "x/cpp/telem", "x::telem"),
			Entry("x/cpp/json", "x/cpp/json", "x::json"),
			// client/cpp paths should use synnax:: namespace
			Entry("client/cpp/channel", "client/cpp/channel", "synnax::channel"),
			Entry("client/cpp/user", "client/cpp/user", "synnax::user"),
			// driver paths should use driver:: namespace
			Entry("driver/modbus", "driver/modbus", "driver::modbus"),
			Entry("driver/ni", "driver/ni", "driver::ni"),
			// unknown paths should default to synnax:: namespace
			Entry("other/path", "other/path", "synnax::path"),
		)
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(cppPlugin.Name()).To(Equal("cpp/types"))
		})

		It("Should filter on cpp domain", func() {
			Expect(cppPlugin.Domains()).To(Equal([]string{"cpp"}))
		})

		It("Should have no dependencies", func() {
			Expect(cppPlugin.Requires()).To(BeNil())
		})
	})

	Describe("Generate", func() {
		Context("basic struct generation", func() {
			It("Should generate struct for simple types", func() {
				source := `
					@cpp output "client/cpp/user"

					User struct {
						key uint32
						name string
						age int32
						active bool
					}
				`
				resp := testutil.MustGenerate(ctx, source, "user", loader, cppPlugin)
				Expect(resp.Files).To(HaveLen(1))

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						`// Code generated by oracle. DO NOT EDIT.`,
						`#pragma once`,
						`#include <cstdint>`,
						`#include <string>`,
						`namespace synnax::user {`,
						`struct User {`,
						`std::uint32_t key = 0;`,
						`std::string name;`,
						`std::int32_t age = 0;`,
						`bool active = false;`,
					)
			})
		})

		Context("primitive type mappings", func() {
			DescribeTable("should generate correct C++ type",
				func(oracleType, expectedFieldDecl string) {
					source := `
						@cpp output "out"

						Test struct {
							field ` + oracleType + `
						}
					`
					resp := testutil.MustGenerate(ctx, source, "test", loader, cppPlugin)
					testutil.ExpectContent(resp, "types.gen.h").ToContain(expectedFieldDecl)
				},
				Entry("string", "string", "std::string field;"),
				Entry("bool", "bool", "bool field = false;"),
				Entry("int8", "int8", "std::int8_t field = 0;"),
				Entry("int16", "int16", "std::int16_t field = 0;"),
				Entry("int32", "int32", "std::int32_t field = 0;"),
				Entry("int64", "int64", "std::int64_t field = 0;"),
				Entry("uint8", "uint8", "std::uint8_t field = 0;"),
				Entry("uint16", "uint16", "std::uint16_t field = 0;"),
				Entry("uint32", "uint32", "std::uint32_t field = 0;"),
				Entry("uint64", "uint64", "std::uint64_t field = 0;"),
				Entry("float32", "float32", "float field = 0;"),
				Entry("float64", "float64", "double field = 0;"),
				Entry("json", "json", "x::json::json field;"),
			)

		})

		It("Should treat soft optional as bare type", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					name string
					task_counter uint32?
					embedded bool?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Soft optionals (?) are just the bare type in C++
			Expect(content).To(ContainSubstring(`std::uint32_t task_counter = 0;`))
			Expect(content).To(ContainSubstring(`bool embedded = false;`))
			Expect(content).NotTo(ContainSubstring(`std::optional`))
		})

		It("Should use std::optional for hard optional types", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					parent uint32??
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Only hard optionals (??) use std::optional in C++
			Expect(content).To(ContainSubstring(`#include <optional>`))
			Expect(content).To(ContainSubstring(`std::optional<std::uint32_t> parent;`))
		})

		It("Should handle array types with std::vector", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					tags string[]
					values int32[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`#include <vector>`))
			Expect(content).To(ContainSubstring(`std::vector<std::string> tags;`))
			Expect(content).To(ContainSubstring(`std::vector<std::int32_t> values;`))
			// Note: vectors don't get = {} default since they have a proper default constructor
		})

		It("Should treat soft optional arrays as bare vector", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					tags string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Soft optional array is just the vector without std::optional
			Expect(content).To(ContainSubstring(`std::vector<std::string> tags;`))
		})

		It("Should wrap hard optional arrays with std::optional", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					tags string[]??
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Hard optional array wraps the vector with std::optional
			Expect(content).To(ContainSubstring(`std::optional<std::vector<std::string>> tags;`))
		})

		It("Should handle json type", func() {
			source := `
				@cpp output "client/cpp/task"

				Task struct {
					key uint64
					config json
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`#include "x/cpp/json/json.h"`))
			Expect(content).To(ContainSubstring(`x::json::json config;`))
		})

		It("Should handle map types", func() {
			source := `
				@cpp output "client/cpp/task"

				Task struct {
					key uint64
					metadata map<string, string>
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`#include <unordered_map>`))
			Expect(content).To(ContainSubstring(`std::unordered_map<std::string, std::string> metadata;`))
		})

		It("Should handle struct extension with field flattening", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					name string
					task_counter uint32?
				}

				New struct extends Rack {
					key uint32??
					-task_counter
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// New struct should have flattened fields with key hard optional and task_counter omitted
			Expect(content).To(ContainSubstring(`struct New {`))
			Expect(content).To(ContainSubstring(`std::optional<std::uint32_t> key;`))
			Expect(content).To(ContainSubstring(`std::string name;`))
			// task_counter should be omitted from New
			Expect(content).NotTo(MatchRegexp(`struct New \{[^}]*task_counter`))
		})

		It("Should handle @cpp name override", func() {
			source := `
				@cpp output "client/cpp/rack"
				@cpp name "RackPayload"

				Rack struct {
					key uint32
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`struct RackPayload {`))
			Expect(content).NotTo(ContainSubstring(`struct Rack {`))
		})

		It("Should handle @cpp name override for fields", func() {
			// This tests renaming a field, e.g., when the field name is a C++ reserved keyword
			source := `
				@cpp output "client/cpp/channel"

				Channel struct {
					key uint32
					name string
					virtual bool {
						@cpp name "is_virtual"
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "channel", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// The field should be renamed to is_virtual instead of virtual
			Expect(content).To(ContainSubstring(`bool is_virtual = false;`))
			// The original snake_case name should NOT appear
			Expect(content).NotTo(ContainSubstring(`bool virtual`))
		})

		It("Should handle @cpp omit", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					name string
				}

				Internal struct {
					data string

					@cpp omit
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`struct Rack {`))
			Expect(content).NotTo(ContainSubstring(`struct Internal {`))
		})

		It("Should generate generic structs with templates", func() {
			source := `
				@cpp output "client/cpp/status"

				Status struct<D> {
					key uint32
					details D
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "status", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`template <typename D>`))
			Expect(content).To(ContainSubstring(`struct Status {`))
			Expect(content).To(ContainSubstring(`D details;`))
		})

		It("Should generate generic struct with type parameter field", func() {
			source := `
				@cpp output "client/cpp/status"

				Status struct<D> {
					key uint32
					details D
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "status", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`template <typename D>`))
			Expect(content).To(ContainSubstring(`struct Status {`))
			Expect(content).To(ContainSubstring(`D details;`))
			Expect(content).To(ContainSubstring(`#include <type_traits>`))
		})

		It("Should generate method declarations for generic struct", func() {
			source := `
				@cpp output "client/cpp/status"

				Status struct<D> {
					key uint32
					details D
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "status", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`static Status parse(x::json::Parser parser);`))
			Expect(content).To(ContainSubstring(`[[nodiscard]] x::json::json to_json() const;`))
		})

		It("Should handle optional generic fields", func() {
			source := `
				@cpp output "client/cpp/status"

				Status struct<D?> {
					key uint32
					details D??
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "status", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`std::optional<D> details;`))
			Expect(content).To(ContainSubstring(`#include <optional>`))
			Expect(content).To(ContainSubstring(`#include <type_traits>`))
		})

		It("Should include type_traits for generic structs", func() {
			source := `
				@cpp output "client/cpp/status"

				Status struct<D> {
					key uint32
					details D
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "status", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`#include <type_traits>`))
		})

		It("Should handle inherited generic fields", func() {
			source := `
				@cpp output "x/cpp/status"

				Variant enum {
					success = "success"
					error   = "error"
				}

				Status struct<Details?, V extends Variant = Variant> {
					key     string
					variant V
					details Details?

					@cpp omit
				}

				GoStatus struct<Details?> extends Status<Details, Variant> {
					variant Variant
					-variant

					@cpp name "Status"
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "status", loader)
			if diag.HasErrors() {
				for _, e := range diag.Errors() {
					GinkgoWriter.Printf("Error: %s\n", e)
				}
			}
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`template <typename Details = std::monostate>`))
			Expect(content).To(ContainSubstring(`struct Status {`))
			Expect(content).To(ContainSubstring(`#include <type_traits>`))
		})

		It("Should handle type aliases", func() {
			source := `
				@cpp output "client/cpp/rack"

				StatusDetails struct {
					rack uint32
				}

				Status = StatusDetails
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`using Status = StatusDetails;`))
		})

		It("Should handle bytes type", func() {
			source := `
				@cpp output "client/cpp/module"

				Module struct {
					key uint32
					wasm bytes
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "module", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`std::vector<std::uint8_t> wasm;`))
		})

		It("Should handle array wrapper distinct types", func() {
			source := `
				@cpp output "client/cpp/types"

				Param struct {
					name string
					value int32
				}

				Params Param[]
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "types", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Array distinct types should generate wrapper structs, not using aliases
			Expect(content).To(ContainSubstring(`struct Params : private std::vector<Param>`))
			Expect(content).NotTo(ContainSubstring(`using Params = void;`))
		})

		It("Should handle array wrapper of structs used in other structs", func() {
			source := `
				@cpp output "client/cpp/types"

				Param struct {
					name string
					value int32
				}

				Params Param[]

				FunctionProperties struct {
					inputs Params
					outputs Params
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "types", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Array distinct types should generate wrapper structs
			Expect(content).To(ContainSubstring(`struct Params : private std::vector<Param>`))
			Expect(content).To(ContainSubstring(`Params inputs;`))
			Expect(content).To(ContainSubstring(`Params outputs;`))
		})

		It("Should order array wrapper after its dependency struct", func() {
			source := `
				@cpp output "client/cpp/types"

				Param struct {
					name string
					value int32
				}

				Params Param[]
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "types", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Param struct definition must appear before Params wrapper
			paramDefIdx := strings.Index(content, "struct Param {")
			paramsIdx := strings.Index(content, "struct Params : private")
			Expect(paramDefIdx).To(BeNumerically("<", paramsIdx),
				"Param struct definition should be declared before Params wrapper")
		})

		It("Should generate forward declarations for structs before array wrappers", func() {
			source := `
				@cpp output "client/cpp/types"

				Param struct {
					name string
					value int32
				}

				Params Param[]
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "types", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Forward declaration must appear before array wrapper
			fwdDeclIdx := strings.Index(content, "struct Param;")
			wrapperIdx := strings.Index(content, "struct Params : private")
			Expect(fwdDeclIdx).To(BeNumerically(">", -1), "Forward declaration should exist")
			Expect(fwdDeclIdx).To(BeNumerically("<", wrapperIdx),
				"Forward declaration should appear before array wrapper")
		})

		It("Should handle cyclic dependencies with forward declarations", func() {
			source := `
				@cpp output "client/cpp/types"

				FunctionProperties struct {
					inputs Params
				}

				Type struct {
					props FunctionProperties
					name string
				}

				Param struct {
					type Type
				}

				Params Param[]
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "types", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// All structs should have forward declarations
			Expect(content).To(ContainSubstring("struct FunctionProperties;"))
			Expect(content).To(ContainSubstring("struct Type;"))
			Expect(content).To(ContainSubstring("struct Param;"))
			// Array wrapper should generate a struct, not a using alias
			Expect(content).To(ContainSubstring("struct Params : private std::vector<Param>"))
		})

		It("Should handle int enums with uint8_t underlying type", func() {
			source := `
				@cpp output "client/cpp/status"

				Variant enum {
					success = 0
					error = 1
					warning = 2
				}

				Status struct {
					variant Variant
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "status", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`enum class Variant : std::uint8_t {`))
			Expect(content).To(ContainSubstring(`Success = 0,`))
			Expect(content).To(ContainSubstring(`Error = 1,`))
			Expect(content).To(ContainSubstring(`Warning = 2,`))
		})

		It("Should use indirect for self-referential optional fields", func() {
			source := `
				@cpp output "client/cpp/types"

				Node struct {
					name string
					left Node??
					right Node??
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "types", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Self-referential fields should use x::mem::indirect, not optional
			Expect(content).To(ContainSubstring(`x::mem::indirect<Node> left;`))
			Expect(content).To(ContainSubstring(`x::mem::indirect<Node> right;`))
			Expect(content).To(ContainSubstring(`#include "x/cpp/mem/indirect.h"`))
			// Should NOT use std::optional for self-referential types
			Expect(content).NotTo(ContainSubstring(`std::optional<Node>`))
		})

		It("Should use optional for non-self-referential optional fields", func() {
			source := `
				@cpp output "client/cpp/types"

				Unit struct {
					name string
					scale float64
				}

				Type struct {
					name string
					unit Unit??
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "types", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Non-self-referential optional fields should use std::optional
			Expect(content).To(ContainSubstring(`std::optional<Unit> unit;`))
			Expect(content).To(ContainSubstring(`#include <optional>`))
		})

		It("Should handle cross-namespace references to handwritten types", func() {
			// First, set up the "status" namespace with an omitted type
			statusSource := `
				@cpp omit
				@cpp include "x/cpp/status/status.h"

				Status struct<D?> {
					key string
					details D?
				}
			`
			loader.Add("schemas/status", statusSource)

			// Then, the "rack" namespace that references it
			rackSource := `
				import "schemas/status"

				@cpp output "client/cpp/rack"

				StatusDetails struct {
					rack uint32
				}

				RackStatus = status.Status<StatusDetails>

				Rack struct {
					key uint32
					status RackStatus??
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, rackSource, "rack", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`#include "x/cpp/status/status.h"`))
			Expect(content).To(ContainSubstring(`using RackStatus = ::status::Status<StatusDetails>;`))
			Expect(content).To(ContainSubstring(`std::optional<RackStatus> status;`))
		})

		It("Should not generate files for structs without @cpp output", func() {
			source := `
				@go output "core/pkg/service/user"

				User struct {
					key uint32
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := cppPlugin.Generate(req)
			Expect(err).To(BeNil())
			Expect(resp.Files).To(HaveLen(0))
		})

		Context("declaration and field order", func() {
			It("Should preserve struct declaration order", func() {
				source := `
					@cpp output "client/cpp/animals"

					Zebra struct {
						name string
					}

					Apple struct {
						color string
					}

					Mango struct {
						ripe bool
					}
				`
				table, diag := analyzer.AnalyzeSource(ctx, source, "animals", loader)
				Expect(diag.HasErrors()).To(BeFalse())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp, err := cppPlugin.Generate(req)
				Expect(err).To(BeNil())

				content := string(resp.Files[0].Content)
				zebraIdx := strings.Index(content, "struct Zebra")
				appleIdx := strings.Index(content, "struct Apple")
				mangoIdx := strings.Index(content, "struct Mango")
				Expect(zebraIdx).To(BeNumerically("<", appleIdx))
				Expect(appleIdx).To(BeNumerically("<", mangoIdx))
			})

			It("Should preserve field declaration order", func() {
				source := `
					@cpp output "client/cpp/order"

					Record struct {
						zebra string
						apple int32
						mango bool
					}
				`
				table, diag := analyzer.AnalyzeSource(ctx, source, "order", loader)
				Expect(diag.HasErrors()).To(BeFalse())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp, err := cppPlugin.Generate(req)
				Expect(err).To(BeNil())

				content := string(resp.Files[0].Content)
				// Fields now have default values, so match the field names in declaration context
				zebraIdx := strings.Index(content, "std::string zebra;")
				appleIdx := strings.Index(content, "std::int32_t apple = 0;")
				mangoIdx := strings.Index(content, "bool mango = false;")
				Expect(zebraIdx).To(BeNumerically(">", -1), "zebra field should exist")
				Expect(appleIdx).To(BeNumerically(">", -1), "apple field should exist")
				Expect(mangoIdx).To(BeNumerically(">", -1), "mango field should exist")
				Expect(zebraIdx).To(BeNumerically("<", appleIdx))
				Expect(appleIdx).To(BeNumerically("<", mangoIdx))
			})
		})

		Describe("Namespace-based enum collection", func() {
			It("Should include standalone enums in the same namespace as structs", func() {
				// This tests the fix for enums that are not referenced by struct fields
				// but are in the same namespace and should be generated in the same file.
				source := `
					@cpp output "x/cpp/control"

					Concurrency enum {
						exclusive = 0
						shared = 1
					}

					Subject struct {
						key string
						name string
					}
				`
				resp := testutil.MustGenerate(ctx, source, "control", loader, cppPlugin)

				// The Concurrency enum should be generated even though Subject
				// doesn't reference it - they're in the same namespace
				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"namespace x::control {",
						"enum class Concurrency : std::uint8_t {",
						"Exclusive = 0,",
						"Shared = 1,",
						"struct Subject {",
					)
			})

			It("Should include enums inherited from file-level output directive", func() {
				source := `
					@cpp output "x/cpp/status"

					Variant enum {
						success = 0
						error = 1
					}

					Status struct {
						key uint32
					}
				`
				resp := testutil.MustGenerate(ctx, source, "status", loader, cppPlugin)

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"enum class Variant : std::uint8_t {",
						"struct Status {",
					)
			})
		})

		Describe("Cross-namespace enum references", func() {
			It("Should use namespace-qualified name for cross-namespace int enums", func() {
				// First, set up the "control" namespace with the enum
				controlSource := `
					@cpp output "x/cpp/control"

					Concurrency enum {
						exclusive = 0
						shared = 1
					}

					Subject struct {
						key string
					}
				`
				loader.Add("schemas/control", controlSource)

				// Then, the "channel" namespace that references it
				channelSource := `
					import "schemas/control"

					@cpp output "client/cpp/channel"

					Channel struct {
						key uint32
						concurrency control.Concurrency
					}
				`
				table, diag := analyzer.AnalyzeSource(ctx, channelSource, "channel", loader)
				Expect(diag.HasErrors()).To(BeFalse())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp, err := cppPlugin.Generate(req)
				Expect(err).To(BeNil())
				// Both channel and control files are generated when importing
				Expect(resp.Files).To(HaveLen(2))

				// Find and check the channel file content
				var channelContent string
				for _, f := range resp.Files {
					if strings.HasSuffix(f.Path, "client/cpp/channel/types.gen.h") {
						channelContent = string(f.Content)
						break
					}
				}
				Expect(channelContent).NotTo(BeEmpty())
				// Should include the control types header
				Expect(channelContent).To(ContainSubstring(`#include "x/cpp/control/types.gen.h"`))
				// Should use namespace-qualified enum type
				Expect(channelContent).To(ContainSubstring(`::x::control::Concurrency concurrency`))
			})

			It("Should not namespace-qualify enums in the same namespace", func() {
				source := `
					@cpp output "client/cpp/channel"

					Concurrency enum {
						exclusive = 0
						shared = 1
					}

					Channel struct {
						key uint32
						concurrency Concurrency
					}
				`
				resp := testutil.MustGenerate(ctx, source, "channel", loader, cppPlugin)

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"enum class Concurrency : std::uint8_t {",
						"Concurrency concurrency",
					).
					ToNotContain(
						"::synnax::channel::Concurrency",
					)
			})
		})

		Describe("Array Wrapper Generation", func() {
			It("Should generate wrapper struct for array distinct types", func() {
				source := `
					@cpp output "arc/cpp/types"

					Channels uint32[]
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, cppPlugin)

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"struct Channels : private std::vector<std::uint32_t>",
						"using Base = std::vector<std::uint32_t>;",
						"using Base::Base;",
						"using Base::begin;",
						"using Base::end;",
						"using Base::size;",
						"using Base::push_back;",
						"using Base::operator[]",
					)
			})

			It("Should generate wrapper struct for array of structs", func() {
				source := `
					@cpp output "arc/cpp/types"

					Param struct {
						name string
						dataType string
					}

					Params Param[]
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, cppPlugin)

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"struct Params : private std::vector<Param>",
						"using Base = std::vector<Param>;",
					)
			})

			It("Should generate parse/to_json declarations for array wrappers", func() {
				source := `
					@cpp output "arc/cpp/types"

					Channels uint32[]
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, cppPlugin)

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"static Channels parse(x::json::Parser parser);",
						"[[nodiscard]] x::json::json to_json() const;",
					)
			})

			// Note: Proto declarations for array wrappers are tested in the pb plugin tests.
			// The types plugin generates declarations when HasProto is set, which
			// requires explicit @pb annotations on the type (tested in pb plugin).

			It("Should support @cpp methods for custom methods on array wrappers", func() {
				source := `
					@cpp output "arc/cpp/types"
					@cpp methods "std::optional<Param> get(const std::string& name) const"

					Param struct {
						name string
					}

					Params Param[]
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, cppPlugin)

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"std::optional<Param> get(const std::string& name) const;",
					)
			})

			It("Should generate initializer_list constructor for array wrappers", func() {
				source := `
					@cpp output "arc/cpp/types"

					Channels uint32[]
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, cppPlugin)

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"Channels(std::initializer_list<std::uint32_t> init) : Base(init) {}",
					)
			})
		})

		Context("documentation", func() {
			It("Should generate doxygen comments from doc domain", func() {
				source := `
					@cpp output "client/cpp/user"

					User struct {
						@doc value "is a representation of a user in the system."

						key uint32 @key {
							@doc value "is the unique identifier for the user."
						}

						name string {
							@doc value "is the user's display name."
						}

						age int32
					}
				`
				resp := testutil.MustGenerate(ctx, source, "user", loader, cppPlugin)

				testutil.ExpectContent(resp, "types.gen.h").
					ToContain(
						"/// @brief User is a representation of a user in the system.",
						"/// @brief key is the unique identifier for the user.",
						"/// @brief name is the user's display name.",
					)
			})
		})
	})
})
