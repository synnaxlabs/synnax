// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/cpp/pb"
	. "github.com/synnaxlabs/oracle/testutil"
)

func TestCppPB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "C++ PB Plugin Suite")
}

var _ = Describe("C++ PB Plugin", func() {
	var (
		ctx      context.Context
		loader   *MockFileLoader
		pbPlugin *pb.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = NewMockFileLoader()
		pbPlugin = pb.New(pb.Options{
			FileNamePattern:  "proto.gen.h",
			DisableFormatter: true,
		})
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(pbPlugin.Name()).To(Equal("cpp/pb"))
		})

		It("Should filter on cpp and pb domains", func() {
			Expect(pbPlugin.Domains()).To(Equal([]string{"cpp", "pb"}))
		})

		It("Should require cpp/types and pb/types", func() {
			Expect(pbPlugin.Requires()).To(Equal([]string{"cpp/types", "pb/types"}))
		})
	})

	Describe("DefaultOptions", func() {
		It("Should return options with default file pattern", func() {
			opts := pb.DefaultOptions()
			Expect(opts.FileNamePattern).To(Equal("proto.gen.h"))
		})
	})

	Describe("Check", func() {
		It("Should return nil", func() {
			Expect(pbPlugin.Check(nil)).To(Succeed())
		})
	})

	Describe("PostWrite", func() {
		It("Should return nil with formatter disabled", func() {
			Expect(pbPlugin.PostWrite(nil)).To(Succeed())
			Expect(pbPlugin.PostWrite([]string{})).To(Succeed())
		})
	})

	Describe("Generate", func() {
		Context("array alias fields (e.g., Params -> Param[])", func() {
			It("Should use add_* for repeated fields in forward conversion", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

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
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Should use add_* for repeated fields, not set_*
						"pb.add_inputs()",
						"pb.add_outputs()",
						// Should iterate over the array
						"for (const auto& item : this->inputs)",
						"for (const auto& item : this->outputs)",
					).
					ToNotContain(
						// Should NOT use set_* for repeated fields
						"pb.set_inputs(",
						"pb.set_outputs(",
					)
			})

			It("Should generate correct backward conversion for array aliases", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

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
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Should use from_proto_repeated helper for struct arrays with explicit element type
						"x::pb::from_proto_repeated<Param>(cpp.inputs, pb.inputs())",
						"x::pb::from_proto_repeated<Param>(cpp.outputs, pb.outputs())",
					)
			})

			It("Should call to_proto/from_proto for struct element types", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Param struct {
						name string
						value int32
					}

					Params Param[]

					FunctionProperties struct {
						inputs Params
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: call to_proto() on each struct element
						"item.to_proto()",
						// Backward: use from_proto_repeated helper
						"from_proto_repeated",
					)
			})
		})

		Context("direct array fields (non-alias)", func() {
			It("Should use add_* for direct array fields with struct elements", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Item struct {
						name string
						value int32
					}

					Container struct {
						items Item[]
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						"for (const auto& item : this->items)",
						"pb.add_items()",
						"item.to_proto()",
					)
			})

			It("Should use add_* for primitive arrays", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Data struct {
						values int32[]
						names string[]
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						"pb.add_values(item)",
						"pb.add_names(item)",
					)
			})
		})

		Context("optional struct fields", func() {
			It("Should use has_value() and mutable_* for optional structs", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Unit struct {
						name string
						scale float64
					}

					Type struct {
						name string
						unit Unit??
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: check has_value() and use mutable_*
						"if (this->unit.has_value())",
						"*pb.mutable_unit()",
						"this->unit->to_proto()",
						// Backward: check has_* and use inline error handling
						"if (pb.has_unit())",
						"Unit::from_proto(pb.unit())",
						"if (err) return {{}, err}",
					)
			})
		})

		Context("self-referential types", func() {
			It("Should handle self-referential optional struct fields", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Node struct {
						name string
						left Node??
						right Node??
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Should use has_value() for indirect<T> fields
						"if (this->left.has_value())",
						"if (this->right.has_value())",
						// Should use -> to access to_proto()
						"this->left->to_proto()",
						"this->right->to_proto()",
					)
			})
		})

		Context("complex type structure from arc/types", func() {
			It("Should handle the complete Type structure with FunctionProperties and self-refs", func() {
				source := `
					@cpp output "arc/cpp/types"
					@pb output "arc/go/types/pb"

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
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// FunctionProperties array alias fields should use add_*
						"pb.add_inputs()",
						"pb.add_outputs()",
						"pb.add_config()",
						// Self-referential fields should use has_value() and ->
						"if (this->elem.has_value())",
						"if (this->constraint.has_value())",
						"this->elem->to_proto()",
						"this->constraint->to_proto()",
					)
			})
		})

		Context("enum handling", func() {
			It("Should generate enum translator functions for string enums", func() {
				source := `
					@cpp output "client/cpp/status"
					@pb output "core/pkg/service/status/pb"

					Variant enum {
						success = "success"
						error = "error"
					}

					Status struct {
						variant Variant
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						"VariantToPB",
						"VariantFromPB",
						// Should use static unordered_map for O(1) lookup
						"static const std::unordered_map",
						"kMap.find(cpp)",
					)
			})

			It("Should use static_cast for int enums", func() {
				source := `
					@cpp output "client/cpp/status"
					@pb output "core/pkg/service/status/pb"

					Kind enum {
						unknown = 0
						known = 1
					}

					Item struct {
						kind Kind
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						"static_cast<",
					)
			})
		})

		Context("any type handling", func() {
			It("Should use x::json::to_value/from_value for any type fields", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Param struct {
						name string
						value any
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: use mutable_* and to_value
						"*pb.mutable_value() = x::json::to_value(this->value).first",
						// Backward: use inline error handling with from_value
						"x::json::from_value(pb.value())",
						"if (err) return {{}, err}",
					).
					ToNotContain(
						// Should NOT use set_value for any type
						"pb.set_value(",
					)
			})

			It("Should handle hard optional any type fields", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Param struct {
						name string
						value any??
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: check has_value() for hard optional
						"if (this->value.has_value())",
						"*pb.mutable_value() = x::json::to_value(*this->value).first",
						// Backward: check has_* and use inline error handling
						"if (pb.has_value())",
						"x::json::from_value(pb.value())",
					)
			})
		})

		Context("json type handling", func() {
			It("Should use x::json::to_struct/from_struct for json type fields", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Config struct {
						name string
						metadata json
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: use mutable_* and to_struct
						"*pb.mutable_metadata() = x::json::to_struct(this->metadata).first",
						// Backward: use inline error handling with from_struct
						"x::json::from_struct(pb.metadata())",
						"if (err) return {{}, err}",
					).
					ToNotContain(
						// Should NOT use set_metadata for json type
						"pb.set_metadata(",
					)
			})
		})

		Context("Map type handling", func() {
			It("Should iterate and insert for map fields", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Channels struct {
						read map<uint32, string>
						write map<uint32, string>
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: iterate over map and insert into mutable_*
						"for (const auto& [k, v] : this->read)",
						"(*pb.mutable_read())[k] = v",
						"for (const auto& [k, v] : this->write)",
						"(*pb.mutable_write())[k] = v",
						// Backward: iterate over pb map and insert into cpp map
						"for (const auto& [k, v] : pb.read())",
						"cpp.read[k] = v",
						"for (const auto& [k, v] : pb.write())",
						"cpp.write[k] = v",
					).
					ToNotContain(
						// Should NOT use set_* for map fields
						"pb.set_read(",
						"pb.set_write(",
					)
			})
		})

		Context("nested array handling (array of arrays)", func() {
			It("Should use wrapper messages for nested arrays via alias", func() {
				// This tests the Strata pattern: Strata = Stratum[] where Stratum = string[]
				source := `
					@cpp output "arc/cpp/ir"
					@pb output "arc/go/ir/pb"

					Stratum = string[]

					Strata Stratum[]

					Stage struct {
						key string
						strata Strata
					}
				`
				resp := MustGenerate(ctx, source, "ir", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: should get a wrapper via add_strata(), then add values
						"for (const auto& item : this->strata)",
						"auto* wrapper = pb.add_strata()",
						"for (const auto& v : item) wrapper->add_values(v)",
						// Backward: should iterate over wrappers and extract values
						"for (const auto& wrapper : pb.strata())",
						"cpp.strata.push_back({wrapper.values().begin(), wrapper.values().end()})",
					).
					ToNotContain(
						// Should NOT directly add items (wrong API for nested arrays)
						"pb.add_strata(item)",
					)
			})

			It("Should handle nested arrays in direct array fields", func() {
				// Oracle doesn't support string[][] directly, so we use an alias
				source := `
					@cpp output "arc/cpp/ir"
					@pb output "arc/go/ir/pb"

					Row = string[]

					Matrix struct {
						rows Row[]
					}
				`
				resp := MustGenerate(ctx, source, "ir", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: should use wrapper pattern
						"auto* wrapper = pb.add_rows()",
						"wrapper->add_values(v)",
						// Backward: should extract from wrapper
						"wrapper.values().begin()",
					)
			})

			It("Should handle nested arrays alongside other fields in a struct", func() {
				// This tests a more complex case similar to IR struct
				source := `
					@cpp output "arc/cpp/ir"
					@pb output "arc/go/ir/pb"

					Stratum = string[]
					Strata Stratum[]

					Node struct {
						key string
					}

					Nodes Node[]

					IR struct {
						nodes Nodes
						strata Strata
					}
				`
				resp := MustGenerate(ctx, source, "ir", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Regular struct array should use normal pattern
						"for (const auto& item : this->nodes) *pb.add_nodes() = item.to_proto()",
						// Nested array should use wrapper pattern
						"for (const auto& item : this->strata)",
						"auto* wrapper = pb.add_strata()",
						"wrapper->add_values(v)",
					)
			})

			It("Should handle nested arrays through distinct type alias", func() {
				source := `
					@cpp output "arc/cpp/ir"
					@pb output "arc/go/ir/pb"

					Row string[]
					Grid Row[]

					Table struct {
						data Grid
					}
				`
				resp := MustGenerate(ctx, source, "ir", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Should detect nested array through alias chain
						"auto* wrapper = pb.add_data()",
						"wrapper->add_values(v)",
					)
			})
		})

		Describe("Array Wrapper Proto Generation", func() {
			// Proto uses repeated fields for arrays, not wrapper messages.
			// So array wrapper distinct types (like Params Param[]) cannot have
			// proto methods - there's no proto message to convert to/from.
			It("Should not generate proto for array wrappers (proto uses repeated fields)", func() {
				source := `
					@cpp output "arc/cpp/types"
					@pb output "x/go/types/pb"

					Channels uint32[]
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				// No proto.gen.h should be generated for array-only schemas
				Expect(len(resp.Files)).To(Equal(0))
			})

			It("Should generate proto for structs but not array wrappers in same schema", func() {
				source := `
					@cpp output "arc/cpp/types"
					@pb output "x/go/types/pb"

					Param struct {
						name string
						dataType string
					}

					Params Param[]
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				// Should generate proto for Param struct only
				ExpectContent(resp, "proto.gen.h").
					ToContain(
						"Param::to_proto() const {",
					).
					ToNotContain(
						// Params wrapper should NOT have proto methods
						"Params::to_proto()",
					)
			})
		})

		Context("includes", func() {
			It("Should include x/cpp/pb/pb.h for helpers", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Item struct {
						name string
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						`#include "x/cpp/pb/pb.h"`,
					)
			})

			It("Should include unordered_map and string for string enums", func() {
				source := `
					@cpp output "client/cpp/status"
					@pb output "core/pkg/service/status/pb"

					Variant enum {
						success = "success"
						error = "error"
					}

					Status struct {
						variant Variant
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain(
						"#include <unordered_map>",
						"#include <string>",
					)
			})

			It("Should only include type_traits for generic types", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Item struct {
						name string
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToNotContain(
						"#include <type_traits>",
					)
			})
		})

		Context("json field conversion", func() {
			It("Should include json struct header for json fields", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Config struct {
						data json
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("x::json")
			})

			It("Should handle optional json fields with has_value check", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Config struct {
						data json??
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("has_value()")
			})
		})

		Context("any field conversion", func() {
			It("Should handle any fields with json value helpers", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Response struct {
						payload any
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("x::json")
			})
		})

		Context("bytes field conversion", func() {
			It("Should handle bytes fields with data/size", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Frame struct {
						payload bytes
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("payload")
			})
		})

		Context("hard optional uuid field", func() {
			It("Should generate has_value check for optional uuid", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Task struct {
						key uuid
						parent uuid??
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("has_value()").
					ToContain("to_string()")
			})
		})

		Context("alias to struct type", func() {
			It("Should generate to_proto/from_proto for alias that targets struct", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Base struct {
						name string
					}

					Custom = Base

					Wrapper struct {
						item Custom
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("to_proto()").
					ToContain("from_proto")
			})
		})

		Context("struct extends with fields", func() {
			It("Should include parent fields in translation", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Base struct {
						key uuid
						name string
					}

					Derived struct extends Base {
						extra int32
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("Base::to_proto()").
					ToContain("Derived::to_proto()").
					ToContain("extra")
			})
		})

		Context("cross-namespace struct reference", func() {
			BeforeEach(func() {
				loader.Add("schemas/common", `
					@cpp output "client/cpp/common"
					@pb output "core/pkg/service/common/pb"

					Info struct {
						name string
						description string
					}
				`)
			})

			It("Should include cross-namespace headers", func() {
				source := `
					import "schemas/common"

					@cpp output "client/cpp/task"
					@pb output "core/pkg/service/task/pb"

					Task struct {
						key uuid
						info common.Info
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("to_proto()").
					ToContain("from_proto")
			})
		})

		Context("map field conversion", func() {
			It("Should handle map fields with mutable accessor", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					Settings struct {
						values map<string, string>
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("mutable_values()")
			})
		})

		Context("distinct type with primitive base", func() {
			It("Should cast through distinct type", func() {
				source := `
					@cpp output "client/cpp/types"
					@pb output "core/pkg/service/types/pb"

					NodeID = uint32

					Node struct {
						id NodeID
					}
				`
				resp := MustGenerate(ctx, source, "types", loader, pbPlugin)

				ExpectContent(resp, "proto.gen.h").
					ToContain("id")
			})
		})
	})
})
