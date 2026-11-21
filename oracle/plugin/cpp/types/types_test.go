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
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/cpp/types"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestCppTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Cpp Types Suite", SuiteConfig())
}

var _ = Describe("C++ Types Plugin", func() {
	var (
		loader    *MockFileLoader
		cppPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		cppPlugin = types.New(types.DefaultOptions())
	})

	Describe("Check", func() {
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
				resp := MustGenerate(source, "test", loader, cppPlugin)
				ExpectContent(resp, "types.gen.h").
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
		It("Should generate the recursive Where of a filter", func() {
			AddFilterSchema(loader)
			resp := MustGenerate(ThingFilter, "thing", loader, cppPlugin)
			ExpectContent(resp, "thing/types.gen.h").
				ToContain(
					"std::optional<std::vector<Where>> and_;",
					"std::optional<std::vector<Where>> or_;",
					"x::mem::indirect<Where> not_;",
				).
				ToNotContain("x::mem::indirect<std::vector<Where>>")
		})

		It("Should type filter lines to take plain values", func() {
			AddFilterSchema(loader)
			source := `
				import "schemas/x/filter"

				@cpp output "client/cpp/thing"

				Thing struct {
					key  uint32
					name string
				}

				Where filter Thing {
					key
					name
					tags   string[]
					active bool
				}
			`
			resp := MustGenerate(source, "thing", loader, cppPlugin)
			ExpectContent(resp, "thing/types.gen.h").ToContain(
				`#include "x/cpp/filter/field.h"`,
				"::x::filter::Field<::x::filter::Ordered<std::uint32_t>> key;",
				"::x::filter::Field<::x::filter::String> name;",
				"::x::filter::List<std::string> tags;",
				"std::optional<bool> active;",
			)
		})

		It("Should combine filters with &&, ||, and !", func() {
			AddFilterSchema(loader)
			resp := MustGenerate(ThingFilter, "thing", loader, cppPlugin)
			ExpectContent(resp, "thing/types.gen.h").ToContain(
				"friend Where operator&&(Where a, Where b) {",
				"w.and_->push_back(std::move(a));",
				"friend Where operator||(Where a, Where b) {",
				"w.or_->push_back(std::move(b));",
				"friend Where operator!(Where a) {",
				"w.not_ = std::move(a);",
			)
		})

		It("Should give a retrieve request its options", func() {
			AddFilterSchema(loader)
			resp := MustGenerate(ThingFilter, "thing", loader, cppPlugin)
			ExpectContent(resp, "thing/types.gen.h").ToContain(
				"struct RetrieveOptions {\n"+
					"    /// @brief after starts the things strictly past this position in their order.\n"+
					"    std::optional<Cursor> after;\n"+
					"    /// @brief limit is the most things to return.\n"+
					"    std::optional<std::uint32_t> limit;\n"+
					"};",
				"static RetrieveRequest\n"+
					"    from(Where where, const RetrieveOptions &options = {}) {\n"+
					"        return {\n"+
					"            .where = std::move(where),\n"+
					"            .after = options.after,\n"+
					"            .limit = options.limit,\n"+
					"        };\n"+
					"    }",
			)
		})

		It("Should not combine a struct that is not a filter", func() {
			source := `
				@cpp output "out"

				Thing struct {
					name string?
				}
			`
			resp := MustGenerate(source, "thing", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain("std::optional<std::string> name;").
				ToNotContain("operator&&", "x/cpp/filter/field.h")
		})

		Context("field-level omit", func() {
			It("Should leave out a field omitted in cpp", func() {
				source := `
					@go output "core/pkg/service/auth"
					@cpp output "client/cpp/auth"

					Session struct {
						key     uint32
						hash    bytes { @cpp omit }
						subject string
					}
				`
				resp := MustGenerate(source, "auth", loader, cppPlugin)
				ExpectContent(resp, "types.gen.h").
					ToContain("std::string subject;").
					ToNotContain("hash")
			})
		})

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
				resp := MustGenerate(source, "user", loader, cppPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "types.gen.h").
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

		Context("schemas with the same file name", func() {
			It("Should leave out the types of a schema with no C++ output", func() {
				loader.Add("schemas/x/color", `
					@cpp output "x/cpp/color"
					@go output  "x/go/color"

					Color struct {
						r uint8
					}
				`)
				loader.Add("schemas/synnax/color", `
					import "schemas/x/color"

					@go output "core/pkg/service/color"

					Named struct {
						light color.Color
					}
				`)
				resp := MustGenerateMulti(loader, cppPlugin)
				Expect(resp.Files).To(HaveLen(1))
				ExpectContent(resp, "x/cpp/color/types.gen.h").
					ToContain("struct Color {").
					ToNotContain("Named")
			})

			It("Should qualify types from a schema with the same file name", func() {
				loader.Add("schemas/x/color", `
					@cpp output "x/cpp/color"

					Kind enum {
						solid = 1
						gradient = 2
					}

					Stop struct {
						position float32
						kind     Kind
					}
				`)
				loader.Add("schemas/synnax/color", `
					import "schemas/x/color"

					@cpp output "client/cpp/color"

					Named struct {
						stop color.Stop
						kind color.Kind
					}
				`)
				resp := MustGenerateMulti(loader, cppPlugin)
				ExpectContent(resp, "client/cpp/color/types.gen.h").
					ToContain(
						`#include "x/cpp/color/types.gen.h"`,
						"::x::color::Stop stop;",
						"::x::color::Kind kind = {};",
					).
					ToNotContain("enum class Kind")
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
					resp := MustGenerate(source, "test", loader, cppPlugin)
					ExpectContent(resp, "types.gen.h").ToContain(expectedFieldDecl)
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
				Entry("record", "record", "x::json::json::object_t field;"),
			)
		})

		Context("nil primitive type", func() {
			It("Should map nil to std::monostate", func() {
				source := `
					@cpp output "x/cpp/status"

					Status struct<Details?> {
						key string
						details Details?
					}
				`
				loader.Add("schemas/status", source)

				channelSource := `
					import "schemas/status"

					@cpp output "client/cpp/channel"

					ChannelStatus = status.Status<nil>

					Channel struct {
						key uint32
						status ChannelStatus?
					}
				`
				resp := MustGenerate(channelSource, "channel", loader, cppPlugin)

				content := MustContentOf(resp, "client/cpp/channel/types.gen.h")
				Expect(content).To(ContainSubstring(`std::monostate`))
				Expect(content).NotTo(ContainSubstring(`void`))
			})
		})

		It("Should wrap optional scalars in std::optional", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					name string
					task_counter uint32?
					embedded bool?
				}
			`
			table, diag := Analyze(source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// A trailing `?` makes the field optional, wrapped in std::optional.
			Expect(content).To(ContainSubstring(`#include <optional>`))
			Expect(
				content,
			).To(ContainSubstring(`std::optional<std::uint32_t> task_counter;`))
			Expect(content).To(ContainSubstring(`std::optional<bool> embedded;`))
		})

		It("Should use std::optional for optional types", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					parent uint32?
				}
			`
			table, diag := Analyze(source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Only optionals (?) use std::optional in C++
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
			table, diag := Analyze(source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`#include <vector>`))
			Expect(content).To(ContainSubstring(`std::vector<std::string> tags;`))
			Expect(content).To(ContainSubstring(`std::vector<std::int32_t> values;`))
			// Note: vectors don't get = {} default since they have a proper default
			// constructor
		})

		It("Should wrap optional arrays with std::optional", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					tags string[]?
				}
			`
			table, diag := Analyze(source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// A trailing `?` wraps the vector in std::optional.
			Expect(
				content,
			).To(ContainSubstring(`std::optional<std::vector<std::string>> tags;`))
		})

		It("Should wrap optional arrays with std::optional", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					tags string[]?
				}
			`
			table, diag := Analyze(source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Optional array wraps the vector with std::optional
			Expect(
				content,
			).To(ContainSubstring(`std::optional<std::vector<std::string>> tags;`))
		})

		It("Should handle record type", func() {
			source := `
				@cpp output "client/cpp/task"

				Task struct {
					key uint64
					config record
				}
			`
			table, diag := Analyze(source, "task", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`#include "x/cpp/json/json.h"`))
			Expect(content).To(ContainSubstring(`x::json::json::object_t config;`))
		})

		It("Should handle map types", func() {
			source := `
				@cpp output "client/cpp/task"

				Task struct {
					key uint64
					metadata map<string, string>
				}
			`
			table, diag := Analyze(source, "task", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`#include <unordered_map>`))
			Expect(
				content,
			).To(ContainSubstring(`std::unordered_map<std::string, std::string> metadata;`))
		})

		It(
			"Should handle struct extension with field flattening",
			func() {
				source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					name string
					task_counter uint32?
				}

				New struct extends Rack {
					key uint32?
					-task_counter
				}
			`
				table, diag := Analyze(source, "rack", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// New struct should have flattened fields with key optional and
				// task_counter omitted
				Expect(content).To(ContainSubstring(`struct New {`))
				Expect(
					content,
				).To(ContainSubstring(`std::optional<std::uint32_t> key;`))
				Expect(content).To(ContainSubstring(`std::string name;`))
				// task_counter should be omitted from New
				Expect(content).NotTo(MatchRegexp(`struct New \{[^}]*task_counter`))
			},
		)

		It(
			"Should keep inheriting when a field restates only a default",
			func() {
				source := `
				@cpp output "client/cpp/opc"

				BaseRead struct {
					sample_rate float64 = 10
					stream_rate float64 = 5
				}

				ReadConfig struct extends BaseRead {
					sample_rate = 50
					stream_rate = 25
					device      string = ""
				}
			`
				resp := MustGenerate(source, "opc", loader, cppPlugin)
				content := MustContentOf(resp, "types.gen.h")
				Expect(content).To(ContainSubstring(
					`struct ReadConfig : public BaseRead {`,
				))
				Expect(content).To(ContainSubstring(
					"ReadConfig() {\n        this->sample_rate = 50;\n" +
						"        this->stream_rate = 25;\n    }",
				))
				// The base already declares both rates; redeclaring them would hide
				// its members from every reference held through a BaseRead.
				Expect(content).NotTo(MatchRegexp(
					`struct ReadConfig : public BaseRead \{[^}]*double sample_rate`,
				))
			},
		)

		It(
			"Should flatten when a field restates an inherited type",
			func() {
				source := `
				@cpp output "client/cpp/opc"

				Base struct {
					port string = ""
					name string = ""
				}

				Child struct extends Base {
					port int32 = 4
				}
			`
				resp := MustGenerate(source, "opc", loader, cppPlugin)
				content := MustContentOf(resp, "types.gen.h")
				// C++ cannot restate an inherited member's type, so Child flattens.
				Expect(content).To(ContainSubstring("struct Child {"))
				Expect(content).To(ContainSubstring(`std::int32_t port = 4;`))
				Expect(content).To(ContainSubstring(`std::string name = "";`))
			},
		)

		Context("parent omitted in C++", func() {
			schema := func(parentDomain, childBody string) string {
				return `
				@go output "core/pkg/service/auth"
				@cpp output "client/cpp/auth"

				Record struct {
					key    uint32
					name   string
					secret string
					` + parentDomain + `
				}

				Shape struct extends Record {
					` + childBody + `
				}
			`
			}

			It(
				"Should declare an inline variant's inherited fields",
				func() {
					source := `
					@go output "core/pkg/service/auth"
					@cpp output "client/cpp/auth"

					Base struct {
						port string
					}

					Record struct {
						key  uint32
						name string
						@cpp omit
					}

					Chan union on type extends Base {
						ai extends Record {
							rate float64
						}
					}
				`
					resp := MustGenerate(source, "auth", loader, cppPlugin)
					ExpectContent(resp, "types.gen.h").
						ToContain(
							"struct AIChan : public Base {",
							"std::uint32_t key",
							"std::string name",
							"double rate",
						).
						ToNotContain("Record")
				},
			)

			It("Should declare the inherited fields", func() {
				resp := MustGenerate(
					schema("@cpp omit", "rate float64"), "auth", loader, cppPlugin,
				)
				ExpectContent(resp, "types.gen.h").
					ToContain(
						"struct Shape {",
						"std::uint32_t key",
						"std::string name",
						"std::string secret",
						"double rate",
					).
					ToNotContain("struct Record")
			})

			It("Should leave out a removed field", func() {
				resp := MustGenerate(
					schema("@cpp omit", "-secret"), "auth", loader, cppPlugin,
				)
				ExpectContent(resp, "types.gen.h").
					ToContain("struct Shape {", "std::string name").
					ToNotContain("secret", "struct Record")
			})

			It("Should inherit from a parent that generates", func() {
				resp := MustGenerate(
					schema("", "rate float64"), "auth", loader, cppPlugin,
				)
				ExpectContent(resp, "types.gen.h").
					ToContain("struct Shape : public Record {")
			})
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
			table, diag := Analyze(source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`struct RackPayload {`))
			Expect(content).NotTo(ContainSubstring(`struct Rack {`))
		})

		It("Should handle @cpp name override for fields", func() {
			// This tests renaming a field, e.g., when the field name is a C++ reserved
			// keyword
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
			table, diag := Analyze(source, "channel", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// The field should be renamed to is_virtual instead of virtual
			Expect(content).To(ContainSubstring(`bool is_virtual = false;`))
			// The original snake_case name should NOT appear
			Expect(content).NotTo(ContainSubstring(`bool virtual`))
		})

		It(
			"Should automatically escape C++ reserved keyword field names",
			func() {
				source := `
				@cpp output "arc/cpp/ir"

				Authorities struct {
					default uint8 { @optional }
					channels Map<uint32, uint8>
				}
			`
				table, diag := Analyze(source, "ir", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`default_`))
				Expect(
					content,
				).NotTo(MatchRegexp(`\bstd::optional<std::uint8_t> default[^_]`))
			},
		)

		It("Should handle @cpp hand", func() {
			source := `
				@cpp output "client/cpp/rack"

				Rack struct {
					key uint32
					name string
				}

				Internal struct {
					data string

					@cpp hand
				}
			`
			table, diag := Analyze(source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

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
			table, diag := Analyze(source, "status", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`template <typename D>`))
			Expect(content).To(ContainSubstring(`struct Status {`))
			Expect(content).To(ContainSubstring(`D details;`))
		})

		It(
			"Should generate generic struct with type parameter field",
			func() {
				source := `
				@cpp output "client/cpp/status"

				Status struct<D> {
					key uint32
					details D
				}
			`
				table, diag := Analyze(source, "status", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`template <typename D>`))
				Expect(content).To(ContainSubstring(`struct Status {`))
				Expect(content).To(ContainSubstring(`D details;`))
				Expect(content).To(ContainSubstring(`#include <type_traits>`))
			},
		)

		It(
			"Should generate method declarations for generic struct",
			func() {
				source := `
				@cpp output "client/cpp/status"

				Status struct<D> {
					key uint32
					details D
				}
			`
				table, diag := Analyze(source, "status", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`static Status parse(x::json::Parser parser);`))
				Expect(
					content,
				).To(ContainSubstring(`[[nodiscard]] x::json::json to_json() const;`))
			},
		)

		It("Should handle optional generic fields", func() {
			source := `
				@cpp output "client/cpp/status"

				Status struct<D?> {
					key uint32
					details D?
				}
			`
			table, diag := Analyze(source, "status", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

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
			table, diag := Analyze(source, "status", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

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

					@cpp hand
				}

				GoStatus struct<Details?> extends Status<Details, Variant> {
					variant Variant
					-variant

					@cpp name "Status"
				}
			`
			table, diag := Analyze(source, "status", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String)

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(
				content,
			).To(ContainSubstring(`template <typename Details = std::monostate>`))
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
			table, diag := Analyze(source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

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
			table, diag := Analyze(source, "module", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

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
			table, diag := Analyze(source, "types", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(cppPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Array distinct types should generate wrapper structs, not using aliases
			Expect(
				content,
			).To(ContainSubstring(`struct Params : private std::vector<Param>`))
			Expect(content).NotTo(ContainSubstring(`using Params = void;`))
		})

		It(
			"Should handle array wrapper of structs used in other structs",
			func() {
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
				table, diag := Analyze(source, "types", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Array distinct types should generate wrapper structs
				Expect(
					content,
				).To(ContainSubstring(`struct Params : private std::vector<Param>`))
				Expect(content).To(ContainSubstring(`Params inputs;`))
				Expect(content).To(ContainSubstring(`Params outputs;`))
			},
		)

		It(
			"Should order array wrapper after its dependency struct",
			func() {
				source := `
				@cpp output "client/cpp/types"

				Param struct {
					name string
					value int32
				}

				Params Param[]
			`
				table, diag := Analyze(source, "types", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Param struct definition must appear before Params wrapper
				paramDefIdx := strings.Index(content, "struct Param {")
				paramsIdx := strings.Index(content, "struct Params : private")
				Expect(paramDefIdx).To(BeNumerically("<", paramsIdx),
					"Param struct definition should be declared before Params wrapper")
			},
		)

		It(
			"Should generate forward declarations for structs before array wrappers",
			func() {
				source := `
				@cpp output "client/cpp/types"

				Param struct {
					name string
					value int32
				}

				Params Param[]
			`
				table, diag := Analyze(source, "types", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Forward declaration must appear before array wrapper
				fwdDeclIdx := strings.Index(content, "struct Param;")
				wrapperIdx := strings.Index(content, "struct Params : private")
				Expect(
					fwdDeclIdx,
				).To(BeNumerically(">", -1), "Forward declaration should exist")
				Expect(fwdDeclIdx).To(BeNumerically("<", wrapperIdx),
					"Forward declaration should appear before array wrapper")
			},
		)

		It(
			"Should handle cyclic dependencies with forward declarations",
			func() {
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
				table, diag := Analyze(source, "types", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// All structs should have forward declarations
				Expect(content).To(ContainSubstring("struct FunctionProperties;"))
				Expect(content).To(ContainSubstring("struct Type;"))
				Expect(content).To(ContainSubstring("struct Param;"))
				// Array wrapper should generate a struct, not a using alias
				Expect(
					content,
				).To(ContainSubstring("struct Params : private std::vector<Param>"))
			},
		)

		It(
			"Should handle int enums with uint8_t underlying type",
			func() {
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
				table, diag := Analyze(source, "status", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`enum class Variant : std::uint8_t {`))
				Expect(content).To(ContainSubstring(`Success = 0,`))
				Expect(content).To(ContainSubstring(`Error = 1,`))
				Expect(content).To(ContainSubstring(`Warning = 2,`))
			},
		)

		It(
			"Should emit an extending enum as the union of its parents",
			func() {
				source := `
				@cpp output "client/cpp/lineplot"

				XAxisKey enum {
					x1 = "x1"
					x2 = "x2"
				}

				YAxisKey enum {
					y1 = "y1"
					y2 = "y2"
				}

				AxisKey enum extends XAxisKey, YAxisKey {}

				Plot struct {
					axis_key AxisKey
				}
			`
				table, diag := Analyze(source, "lineplot", loader)
				Expect(diag.Ok()).To(BeTrue())

				resp := MustSucceed(
					cppPlugin.Generate(&plugin.Request{Resolutions: table}),
				)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`AXIS_KEY_X_1 = "x1";`))
				Expect(content).To(ContainSubstring(`AXIS_KEY_Y_2 = "y2";`))
			},
		)

		It(
			"Should use indirect for self-referential optional fields",
			func() {
				source := `
				@cpp output "client/cpp/types"

				Node struct {
					name string
					left Node?
					right Node?
				}
			`
				table, diag := Analyze(source, "types", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Self-referential fields should use x::mem::indirect, not optional
				Expect(content).To(ContainSubstring(`x::mem::indirect<Node> left;`))
				Expect(content).To(ContainSubstring(`x::mem::indirect<Node> right;`))
				Expect(content).To(ContainSubstring(`#include "x/cpp/mem/indirect.h"`))
				// Should NOT use std::optional for self-referential types
				Expect(content).NotTo(ContainSubstring(`std::optional<Node>`))
			},
		)

		It(
			"Should use indirect for mutually recursive optional fields",
			func() {
				source := `
				@cpp output "client/cpp/types"

				A struct {
					b B?
				}
				B struct {
					a A?
				}
			`
				resp := MustGenerate(source, "types", loader, cppPlugin)
				ExpectContent(resp, "types.gen.h").
					ToContain(
						`x::mem::indirect<B> b;`,
						`x::mem::indirect<A> a;`,
					).
					ToNotContain(
						`std::optional<A>`,
						`std::optional<B>`,
					)
			},
		)

		It(
			"Should use indirect for cycles through array wrappers",
			func() {
				source := `
				@cpp output "client/cpp/types"

				Node struct {
					children Node[]
					parent Parent?
				}
				Parent struct {
					nodes Node[]
				}
			`
				resp := MustGenerate(source, "types", loader, cppPlugin)
				ExpectContent(resp, "types.gen.h").
					ToContain(`x::mem::indirect<Parent> parent;`).
					ToNotContain(`std::optional<Parent>`)
			},
		)

		It(
			"Should use indirect for cycles through a distinct struct wrapper",
			func() {
				source := `
				@cpp output "client/cpp/types"

				A struct {
					b BWrap?
				}
				B struct {
					a A?
				}
				BWrap B
			`
				resp := MustGenerate(source, "types", loader, cppPlugin)
				ExpectContent(resp, "types.gen.h").
					ToContain(
						`x::mem::indirect<BWrap> b;`,
						`x::mem::indirect<A> a;`,
					).
					ToNotContain(
						`std::optional<BWrap>`,
						`std::optional<A>`,
					)
			},
		)

		It(
			"Should use optional for non-self-referential optional fields",
			func() {
				source := `
				@cpp output "client/cpp/types"

				Unit struct {
					name string
					scale float64
				}

				Type struct {
					name string
					unit Unit?
				}
			`
				table, diag := Analyze(source, "types", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Non-self-referential optional fields should use std::optional
				Expect(content).To(ContainSubstring(`std::optional<Unit> unit;`))
				Expect(content).To(ContainSubstring(`#include <optional>`))
			},
		)

		It(
			"Should handle cross-namespace references to handwritten types",
			func() {
				// First, set up the "status" namespace with an omitted type
				statusSource := `
				@cpp output "x/cpp/status"
				@cpp hand
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
					status RackStatus?
				}
			`
				table, diag := Analyze(rackSource, "rack", loader)
				Expect(diag.Ok()).To(BeTrue(), diag.String())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))
				Expect(resp.Files).To(HaveLen(1))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`#include "x/cpp/status/status.h"`))
				Expect(
					content,
				).To(ContainSubstring(`using RackStatus = ::x::status::Status<StatusDetails>;`))
				Expect(
					content,
				).To(ContainSubstring(`std::optional<RackStatus> status;`))
			},
		)

		It(
			"Should not generate files for structs without @cpp output",
			func() {
				source := `
				@go output "core/pkg/service/user"

				User struct {
					key uint32
					name string
				}
			`
				table, diag := Analyze(source, "user", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))
				Expect(resp.Files).To(BeEmpty())
			},
		)

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
				table, diag := Analyze(source, "animals", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

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
				table, diag := Analyze(source, "order", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(cppPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Fields now have default values, so match the field names in
				// declaration context
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
			It(
				"Should include standalone enums in the same namespace as structs",
				func() {
					// This tests the fix for enums that are not referenced by struct
					// fields but are in the same namespace and should be generated in
					// the same file.
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
					resp := MustGenerate(source, "control", loader, cppPlugin)

					// The Concurrency enum should be generated even though Subject
					// doesn't reference it - they're in the same namespace
					ExpectContent(resp, "types.gen.h").
						ToContain(
							"namespace x::control {",
							"enum class Concurrency : std::uint8_t {",
							"Exclusive = 0,",
							"Shared = 1,",
							"struct Subject {",
						)
				},
			)

			It(
				"Should include enums inherited from file-level output directive",
				func() {
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
					resp := MustGenerate(source, "status", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(
							"enum class Variant : std::uint8_t {",
							"struct Status {",
						)
				},
			)

			It("Should include cstdint for a standalone int enum", func() {
				source := `
					@cpp output "x/cpp/telem"

					Code enum {
						unknown = 0
						uint8 = 1
					}
				`
				resp := MustGenerate(source, "telem", loader, cppPlugin)

				ExpectContent(resp, "types.gen.h").
					ToContain(
						"#include <cstdint>",
						"enum class Code : std::uint8_t {",
					)
			})

			It("Should add the headers that an enum's includes name", func() {
				source := `
					@cpp output "x/cpp/telem"

					Code enum {
						unknown = 0

						@cpp includes "x/cpp/telem/telem.h"
						@cpp system_includes "string"
					}
				`
				resp := MustGenerate(source, "telem", loader, cppPlugin)

				ExpectContent(resp, "types.gen.h").
					ToContain(
						"#include <string>",
						`#include "x/cpp/telem/telem.h"`,
						"enum class Code : std::uint8_t {",
					)
			})
		})

		Describe("Cross-namespace enum references", func() {
			It(
				"Should use namespace-qualified name for cross-namespace int enums",
				func() {
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
					table, diag := Analyze(
						channelSource,
						"channel",
						loader,
					)
					Expect(diag.Ok()).To(BeTrue())

					req := &plugin.Request{
						Resolutions: table,
					}

					resp := MustSucceed(cppPlugin.Generate(req))
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
					Expect(
						channelContent,
					).To(ContainSubstring(`#include "x/cpp/control/types.gen.h"`))
					// Should use namespace-qualified enum type
					Expect(
						channelContent,
					).To(ContainSubstring(`::x::control::Concurrency concurrency`))
				},
			)

			It(
				"Should not namespace-qualify enums in the same namespace",
				func() {
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
					resp := MustGenerate(source, "channel", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(
							"enum class Concurrency : std::uint8_t {",
							"Concurrency concurrency",
						).
						ToNotContain(
							"::synnax::channel::Concurrency",
						)
				},
			)
		})

		Describe("Explicit @cpp include", func() {
			It(
				"Should include the declared header for a cross-namespace distinct type",
				func() {
					loader.Add("schemas/rack", `
					@cpp output "client/cpp/rack"

					Key uint32 {
						@cpp include "client/cpp/rack/key.h"
					}
				`)

					source := `
					import "schemas/rack"

					@cpp output "client/cpp/task"

					Task struct {
						rack rack.Key
					}
				`
					resp := MustGenerate(source, "task", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(`#include "client/cpp/rack/key.h"`).
						ToNotContain(`#include "client/cpp/rack/types.gen.h"`)
				},
			)

			It(
				"Should include the declared header for a cross-namespace alias",
				func() {
					loader.Add("schemas/status", `
					@cpp output "client/cpp/status"

					Detail struct {
						message string
					}

					Summary = Detail {
						@cpp include "client/cpp/status/summary.h"
					}
				`)

					source := `
					import "schemas/status"

					@cpp output "client/cpp/task"

					Task struct {
						summary status.Summary
					}
				`
					resp := MustGenerate(source, "task", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(`#include "client/cpp/status/summary.h"`).
						ToNotContain(`#include "client/cpp/status/types.gen.h"`)
				},
			)

			It(
				"Should fall back to types.gen.h when no header is declared",
				func() {
					loader.Add("schemas/rack", `
					@cpp output "client/cpp/rack"

					Key uint32
				`)

					source := `
					import "schemas/rack"

					@cpp output "client/cpp/task"

					Task struct {
						rack rack.Key
					}
				`
					resp := MustGenerate(source, "task", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(`#include "client/cpp/rack/types.gen.h"`)
				},
			)

			It(
				"Should include the declared header for a same-namespace hand struct",
				func() {
					source := `
					@cpp output "x/cpp/color"

					Color struct {
						r uint8

						@cpp hand
						@cpp include "x/cpp/color/color.h"
					}

					Stop struct {
						color Color
					}
				`
					resp := MustGenerate(source, "color", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(`#include "x/cpp/color/color.h"`)
				},
			)

			It(
				"Should not include a header for a same-namespace generated struct",
				func() {
					source := `
					@cpp output "x/cpp/color"

					Color struct {
						r uint8

						@cpp include "x/cpp/color/color.h"
					}

					Stop struct {
						color Color
					}
				`
					resp := MustGenerate(source, "color", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToNotContain(`#include "x/cpp/color/color.h"`)
				},
			)
		})

		Describe("Array Wrapper Generation", func() {
			It(
				"Should generate wrapper struct for array distinct types",
				func() {
					source := `
					@cpp output "arc/cpp/types"

					Channels uint32[]
				`
					resp := MustGenerate(source, "types", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
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
				},
			)

			It(
				"Should generate wrapper struct for array of structs",
				func() {
					source := `
					@cpp output "arc/cpp/types"

					Param struct {
						name string
						dataType string
					}

					Params Param[]
				`
					resp := MustGenerate(source, "types", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(
							"struct Params : private std::vector<Param>",
							"using Base = std::vector<Param>;",
						)
				},
			)

			It(
				"Should generate parse/to_json declarations for array wrappers",
				func() {
					source := `
					@cpp output "arc/cpp/types"

					Channels uint32[]
				`
					resp := MustGenerate(source, "types", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(
							"static Channels parse(x::json::Parser parser);",
							"[[nodiscard]] x::json::json to_json() const;",
						)
				},
			)

			// Note: Proto declarations for array wrappers are tested in the pb plugin
			// tests. The types plugin generates declarations when HasProto is set,
			// which requires explicit @pb annotations on the type (tested in pb
			// plugin).

			It(
				"Should support @cpp methods for custom methods on array wrappers",
				func() {
					source := `
					@cpp output "arc/cpp/types"
					@cpp methods "std::optional<Param> get(const std::string& name) const"

					Param struct {
						name string
					}

					Params Param[]
				`
					resp := MustGenerate(source, "types", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(
							"std::optional<Param> get(const std::string& name) const;",
						)
				},
			)

			It(
				"Should generate initializer_list constructor for array wrappers",
				func() {
					source := `
					@cpp output "arc/cpp/types"

					Channels uint32[]
				`
					resp := MustGenerate(source, "types", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(
							"Channels(std::initializer_list<std::uint32_t> init) : Base(init) {}",
						)
				},
			)
		})

		Context("documentation", func() {
			It(
				"Should generate doxygen comments from doc domain",
				func() {
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
					resp := MustGenerate(source, "user", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(
							"/// @brief User is a representation of a user in the system.",
							"/// @brief key is the unique identifier for the user.",
							"/// @brief name is the user's display name.",
						)
				},
			)

			It(
				"Should generate doxygen comments for type definitions",
				func() {
					source := `
					@cpp output "client/cpp/node"

					Key uint12 {
						@doc value "is a 12-bit unsigned integer identifying a node."
					}

					Keys Key[] {
						@doc value "is a list of node keys."
					}
				`
					resp := MustGenerate(source, "node", loader, cppPlugin)

					ExpectContent(resp, "types.gen.h").
						ToContain(
							"/// @brief Key is a 12-bit unsigned integer identifying a node.\nusing Key = std::uint16_t;",
							"/// @brief Keys is a list of node keys.\nstruct Keys : private std::vector<Key> {",
						)
				},
			)
		})

		Context("enum variant defaults", func() {
			It(
				"Should generate default for same-namespace enum variant",
				func() {
					source := `
					@cpp output "out"
					@pb

					Mode enum {
						automatic = 0
						manual    = 1
					}

					Config struct {
						mode Mode = ModeAutomatic
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					ExpectContent(resp, "types.gen.h").
						ToContain(`Mode mode = Mode::Automatic`)
				},
			)

			It(
				"Should zero-initialize an int enum field with no default",
				func() {
					source := `
					@cpp output "out"

					Mode enum {
						automatic = 1
						manual    = 2
					}

					Config struct {
						mode     Mode
						override Mode?
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					ExpectContent(resp, "types.gen.h").
						ToContain(`Mode mode = {};`, `std::optional<Mode> override;`)
				},
			)

			It(
				"Should reference string enum defaults via their generated constants",
				func() {
					source := `
					@cpp output "out"

					Units enum {
						volts = "Volts"
						amps  = "Amps"
					}

					Config struct {
						units Units = volts
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					content := ExpectContent(resp, "types.gen.h")
					content.ToContain(`units = UNITS_VOLTS`)
					content.ToNotContain(`Units::Volts`)
				},
			)

			It(
				"Should emit scalar field defaults as member initializers",
				func() {
					source := `
					@cpp output "out"

					Config struct {
						enabled     bool = false
						sample_rate float64 = 10
						label       string = "dflt"
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					ExpectContent(resp, "types.gen.h").
						ToContain(
							`bool enabled = false;`,
							`double sample_rate = 10;`,
							`std::string label = "dflt";`,
						)
				},
			)

			It(
				"Should wrap numeric defaults on telem-typed fields in their constructors",
				func() {
					loader.Add("schemas/telem", `
					@cpp output "x/cpp/telem"

					TimeStamp int64 {
						@cpp hand
					}

					TimeSpan int64 {
						@cpp hand
					}

					Rate float64 {
						@cpp hand
					}
				`)
					source := `
					import "schemas/telem"

					@cpp output "out"

					Config struct {
						duration    telem.TimeSpan = 0
						start       telem.TimeStamp = 5
						sample_rate telem.Rate = 10
						stream_rate telem.Rate = 2.5
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					ExpectContent(resp, "out/types.gen.h").
						ToContain(
							`duration = ::x::telem::TimeSpan(0);`,
							`start = ::x::telem::TimeStamp(5);`,
							`sample_rate = ::x::telem::Rate(10);`,
							`stream_rate = ::x::telem::Rate(2.500000);`,
						)
				},
			)

			It(
				"Should leave an optional telem-typed field without a value",
				func() {
					loader.Add("schemas/telem", `
					@cpp output "x/cpp/telem"

					TimeStamp int64 {
						@cpp hand
					}
				`)
					source := `
					import "schemas/telem"

					@cpp output "out"

					Config struct {
						start telem.TimeStamp?
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					ExpectContent(resp, "out/types.gen.h").
						ToContain("std::optional<::x::telem::TimeStamp> start;")
				},
			)

			It(
				"Should wrap string defaults in the distinct type's constructor",
				func() {
					loader.Add("schemas/telem", `
					@cpp output "x/cpp/telem"

					DataType string {
						@cpp hand
					}
				`)
					source := `
					import "schemas/telem"

					@cpp output "out"

					Config struct {
						data_type telem.DataType = "float32"
						label     string = "dflt"
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					ExpectContent(resp, "out/types.gen.h").
						ToContain(
							`data_type = ::x::telem::DataType("float32");`,
							`std::string label = "dflt";`,
						)
				},
			)

			It(
				"Should wrap numeric defaults on distinct types outside x::telem",
				func() {
					loader.Add("schemas/units", `
					@cpp output "x/cpp/units"

					Voltage float64 {
						@cpp hand
					}

					Count uint32 {
						@cpp hand
					}
				`)
					source := `
					import "schemas/units"

					@cpp output "out"

					Config struct {
						limit   units.Voltage = 5.5
						retries units.Count = 3
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					ExpectContent(resp, "out/types.gen.h").
						ToContain(
							`limit = ::x::units::Voltage(5.500000);`,
							`retries = ::x::units::Count(3);`,
						)
				},
			)

			It(
				"Should map the now sentinel to TimeStamp::now and skip unrenderable sentinels",
				func() {
					loader.Add("schemas/telem", `
					@cpp output "x/cpp/telem"

					TimeStamp int64 {
						@cpp hand
					}
				`)
					source := `
					import "schemas/telem"

					@cpp output "out"

					Status struct {
						key  string = create
						time telem.TimeStamp = now
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					content := ExpectContent(resp, "out/types.gen.h")
					content.ToContain(
						`std::string key;`,
						`time = x::telem::TimeStamp::now();`,
					)
					content.ToNotContain(`= create;`, `= now;`)
				},
			)

			It(
				"Should include the header of a cross-namespace string enum default",
				func() {
					loader.Add("schemas/wiring", `
					@cpp output "client/cpp/wiring"

					Mode enum {
						rse  = "RSE"
						diff = "Diff"
					}
				`)
					source := `
					import "schemas/wiring"

					@cpp output "client/cpp/channel"

					Channel struct {
						mode wiring.Mode = rse
					}
				`
					resp := MustGenerate(source, "channel", loader, cppPlugin)
					ExpectContent(resp, "client/cpp/channel/types.gen.h").ToContain(
						`#include "client/cpp/wiring/types.gen.h"`,
						`std::string mode = ::synnax::wiring::MODE_RSE;`,
					)
				},
			)

			It(
				"Should skip the header of a cross-namespace string enum with no default",
				func() {
					loader.Add("schemas/wiring", `
					@cpp output "client/cpp/wiring"

					Mode enum {
						rse  = "RSE"
						diff = "Diff"
					}
				`)
					source := `
					import "schemas/wiring"

					@cpp output "client/cpp/channel"

					Channel struct {
						mode wiring.Mode
					}
				`
					resp := MustGenerate(source, "channel", loader, cppPlugin)
					ExpectContent(resp, "client/cpp/channel/types.gen.h").
						ToContain(`std::string mode;`).
						ToNotContain(`#include "client/cpp/wiring/types.gen.h"`)
				},
			)

			It(
				"Should generate default for cross-namespace enum variant",
				func() {
					loader.Add("schemas/control", `
					@cpp output "x/cpp/control"
					@pb

					Concurrency enum {
						exclusive = 0
						shared    = 1
					}
				`)
					source := `
					import "schemas/control"

					@cpp output "client/cpp/channel"
					@pb

					Channel struct {
						concurrency control.Concurrency = control.ConcurrencyExclusive
					}
				`
					resp := MustGenerate(source, "channel", loader, cppPlugin)
					ExpectContent(resp, "types.gen.h").
						ToContain(`::x::control::Concurrency concurrency = ::x::control::Concurrency::Exclusive`)
				},
			)

			It("Should generate brace-init defaults for arrays", func() {
				source := `
					@cpp output "out"

					Config struct {
						empty float64[] = []
						vals  float64[] = [1.5, 2.5]
					}
				`
				resp := MustGenerate(source, "config", loader, cppPlugin)
				content := MustContentOf(resp, "types.gen.h")
				Expect(content).To(ContainSubstring(`empty = {}`))
				Expect(content).To(ContainSubstring(`vals = {1.500000, 2.500000}`))
			})

			It(
				"Should generate designated-initializer defaults for structs",
				func() {
					source := `
					@cpp output "out"

					Point struct {
						x int32
						y int32
					}

					Config struct {
						p Point = { x = 1, y = 2 }
					}
				`
					resp := MustGenerate(source, "config", loader, cppPlugin)
					content := MustContentOf(resp, "types.gen.h")
					Expect(content).To(ContainSubstring(`p = {.x = 1, .y = 2}`))
				},
			)
		})
	})
})

var _ = Describe("C++ Union Generation", func() {
	var (
		loader    *MockFileLoader
		cppPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		cppPlugin = types.New(types.DefaultOptions())
	})

	It(
		"Should generate variant structs and a std::variant alias",
		func() {
			source := `
			@cpp output "out"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams

				@doc value "determines how raw values are transformed."
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain(
					`struct LinearScale : public LinearParams {`,
					`std::string type = "linear";`,
					`struct NoneScale : public NoneParams {`,
					`std::string type = "none";`,
					"/// @brief Scale determines how raw values are transformed.\nusing Scale = std::variant<LinearScale, NoneScale>;",
					`Scale parse_scale(x::json::Parser parser);`,
					`[[nodiscard]] x::json::json to_json(const Scale& value);`,
				)
		},
	)

	It("Should emit resource helpers for a union", func() {
		source := `
			@cpp output "out"

			Key uuid

			Base struct {
				key Key @key
			}

			View union on type extends Base {
				first {
					name string
				}
				second {
					size int32
				}

				@resource type "view"
			}
		`
		resp := MustGenerate(source, "view", loader, cppPlugin)
		ExpectContent(resp, "types.gen.h").
			ToContain(
				`const synnax::resource::ID TYPE_RESOURCE_ID("view", "");`,
				`return synnax::resource::ID("view", key.to_string());`,
			)
	})

	It("Should type union resource helpers by a primitive key", func() {
		source := `
			@cpp output "out"

			Base struct {
				key uuid @key
			}

			View union on type extends Base {
				first {
					name string
				}

				@resource type "view"
			}
		`
		resp := MustGenerate(source, "view", loader, cppPlugin)
		ExpectContent(resp, "types.gen.h").
			ToContain(`resource_id(const x::uuid::UUID& key)`)
	})

	It(
		"Should default the discriminator to the borrowed enum constant",
		func() {
			source := `
			@cpp output "out"

			Method enum {
				password = "password"
			}
			Credentials struct { username string }

			Proof union on method Method {
				password Credentials
			}
		`
			resp := MustGenerate(source, "auth", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain(
					`constexpr const char* METHOD_PASSWORD = "password";`,
					`std::string method = METHOD_PASSWORD;`,
				)
		},
	)

	It("Should include the header of a borrowed enum from another namespace", func() {
		loader.Add("schemas/auth", `
			@cpp output "client/cpp/auth"

			Method enum {
				password = "password"
			}
		`)
		source := `
			import "schemas/auth"

			@cpp output "client/cpp/login"

			Credentials struct { username string }

			Proof union on method auth.Method {
				password Credentials
			}
		`
		resp := MustGenerate(source, "login", loader, cppPlugin)
		ExpectContent(resp, "types.gen.h").
			ToContain(`#include "client/cpp/auth/types.gen.h"`, `METHOD_PASSWORD;`)
	})

	It(
		"Should declare inline variant fields directly on the variant struct",
		func() {
			source := `
			@cpp output "out"

			TabBase struct { key string }
			Labeled struct { label string }

			Tab union on variant extends TabBase {
				view extends Labeled {
					type string
				}
				empty {}
			}
		`
			resp := MustGenerate(source, "panel", loader, cppPlugin)
			content := ExpectContent(resp, "types.gen.h")
			content.ToContain(
				`struct ViewTab : public TabBase, public Labeled {`,
				`std::string type;`,
				`struct EmptyTab : public TabBase {`,
				`using Tab = std::variant<ViewTab, EmptyTab>;`,
			)
			content.ToNotContain("ViewTabPayload")
		},
	)

	It(
		"Should omit the base list for a variant whose union and payload have no extends",
		func() {
			source := `
			@cpp output "out"

			Other struct { x int32 }

			DIChannel union on type {
				digital_input {
					port int32
				}
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			content := ExpectContent(resp, "types.gen.h")
			content.ToContain(`struct DigitalInputDIChannel {`)
			content.ToNotContain(`struct DigitalInputDIChannel : {`)
		},
	)

	It(
		"Should flatten a base a variant omits a field from",
		func() {
			source := `
			@cpp output "out"

			BaseAIChan struct {
				port int32
				enabled bool
			}

			AIChannel union on type extends BaseAIChan {
				ai_voltage { minVal float64 }
				ai_temp_builtin {
					-port
					units string
				}
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			content := MustContentOf(resp, "types.gen.h")
			Expect(content).To(ContainSubstring("struct AITempBuiltinChannel {"))
			Expect(content).
				To(ContainSubstring("struct AIVoltageChannel : public BaseAIChan {"))
		},
	)

	It(
		"Should inherit the union base and payload in every variant struct, not flatten",
		func() {
			source := `
			@cpp output "out"

			BaseAIChan struct { port int32 }
			VoltageFields struct { minVal float64 }

			AIChannel union on type extends BaseAIChan {
				ai_voltage VoltageFields
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain(
					`struct AIVoltageChannel : public BaseAIChan, public VoltageFields {`,
					`std::string type = "ai_voltage";`,
					`std::int32_t port = 0;`,
					`double min_val = 0;`,
					`using AIChannel = std::variant<AIVoltageChannel>;`,
				)
		},
	)

	It(
		"Should emit a base accessor for a union whose variants share one base",
		func() {
			source := `
			@cpp output "out"

			BaseAIChan struct { port int32 }
			VoltageFields struct { minVal float64 }

			AIChannel union on type extends BaseAIChan {
				ai_voltage VoltageFields
				ai_current { units string }
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain(
					"[[nodiscard]] inline const BaseAIChan& " +
						"base(const AIChannel& value) {\n" +
						"    return std::visit([](const auto& v) -> const BaseAIChan& " +
						"{ return v; }, value);\n}",
				).
				ToNotContain("base(AIChannel& value)")
		},
	)

	It(
		"Should qualify the base accessor type of a base from another namespace",
		func() {
			loader.Add("schemas/common", `
			@cpp output "x/cpp/common"

			BaseAIChan struct { port int32 }
		`)
			source := `
			import "schemas/common"

			@cpp output "client/cpp/ni"

			VoltageFields struct { minVal float64 }

			AIChannel union on type extends common.BaseAIChan {
				ai_voltage VoltageFields
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "ni/types.gen.h").
				ToContain(
					"inline const ::x::common::BaseAIChan& " +
						"base(const AIChannel& value)",
				)
		},
	)

	DescribeTable(
		"Should not emit a base accessor without one base every variant shares",
		func(source string) {
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain("using AIChannel = std::variant<").
				ToNotContain(" base(")
		},
		Entry("no base", `
			@cpp output "out"

			VoltageFields struct { minVal float64 }

			AIChannel union on type {
				ai_voltage VoltageFields
			}
		`),
		Entry("two bases", `
			@cpp output "out"

			BaseAIChan struct { port int32 }
			Range struct { minVal float64 }

			AIChannel union on type extends BaseAIChan, Range {
				ai_voltage {}
			}
		`),
		Entry("a variant that omits a base field", `
			@cpp output "out"

			BaseAIChan struct {
				port int32
				enabled bool
			}

			AIChannel union on type extends BaseAIChan {
				ai_voltage { minVal float64 }
				ai_temp_builtin {
					-port
					units string
				}
			}
		`),
	)

	It(
		"Should move a variant's restated base default into a constructor",
		func() {
			source := `
			@cpp output "out"

			BaseInputChannel struct {
				port string = ""
			}

			InputChannel union on type extends BaseInputChannel {
				AI { port string = "AIN0" }
				DI { port string = "DIO4" }
			}
		`
			resp := MustGenerate(source, "labjack", loader, cppPlugin)
			// A derived struct cannot restate a base member's initializer, and
			// declaring the member again would hide the base's from every
			// reference held through a base type.
			ExpectContent(resp, "types.gen.h").
				ToContain(
					`struct AIInputChannel : public BaseInputChannel {`,
					"AIInputChannel() {\n        this->port = \"AIN0\";\n    }",
					"DIInputChannel() {\n        this->port = \"DIO4\";\n    }",
				).
				ToNotContain(`std::string port = "AIN0";`, `std::string port = "DIO4";`)
		},
	)

	It(
		"Should inherit a union base struct from another namespace with include",
		func() {
			loader.Add("schemas/common", `
			@cpp output "x/cpp/common"

			BaseAIChan struct { port int32 }
		`)
			source := `
			import "schemas/common"

			@cpp output "client/cpp/ni"

			VoltageFields struct { minVal float64 }

			AIChannel union on type extends common.BaseAIChan {
				ai_voltage VoltageFields
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "ni/types.gen.h").
				ToContain(
					`#include "x/cpp/common/types.gen.h"`,
					`struct AIVoltageChannel : public ::x::common::BaseAIChan, `+
						`public VoltageFields {`,
					`using AIChannel = std::variant<AIVoltageChannel>;`,
				)
		},
	)

	It(
		"Should resolve a union-typed field to the variant alias",
		func() {
			source := `
			@cpp output "out"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams
			}

			Channel struct {
				customScale Scale
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain(`Scale custom_scale;`)
		},
	)

	It(
		"Should initialize a union-typed field with its defaulted variant",
		func() {
			source := `
			@cpp output "out"

			LinearParams struct { slope float64 = 1 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams
			}

			Channel struct {
				customScale Scale = none
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain(`Scale custom_scale = NoneScale{};`)
		},
	)
})

var _ = Describe("C++ Union Variant Doc Coverage", func() {
	var (
		loader    *MockFileLoader
		cppPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		cppPlugin = types.New(types.DefaultOptions())
	})

	It(
		"Should resolve a union field from an imported namespace",
		func() {
			loader.Add("schemas/scale", `
			@cpp output "client/cpp/scale"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none   NoneParams
			}
		`)
			source := `
			import "schemas/scale"

			@cpp output "client/cpp/ni"

			Channel struct {
				custom_scale scale.Scale
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain(
					`#include "client/cpp/scale/types.gen.h"`,
					"::scale::Scale custom_scale;",
				)
		},
	)

	It(
		"Should render a per-variant doc comment on the variant struct",
		func() {
			source := `
			@cpp output "out"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams {
					@doc value "a linear scale."
				}
				none NoneParams
			}
		`
			resp := MustGenerate(source, "ni", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain("/// @brief LinearScale a linear scale.", "struct LinearScale : public LinearParams {")
		},
	)

	It(
		"Should route per-type cpp output overrides to their own file",
		func() {
			source := `
			@cpp output "client/cpp/task"

			Task struct { key string }

			BaseConfig struct {
				auto_start bool
				@cpp output "client/cpp/task/common"
			}
		`
			resp := MustGenerate(source, "task", loader, cppPlugin)
			taskContent := ExpectContent(resp, "client/cpp/task/types.gen.h")
			taskContent.ToContain("namespace synnax::task {", "struct Task {")
			taskContent.ToNotContain("struct BaseConfig {")
			commonContent := ExpectContent(resp, "client/cpp/task/common/types.gen.h")
			commonContent.ToContain(
				"namespace synnax::task::common {",
				"struct BaseConfig {",
			)
			commonContent.ToNotContain("struct Task {")
		},
	)
	It(
		"Should declare proto translators for a union that emits protobuf",
		func() {
			source := `
			@cpp output "arc/cpp/ir"
			@go output "arc/go/ir"
			@pb

			Target union on type {
				exit {}
				step { key string }
			}

			Transition struct { target Target }
			`
			resp := MustGenerate(source, "ir", loader, cppPlugin)
			ExpectContent(resp, "types.gen.h").
				ToContain(
					"using proto_type = ::arc::ir::pb::TargetExitPayload;",
					"static std::pair<StepTarget, x::errors::Error> "+
						"from_proto(const ::arc::ir::pb::TargetStepPayload& pb);",
					"[[nodiscard]] std::pair<::arc::ir::pb::Target, x::errors::Error> "+
						"to_proto(const Target& value);",
					"std::pair<Target, x::errors::Error> "+
						"target_from_proto(const ::arc::ir::pb::Target& pb);",
					`#include "arc/go/ir/pb/ir.pb.h"`,
				)
		},
	)
	It(
		"Should name a variant's proto message from its payload's package",
		func() {
			loader.Add("schemas/hw", `
			@cpp output "client/cpp/hw"
			@go output "core/pkg/service/hw"
			@pb

			Valve struct { open bool }
			`)
			source := `
			import "schemas/hw"

			@cpp output "arc/cpp/ir"
			@go output "arc/go/ir"
			@pb

			Node union on variant {
				valve hw.Valve
			}

			Graph struct { node Node }
			`
			resp := MustGenerate(source, "ir", loader, cppPlugin)
			ExpectContent(resp, "ir/types.gen.h").ToContain(
				"using proto_type = ::service::hw::pb::Valve;",
				`#include "core/pkg/service/hw/pb/hw.pb.h"`,
			)
		},
	)
})

var _ = ShouldNotLeakGoroutinesPerSpec()
