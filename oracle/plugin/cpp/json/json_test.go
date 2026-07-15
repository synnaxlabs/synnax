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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/cpp/json"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("C++ JSON Plugin", func() {
	var (
		loader     *MockFileLoader
		jsonPlugin *json.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		jsonPlugin = json.New(json.Options{
			FileNamePattern: "json.gen.h",
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
			It("Should generate correct parsing for array alias fields", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// Should parse using the wrapper type which has its own parse method
						`parser.field<Params>("inputs")`,
						`parser.field<Params>("outputs")`,
					)
			})

			It("Should generate correct to_json for array alias fields", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// Wrapper types have their own to_json() method
						`j["inputs"] = this->inputs.to_json()`,
						`j["outputs"] = this->outputs.to_json()`,
					)
			})

			It("Should handle optional array alias fields", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// Optional (?) wrapper types still use field<> with wrapper type
						`parser.field<Params>("inputs")`,
						`parser.field<Params>("outputs")`,
					)
			})
		})

		Context("self-referential types with indirect<T>", func() {
			It("Should generate correct parsing for self-referential fields", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					Node struct {
						name string
						left Node?
						right Node?
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// Self-referential fields use x::mem::indirect<T> with implicit nullptr default
						`parser.field<x::mem::indirect<Node>>("left")`,
						`parser.field<x::mem::indirect<Node>>("right")`,
					)
			})

			It("Should generate correct to_json for self-referential fields", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					Node struct {
						name string
						left Node?
						right Node?
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// Should use has_value() and -> operator for indirect<T>
						`if (this->left.has_value()) j["left"] = this->left->to_json()`,
						`if (this->right.has_value()) j["right"] = this->right->to_json()`,
					)
			})

			It("Should handle mutually recursive types", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					A struct {
						b B?
					}
					B struct {
						a A?
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<x::mem::indirect<B>>("b")`,
						`parser.field<x::mem::indirect<A>>("a")`,
						`if (this->b.has_value()) j["b"] = this->b->to_json()`,
						`if (this->a.has_value()) j["a"] = this->a->to_json()`,
					)
			})

			It("Should handle cycles through a distinct struct wrapper", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<x::mem::indirect<BWrap>>("b")`,
						`parser.field<x::mem::indirect<A>>("a")`,
						`if (this->b.has_value()) j["b"] = this->b->to_json()`,
						`if (this->a.has_value()) j["a"] = this->a->to_json()`,
					)
			})

			It("Should handle nested self-references via type arguments", func(ctx SpecContext) {
				source := `
					@cpp output "arc/cpp/types"

					Type struct {
						kind string
						name string
						elem Type?
						constraint Type?
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<x::mem::indirect<Type>>("elem")`,
						`parser.field<x::mem::indirect<Type>>("constraint")`,
					)
			})
		})

		Context("optional struct fields (non-self-referential)", func() {
			It("Should generate correct to_json for optional struct fields", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// Should use has_value() and -> operator for optional<T>
						`if (this->unit.has_value()) j["unit"] = this->unit->to_json()`,
					)
			})

			It("Should generate correct parsing for optional struct fields", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// Optional struct fields use std::optional<T> with implicit nullopt default
						`parser.field<std::optional<Unit>>("unit")`,
					)
			})
		})

		Context("map type handling", func() {
			It("Should use std::unordered_map for Map types in json serialization", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/task"

					Task struct {
						key uint64
						metadata map<string, string>
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<std::unordered_map<std::string, std::string>>("metadata")`,
					)
			})
		})

		Context("complex type structure from arc/types", func() {
			It("Should handle the complete Type structure with FunctionProperties", func(ctx SpecContext) {
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
						elem Type?
						constraint Type?
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// FunctionProperties fields use wrapper type (optional uses bare type)
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
			It("Should generate simple assignment for primitive arrays", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					Data struct {
						values int32[]
						names string[]
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
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
			It("Should use to_array helper for struct array elements", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<std::vector<Item>>("items")`,
						`j["items"] = x::json::to_array(this->items)`,
					)
			})
		})

		Context("array wrapper distinct types", func() {
			It("Should generate parse/to_json for array wrappers with primitive elements", func(ctx SpecContext) {
				source := `
					@cpp output "arc/cpp/types"

					Channels uint32[]
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
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

			It("Should generate parse/to_json for array wrappers with struct elements", func(ctx SpecContext) {
				source := `
					@cpp output "arc/cpp/types"

					Param struct {
						name string
					}

					Params Param[]
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
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

		Context("generic struct field type args", func() {
			It("Should propagate type args to fields referencing generic structs", func(ctx SpecContext) {
				source := `
					@cpp output "x/cpp/control"

					State struct<R> {
						resource R
					}

					Transfer struct<R> {
						from State<R>?
						to   State<R>?
					}

					Update struct<R> {
						transfers Transfer<R>[]
					}
				`
				resp := MustGenerate(ctx, source, "control", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<std::optional<State<R>>>("from")`,
						`parser.field<std::optional<State<R>>>("to")`,
						`parser.field<std::vector<Transfer<R>>>("transfers")`,
					)
			})

			It("Should skip defaulted type args for non-generic structs", func(ctx SpecContext) {
				source := `
					@cpp output "x/cpp/task"

					Details struct<D? = record> {
						data D
					}

					Task struct {
						details Details
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(`parser.field<Details>("details")`).
					ToNotContain(`Details<`)
			})
		})

		Context("optional primitive defaults", func() {
			It("Should parse optional primitives as std::optional", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					Settings struct {
						count    uint32?
						ratio    float64?
						enabled  bool?
						label    string?
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<std::optional<std::uint32_t>>("count")`,
						`parser.field<std::optional<double>>("ratio")`,
						`parser.field<std::optional<bool>>("enabled")`,
						`parser.field<std::optional<std::string>>("label")`,
					)
			})

			It("Should wrap telem-typed numeric defaults in their constructors", func(ctx SpecContext) {
				loader.Add("schemas/telem", `
					@cpp output "x/cpp/telem"

					TimeSpan int64 {
						@cpp omit
					}

					Rate float64 {
						@cpp omit
					}
				`)
				source := `
					import "schemas/telem"

					@cpp output "out"

					Config struct {
						stream_rate telem.Rate = 5
						duration    telem.TimeSpan = 0
					}
				`
				resp := MustGenerate(ctx, source, "config", loader, jsonPlugin)
				ExpectContent(resp, "out/json.gen.h").
					ToContain(
						`parser.field<::x::telem::Rate>("stream_rate", x::telem::Rate(5))`,
						`parser.field<::x::telem::TimeSpan>("duration", x::telem::TimeSpan(0))`,
					)
			})

			It("Should parse optional uuid fields as std::optional", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					Record struct {
						owner uuid?
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(`parser.field<std::optional<x::uuid::UUID>>("owner")`)
			})

			It("Should parse optional signed integer fields as std::optional", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					Reading struct {
						delta_small int8?
						delta_med   int16?
						delta_big   int64?
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						`parser.field<std::optional<std::int8_t>>("delta_small")`,
						`parser.field<std::optional<std::int16_t>>("delta_med")`,
						`parser.field<std::optional<std::int64_t>>("delta_big")`,
					)
			})
		})

		Context("uuid detection through distinct and alias chains", func() {
			It("Should detect a distinct type wrapping uuid and call to_json()", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					Key uuid

					Entity struct {
						id Key
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(`j["id"] = this->id.to_json();`)
			})

			It("Should detect a uuid via an alias chain and call to_json()", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/types"

					Primary = uuid
					KeyRef  = Primary

					Entity struct {
						id KeyRef
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(`j["id"] = this->id.to_json();`)
			})
		})

		Context("fixed-size uint8 array from another namespace", func() {
			It("Should include the distinct-type-specific header, not json.gen.h, for a cross-namespace reference", func(ctx SpecContext) {
				loader.Add("schemas/crypto", `
					@cpp output "x/cpp/crypto"

					Hash uint8[32]
				`)

				source := `
					import "schemas/crypto"

					@cpp output "client/cpp/types"

					Digest struct {
						hash crypto.Hash
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, jsonPlugin)

				ExpectContent(resp, "json.gen.h").
					ToContain(
						// isFixedSizeUint8ArrayType drives typeRefToCpp to include
						// <output>/<snake_case name>.h rather than json.gen.h.
						`#include "x/cpp/crypto/hash.h"`,
						// Qualified type reference uses the derived namespace.
						`::x::crypto::Hash`,
					).
					ToNotContain(
						`#include "x/cpp/crypto/json.gen.h"`,
					)
			})
		})

		Context("cross-namespace struct extension", func() {
			It("Should qualify the base type by namespace and include the base's json header", func(ctx SpecContext) {
				loader.Add("schemas/base", `
					@cpp output "x/cpp/base"

					BaseEntity struct {
						key string
					}
				`)

				source := `
					import "schemas/base"

					@cpp output "client/cpp/derived"

					Derived struct extends base.BaseEntity {
						name string
					}
				`
				resp := MustGenerate(ctx, source, "derived", loader, jsonPlugin)

				ExpectContent(resp, "derived/json.gen.h").
					ToContain(
						// resolveExtendsType emits the cross-namespace qualified
						// name and registers the base's json.gen.h include.
						`#include "x/cpp/base/json.gen.h"`,
						`::x::base::BaseEntity`,
					)
			})
		})

		Context("package with nothing to serialize", func() {
			It("Should request deletion of json.gen.h instead of emitting a file", func(ctx SpecContext) {
				source := `
					@cpp output "client/cpp/node"

					Key uint12 {
						@doc value "is a 12-bit unsigned integer identifying a node."
					}
				`
				resp := MustGenerate(ctx, source, "node", loader, jsonPlugin)

				Expect(resp.Files).To(BeEmpty())
				Expect(resp.Deletions).To(ConsistOf("client/cpp/node/json.gen.h"))
			})
		})

		Context("plugin interface", func() {
			It("Should return default options with json.gen.h filename", func() {
				opts := json.DefaultOptions()
				Expect(opts.FileNamePattern).To(Equal("json.gen.h"))
			})

			It("Should report nil for Check", func() {
				Expect(jsonPlugin.Check(nil)).To(Succeed())
			})
		})
	})
})

var _ = Describe("C++ JSON Union Generation", func() {
	var (
		loader     *MockFileLoader
		jsonPlugin *json.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		jsonPlugin = json.New(json.Options{FileNamePattern: "json.gen.h"})
	})

	It("Should generate a parse dispatch and to_json overload for the union", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, jsonPlugin)
		ExpectContent(resp, "json.gen.h").
			ToContain(
				`inline Scale parse_scale(x::json::Parser parser) {`,
				`const auto discriminator = parser.field<std::string>("type");`,
				`if (discriminator == "linear") return ScaleLinear::parse(parser);`,
				`if (discriminator == "none") return ScaleNone::parse(parser);`,
				`parser.field_err("type", "unknown Scale type: " + discriminator);`,
				`return {};`,
				`inline x::json::json to_json(const Scale& value) {`,
				`return std::visit([](const auto& v) { return v.to_json(); }, value);`,
			)
	})

	It("Should parse inline variant fields directly on the variant struct", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			TabBase struct { key string }

			Tab union on variant extends TabBase {
				view {
					type string
				}
				empty {}
			}
		`
		resp := MustGenerate(ctx, source, "panel", loader, jsonPlugin)
		content := ExpectContent(resp, "json.gen.h")
		content.ToContain(
			`if (discriminator == "view") return TabView::parse(parser);`,
			`if (discriminator == "empty") return TabEmpty::parse(parser);`,
		)
		content.ToNotContain("TabViewPayload")
	})

	It("Should generate per-variant parse/to_json bodies", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, jsonPlugin)
		ExpectContent(resp, "json.gen.h").
			ToContain(
				`inline ScaleLinear ScaleLinear::parse(x::json::Parser parser) {`,
				`static_cast<LinearScale&>(result) = LinearScale::parse(parser);`,
				`result.type = parser.field<std::string>("type");`,
				// A field-less struct never reads the parser, so the parameter is
				// left unnamed to avoid an unused-parameter warning.
				`inline NoneScale NoneScale::parse(x::json::Parser) {`,
			)
	})

	It("Should dispatch a union-typed field through the free functions", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}

			Channel struct {
				customScale Scale
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, jsonPlugin)
		ExpectContent(resp, "json.gen.h").
			ToContain(
				`.custom_scale = parse_scale(parser.child("custom_scale")),`,
				`j["custom_scale"] = ::synnax::out::to_json(this->custom_scale);`,
			)
	})

	It("Should parse defaulted fields with their schema defaults", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			Units enum {
				volts = "Volts"
				amps  = "Amps"
			}

			Config struct {
				enabled bool = false
				units   Units = volts
				label   string = ""
			}
		`
		resp := MustGenerate(ctx, source, "config", loader, jsonPlugin)
		ExpectContent(resp, "json.gen.h").
			ToContain(
				`.enabled = parser.field<bool>("enabled", false),`,
				`.units = parser.field<std::string>("units", "Volts"),`,
				`.label = parser.field<std::string>("label", ""),`,
			)
	})

	It("Should parse record fields with an empty default as an empty object", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			Command struct {
				type string
				args record = {}
			}
		`
		resp := MustGenerate(ctx, source, "config", loader, jsonPlugin)
		ExpectContent(resp, "json.gen.h").
			ToContain(
				`.args = parser.field<x::json::json::object_t>("args", x::json::json::object_t{}),`,
			)
	})

	It("Should parse defaulted distinct-typed fields with their schema defaults", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			Key string {}

			Config struct {
				device Key = ""
			}
		`
		resp := MustGenerate(ctx, source, "config", loader, jsonPlugin)
		ExpectContent(resp, "json.gen.h").
			ToContain(`.device = parser.field<Key>("device", ""),`)
	})

	It("Should wrap telem-typed numeric defaults in their constructors", func(ctx SpecContext) {
		loader.Add("schemas/telem", `
			@cpp output "x/cpp/telem"

			TimeSpan int64 {
				@cpp omit
			}
		`)
		source := `
			import "schemas/telem"

			@cpp output "out"

			Config struct {
				duration telem.TimeSpan = 0
			}
		`
		resp := MustGenerate(ctx, source, "config", loader, jsonPlugin)
		ExpectContent(resp, "out/json.gen.h").
			ToContain(`.duration = parser.field<::x::telem::TimeSpan>("duration", x::telem::TimeSpan(0)),`)
	})

	It("Should keep fields with sentinel defaults required", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			Status struct {
				key  string = create
				name string = ""
			}
		`
		resp := MustGenerate(ctx, source, "config", loader, jsonPlugin)
		ExpectContent(resp, "json.gen.h").
			ToContain(
				`.key = parser.field<std::string>("key"),`,
				`.name = parser.field<std::string>("name", ""),`,
			)
	})
})

var _ = Describe("C++ JSON Union Array Fields", func() {
	var (
		loader     *MockFileLoader
		jsonPlugin *json.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		jsonPlugin = json.New(json.Options{FileNamePattern: "json.gen.h"})
	})

	It("Should dispatch an array of unions element-by-element", func(ctx SpecContext) {
		source := `
			@cpp output "out"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}

			Channel struct {
				scales Scale[]
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, jsonPlugin)
		ExpectContent(resp, "json.gen.h").
			ToContain(
				`parser.iter("scales", [&result](x::json::Parser& p) { result.push_back(parse_scale(p)); });`,
				`for (const auto& item : this->scales) arr.push_back(::synnax::out::to_json(item));`,
			)
	})
})
