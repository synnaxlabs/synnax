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
	"github.com/synnaxlabs/oracle/testutil"
)

func TestCppPB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "C++ PB Plugin Suite")
}

var _ = Describe("C++ PB Plugin", func() {
	var (
		ctx      context.Context
		loader   *testutil.MockFileLoader
		pbPlugin *pb.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)
				Expect(resp.Files).To(HaveLen(1))

				testutil.ExpectContent(resp, "proto.gen.h").
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Should iterate over pb array and push_back to cpp vector
						"for (const auto& item : pb.inputs())",
						"for (const auto& item : pb.outputs())",
						"cpp.inputs.push_back",
						"cpp.outputs.push_back",
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: call to_proto() on each struct element
						"item.to_proto()",
						// Backward: call from_proto() on each pb message
						"::from_proto(item)",
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: check has_value() and use mutable_*
						"if (this->unit.has_value())",
						"*pb.mutable_unit()",
						"this->unit->to_proto()",
						// Backward: check has_* and call from_proto
						"if (pb.has_unit())",
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)
				Expect(resp.Files).To(HaveLen(1))

				testutil.ExpectContent(resp, "proto.gen.h").
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
				resp := testutil.MustGenerate(ctx, source, "status", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
					ToContain(
						"VariantToPB",
						"VariantFromPB",
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
				resp := testutil.MustGenerate(ctx, source, "status", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: use mutable_* and to_value
						"*pb.mutable_value() = x::json::to_value(this->value).first",
						// Backward: use from_value
						"x::json::from_value(pb.value())",
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: check has_value() for hard optional
						"if (this->value.has_value())",
						"*pb.mutable_value() = x::json::to_value(*this->value).first",
						// Backward: check has_* for hard optional
						"if (pb.has_value())",
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
					ToContain(
						// Forward: use mutable_* and to_struct
						"*pb.mutable_metadata() = x::json::to_struct(this->metadata).first",
						// Backward: use from_struct
						"x::json::from_struct(pb.metadata())",
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
				resp := testutil.MustGenerate(ctx, source, "types", loader, pbPlugin)

				testutil.ExpectContent(resp, "proto.gen.h").
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
	})
})
