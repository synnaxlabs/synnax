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
						// Should parse as vector of Param
						`parser.field<std::vector<Param>>("inputs")`,
						`parser.field<std::vector<Param>>("outputs")`,
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
						// Should iterate over array and call to_json() on each struct element
						"for (const auto& item : this->inputs) arr.push_back(item.to_json())",
						"for (const auto& item : this->outputs) arr.push_back(item.to_json())",
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
						`parser.field<std::vector<Param>>("inputs")`,
						`parser.field<std::vector<Param>>("outputs")`,
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
						// Self-referential fields should check parser.has() first to avoid infinite recursion
						`parser.has("left") ? x::mem::indirect<Node>(parser.field<Node>("left")) : nullptr`,
						`parser.has("right") ? x::mem::indirect<Node>(parser.field<Node>("right")) : nullptr`,
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
						`parser.has("elem") ? x::mem::indirect<Type>(parser.field<Type>("elem")) : nullptr`,
						`parser.has("constraint") ? x::mem::indirect<Type>(parser.field<Type>("constraint")) : nullptr`,
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
						`parser.field<Unit>("unit")`,
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
						// FunctionProperties fields should use vector parsing
						`parser.field<std::vector<Param>>("inputs")`,
						`parser.field<std::vector<Param>>("outputs")`,
						`parser.field<std::vector<Param>>("config")`,
						// Type fields should use indirect with has() guard
						`parser.has("elem") ? x::mem::indirect<Type>(parser.field<Type>("elem")) : nullptr`,
						`parser.has("constraint") ? x::mem::indirect<Type>(parser.field<Type>("constraint")) : nullptr`,
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
			It("Should iterate and call to_json for struct array elements", func() {
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
						"for (const auto& item : this->items) arr.push_back(item.to_json())",
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
						// parse method
						"inline Channels Channels::parse(x::json::Parser parser) {",
						"for (auto& elem : parser.array())",
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
						// parse method should call Param::parse for each element
						"inline Params Params::parse(x::json::Parser parser) {",
						"result.push_back(Param::parse(elem))",
						// to_json method should call item.to_json() for each element
						"inline x::json::json Params::to_json() const {",
						"j.push_back(item.to_json())",
					)
			})
		})
	})
})
