// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/cpp/json"
	"github.com/synnaxlabs/oracle/testutil"
)

func TestCppJson(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "C++ JSON Plugin Suite")
}

var _ = Describe("C++ JSON Plugin", func() {
	var (
		ctx        context.Context
		loader     *testutil.MockFileLoader
		jsonPlugin *json.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
		jsonPlugin = json.New(json.Options{
			FileNamePattern:  "json.gen.h",
			DisableFormatter: true,
		})
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(jsonPlugin.Name()).To(Equal("cpp/json"))
		})

		It("Should filter on cpp domain", func() {
			Expect(jsonPlugin.Domains()).To(Equal([]string{"cpp"}))
		})

		It("Should require cpp/types", func() {
			Expect(jsonPlugin.Requires()).To(Equal([]string{"cpp/types"}))
		})
	})

	Describe("Generate", func() {
		Context("array alias fields (e.g., Params -> Param[])", func() {
			It("Should generate correct parsing for array alias fields", func() {
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)
				Expect(resp.Files).To(HaveLen(1))

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// Should parse using the wrapper type which has its own parse method
						`parser.field<Params>("inputs")`,
						`parser.field<Params>("outputs")`,
					)
			})

			It("Should generate correct to_json for array alias fields", func() {
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// Wrapper types have their own to_json() method
						`j["inputs"] = this->inputs.to_json()`,
						`j["outputs"] = this->outputs.to_json()`,
					)
			})

			It("Should handle optional array alias fields", func() {
				source := `
					@cpp output "client/cpp/types"

					Param struct {
						name string
						value int32
					}

					Params Param[]

					FunctionProperties struct {
						inputs Params?
						outputs Params?
					}
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// Soft optional (?) wrapper types still use field<> with wrapper type
						`parser.field<Params>("inputs")`,
						`parser.field<Params>("outputs")`,
					)
			})
		})

		Context("self-referential types with indirect<T>", func() {
			It("Should generate correct parsing for self-referential fields", func() {
				source := `
					@cpp output "client/cpp/types"

					Node struct {
						name string
						left Node??
						right Node??
					}
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)
				Expect(resp.Files).To(HaveLen(1))

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// Self-referential fields use x::mem::indirect<T> with implicit nullptr default
						`parser.field<x::mem::indirect<Node>>("left")`,
						`parser.field<x::mem::indirect<Node>>("right")`,
					)
			})

			It("Should generate correct to_json for self-referential fields", func() {
				source := `
					@cpp output "client/cpp/types"

					Node struct {
						name string
						left Node??
						right Node??
					}
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// Should use has_value() and -> operator for indirect<T>
						`if (this->left.has_value()) j["left"] = this->left->to_json()`,
						`if (this->right.has_value()) j["right"] = this->right->to_json()`,
					)
			})

			It("Should handle nested self-references via type arguments", func() {
				source := `
					@cpp output "arc/cpp/types"

					Type struct {
						kind string
						name string
						elem Type??
						constraint Type??
					}
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<x::mem::indirect<Type>>("elem")`,
						`parser.field<x::mem::indirect<Type>>("constraint")`,
					)
			})
		})

		Context("hard optional struct fields (non-self-referential)", func() {
			It("Should generate correct to_json for optional struct fields", func() {
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// Should use has_value() and -> operator for optional<T>
						`if (this->unit.has_value()) j["unit"] = this->unit->to_json()`,
					)
			})

			It("Should generate correct parsing for optional struct fields", func() {
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// Hard optional struct fields use std::optional<T> with implicit nullopt default
						`parser.field<std::optional<Unit>>("unit")`,
					)
			})
		})

		Context("map type handling", func() {
			It("Should use std::unordered_map for Map types in json serialization", func() {
				source := `
					@cpp output "client/cpp/task"

					Task struct {
						key uint64
						metadata map<string, string>
					}
				`
				resp := testutil.MustGenerate(ctx, source, "task", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<std::unordered_map<std::string, std::string>>("metadata")`,
					)
			})
		})

		Context("complex type structure from arc/types", func() {
			It("Should handle the complete Type structure with FunctionProperties", func() {
				source := `
					@cpp output "arc/cpp/types"

					Param struct {
						name string
						kind string
					}

					Params Param[]

					FunctionProperties struct {
						inputs Params?
						outputs Params?
						config Params?
					}

					Type struct extends FunctionProperties {
						kind string
						name string
						elem Type??
						constraint Type??
					}
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)
				Expect(resp.Files).To(HaveLen(1))

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// FunctionProperties fields use wrapper type (soft optional uses bare type)
						`parser.field<Params>("inputs")`,
						`parser.field<Params>("outputs")`,
						`parser.field<Params>("config")`,
						// Type fields should use indirect<T> with implicit nullptr default
						`parser.field<x::mem::indirect<Type>>("elem")`,
						`parser.field<x::mem::indirect<Type>>("constraint")`,
					)
			})
		})

		Context("primitive array handling", func() {
			It("Should generate simple assignment for primitive arrays", func() {
				source := `
					@cpp output "client/cpp/types"

					Data struct {
						values int32[]
						names string[]
					}
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<std::vector<std::int32_t>>("values")`,
						`parser.field<std::vector<std::string>>("names")`,
						// Primitive arrays should use direct assignment in to_json
						`j["values"] = this->values`,
						`j["names"] = this->names`,
					)
			})
		})

		Context("struct arrays without alias", func() {
			It("Should use to_array helper for struct array elements", func() {
				source := `
					@cpp output "client/cpp/types"

					Item struct {
						name string
						value int32
					}

					Container struct {
						items Item[]
					}
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<std::vector<Item>>("items")`,
						`j["items"] = x::json::to_array(this->items)`,
					)
			})
		})

		Context("array wrapper distinct types", func() {
			It("Should generate parse/to_json for array wrappers with primitive elements", func() {
				source := `
					@cpp output "arc/cpp/types"

					Channels uint32[]
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// parse method uses parser.field<std::vector<T>>()
						"inline Channels Channels::parse(x::json::Parser parser) {",
						"for (auto& item : parser.field<std::vector<std::uint32_t>>())",
						// to_json method
						"inline x::json::json Channels::to_json() const {",
						"x::json::json j = x::json::json::array()",
						"for (const auto& item : *this)",
					)
			})

			It("Should generate parse/to_json for array wrappers with struct elements", func() {
				source := `
					@cpp output "arc/cpp/types"

					Param struct {
						name string
					}

					Params Param[]
				`
				resp := testutil.MustGenerate(ctx, source, "types", loader, jsonPlugin)

				testutil.ExpectContent(resp, "json.gen.h").
					ToContain(
						// parse method uses parser.field<std::vector<StructType>>()
						"inline Params Params::parse(x::json::Parser parser) {",
						"for (auto& item : parser.field<std::vector<Param>>())",
						// to_json method should call item.to_json() for each element
						"inline x::json::json Params::to_json() const {",
						"j.push_back(item.to_json())",
					)
			})
		})
	})
})
