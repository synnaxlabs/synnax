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
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/pb"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestGoPB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Go PB Suite")
}

var _ = Describe("Go PB Plugin", func() {
	var (
		loader   *MockFileLoader
		pbPlugin *pb.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		pbPlugin = pb.New(pb.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(pbPlugin.Name()).To(Equal("go/pb"))
		})

		It("Should filter on pb domain", func() {
			Expect(pbPlugin.Domains()).To(Equal([]string{"pb"}))
		})

		It("Should require go/types and pb/types", func() {
			Expect(pbPlugin.Requires()).To(Equal([]string{"go/types", "pb/types"}))
		})
	})

	Describe("Generate", func() {
		Context("union translation", func() {
			It("Should generate oneof translators for a discriminated union", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Spec struct {
						type string
						props record
					}

					Source union on value_type {
						boolean Spec
						number Spec
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"func SourceToPB(r schematic.Source) (*Source, error)",
						"if r.Variant == nil {",
						"case schematic.SourceBoolean:",
						"inner, err := SpecToPB(v.Spec)",
						"pb.Variant = &Source_Boolean{Boolean: inner}",
						`errors.Newf("Source: unknown variant %T", r.Variant)`,
						"func SourceFromPB(pb *Source) (schematic.Source, error)",
						"case *Source_Number:",
						"r.Variant = schematic.SourceNumber{Spec: inner}",
						"func SourcesToPB(rs []schematic.Source) ([]*Source, error)",
					)
			})

			It("Should route union-typed fields through the union translators", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Spec struct {
						type string
					}

					Source union on value_type {
						boolean Spec
					}

					Config struct {
						source Source
						sources Source[]
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"sourceVal, err := SourceToPB(r.Source)",
						"r.Source, err = SourceFromPB(pb.Source)",
						"SourcesToPB(r.Sources)",
						"SourcesFromPB(pb.Sources)",
					)
			})

			It("Should camelize multi-word variant values in oneof wrapper names", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Body struct {
						width float64
					}

					Shape union on variant {
						isoCap Body
						tJunction Body
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"case schematic.ShapeIsoCap:",
						"pb.Variant = &Shape_IsoCap{IsoCap: inner}",
						"case *Shape_TJunction:",
						"r.Variant = schematic.ShapeTJunction{Body: inner}",
					)
			})

			It("Should suffix oneof wrapper names that collide with protoc methods", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Spec struct {
						type string
					}

					Sink union on value_type {
						string Spec
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"pb.Variant = &Sink_String_{String_: inner}",
						"case *Sink_String_:",
					)
			})

			It("Should convert record array fields through shared helpers", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Config struct {
						overrides record[]
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"overridesVal, err := recordsToPB(r.Overrides)",
						"r.Overrides = recordsFromPB(pb.Overrides)",
						"func recordsToPB(rs []msgpack.EncodedJSON) ([]*structpb.Struct, error)",
						"func recordsFromPB(pbs []*structpb.Struct) []msgpack.EncodedJSON",
					)
			})

			It("Should convert union map values through union translators", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Spec struct {
						type string
					}

					Source union on value_type {
						boolean Spec
					}

					Config struct {
						sources map<string, Source>
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"SourceToPB(v)",
						"SourceFromPB(v)",
					)
			})

			It("Should deref optional typedef fields for conversion", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Key uint32 {
						@doc value "is a channel key."
					}

					Config struct {
						state_channel Key?
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"v := uint32(*r.StateChannel)",
						"pb.StateChannel = &v",
					)
			})

			It("Should translate union extends bases through the bases' own translators", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Base struct {
						key string
					}

					Body struct {
						width float64
					}

					Shape union on variant extends Base {
						square Body
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToBeValidGoSource().
					ToContain(
						"pb.Base, err = BaseToPB(v.Base)",
						"m := schematic.ShapeSquare{Body: inner}",
						"m.Base, err = BaseFromPB(pb.Base)",
						"r.Variant = m",
					)
			})

			It("Should translate inline variants against the variant member", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Base struct {
						key string
					}

					Shape union on variant extends Base {
						square {
							width float64
						}
						empty {}
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToBeValidGoSource().
					ToContain(
						"func ShapeSquareToPB(r schematic.ShapeSquare) (*ShapeSquarePayload, error)",
						"inner, err := ShapeSquareToPB(v)",
						"m := inner",
						"m.Base, err = BaseFromPB(pb.Base)",
					)
			})

			It("Should translate inline variants inherited through union composition", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					NodeConfig union on variant {
						box {
							width float64
						}
					}

					EdgeConfig union on variant {
						pipe {
							length float64
						}
					}

					ElementConfig union on variant extends NodeConfig, EdgeConfig {}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, pbPlugin)
				ExpectContent(resp, "translator.gen.go").
					ToBeValidGoSource().
					ToContain(
						"func NodeConfigBoxToPB(r schematic.NodeConfigBox) (*NodeConfigBoxPayload, error)",
						"func ElementConfigBoxToPB(r schematic.ElementConfigBox) (*NodeConfigBoxPayload, error)",
						"func ElementConfigPipeToPB(r schematic.ElementConfigPipe) (*EdgeConfigPipePayload, error)",
						"pb.Variant = &ElementConfig_Box{Box: inner}",
					).
					ToNotContain("NodeConfigBoxPayloadToPB")
			})
		})

		Context("simple struct translation", func() {
			It("Should generate ToPB and FromPB functions", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/user"
					@pb

					User struct {
						key uuid
						name string
						age int32
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, pbPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"// Code generated by oracle. DO NOT EDIT.",
						"package pb",
						"func UserToPB(r user.User) (*User, error)",
						"func UserFromPB(pb *User) (user.User, error)",
					).
					ToPreserveOrder(
						"func UserToPB",
						"func UserFromPB",
					)
			})

			It("Should generate slice translators", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/user"
					@pb

					User struct {
						key uuid
						name string
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"func UsersToPB(rs []user.User) ([]*User, error)",
						"func UsersFromPB(pbs []*User) ([]user.User, error)",
					)
			})

			It("Should use List suffix for already-plural type names", func(ctx SpecContext) {
				source := `
					@go output "arc/go/ir"
					@pb

					Authorities struct {
						default uint8?
					}
				`
				resp := MustGenerate(ctx, source, "ir", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"func AuthoritiesListToPB(",
						"func AuthoritiesListFromPB(",
					)
			})
		})

		Context("primitive field conversions", func() {
			DescribeTable("should convert primitive types correctly",
				func(ctx SpecContext, fieldDecl, expectedForward, expectedBackward string) {
					source := `
						@go output "core/test"
						@pb

						Test struct {
							` + fieldDecl + `
						}
					`
					resp := MustGenerate(ctx, source, "test", loader, pbPlugin)
					content := MustContentOf(resp, "translator.gen.go")
					Expect(content).To(ContainSubstring(expectedForward))
					Expect(content).To(ContainSubstring(expectedBackward))
				},
				Entry("uuid to string",
					"key uuid",
					"Key: r.Key.String()",
					"r.Key, err = uuid.Parse(pb.Key)",
				),
				Entry("string passthrough",
					"name string",
					"Name: r.Name",
					"r.Name = pb.Name",
				),
				Entry("bool passthrough",
					"active bool",
					"Active: r.Active",
					"r.Active = pb.Active",
				),
				Entry("int32 passthrough",
					"count int32",
					"Count: r.Count",
					"r.Count = pb.Count",
				),
				Entry("int64 passthrough",
					"count int64",
					"Count: r.Count",
					"r.Count = pb.Count",
				),
				Entry("uint32 passthrough",
					"count uint32",
					"Count: r.Count",
					"r.Count = pb.Count",
				),
				Entry("uint64 passthrough",
					"count uint64",
					"Count: r.Count",
					"r.Count = pb.Count",
				),
				Entry("float32 passthrough",
					"value float32",
					"Value: r.Value",
					"r.Value = pb.Value",
				),
				Entry("float64 passthrough",
					"value float64",
					"Value: r.Value",
					"r.Value = pb.Value",
				),
				Entry("bytes passthrough",
					"data bytes",
					"Data: r.Data",
					"r.Data = pb.Data",
				),
			)
		})

		Context("array field conversions", func() {
			It("Should handle uuid array to string array", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						keys uuid[]
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("Keys: lo.Map(r.Keys").
					ToContain("r.Keys, err = func() ([]uuid.UUID, error)")
			})

			It("Should handle string array passthrough", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						tags string[]
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("Tags: r.Tags").
					ToContain("r.Tags = pb.Tags")
			})
		})

		Context("enum translation", func() {
			It("Should generate enum ToPB and FromPB functions that return errors", func(ctx SpecContext) {
				source := `
					@go output "core/status"
					@pb

					Status enum {
						unknown = 0
						pending = 1
						active = 2
					}

					Task struct {
						key uuid
						status Status
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"func StatusToPB(v status.Status) (Status, error)",
						"func StatusFromPB(v Status) (status.Status, error)",
					).
					ToContain(
						"case status.StatusUnknown:",
						"case status.StatusPending:",
						"case status.StatusActive:",
					)
			})

			It("Should return an error for unrecognized enum values", func(ctx SpecContext) {
				source := `
					@go output "core/status"
					@pb

					Status enum {
						unknown = 0
						active = 1
					}

					Task struct {
						key uuid
						status Status
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)
				content := MustContentOf(resp, "translator.gen.go")

				// ToPB default case should return error, not a silent default
				Expect(content).ToNot(ContainSubstring("default:\n\t\treturn Status_STATUS_"))
				Expect(content).To(ContainSubstring("default:"))
				Expect(content).To(ContainSubstring("errors.Newf"))

				// FromPB default case should return error, not a silent default
				Expect(content).ToNot(ContainSubstring("default:\n\t\treturn status.Status"))
			})

			It("Should propagate enum errors in struct ToPB fields", func(ctx SpecContext) {
				source := `
					@go output "core/status"
					@pb

					Status enum {
						active = 0
						inactive = 1
					}

					Task struct {
						key uuid
						status Status
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)
				content := MustContentOf(resp, "translator.gen.go")

				// Enum conversion in struct should handle the error return
				Expect(content).To(ContainSubstring("StatusToPB(r.Status)"))
				Expect(content).To(ContainSubstring("if err != nil"))
			})

			It("Should propagate enum errors in struct FromPB fields", func(ctx SpecContext) {
				source := `
					@go output "core/status"
					@pb

					Status enum {
						active = 0
						inactive = 1
					}

					Task struct {
						key uuid
						status Status
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)
				content := MustContentOf(resp, "translator.gen.go")

				// Enum conversion in struct should handle the error return
				Expect(content).To(ContainSubstring("StatusFromPB(pb.Status)"))
				Expect(content).To(ContainSubstring("if err != nil"))
			})
		})

		It("Should preserve enum declaration order", func(ctx SpecContext) {
			source := `
					@go output "core/status"
					@pb

					Alpha enum {
						unknown = 0
						active = 1
					}

					Beta enum {
						unknown = 0
						active = 1
					}

					Charlie enum {
						unknown = 0
						active = 1
					}

					Task struct {
						key uuid
						alpha Alpha
						beta Beta
						charlie Charlie
					}
				`
			resp := MustGenerate(ctx, source, "status", loader, pbPlugin)

			ExpectContent(resp, "translator.gen.go").
				ToPreserveOrder(
					"func AlphaToPB",
					"func AlphaFromPB",
					"func BetaToPB",
					"func BetaFromPB",
					"func CharlieToPB",
					"func CharlieFromPB",
				)
		})

		Context("optional fields", func() {
			It("Should handle optional primitive with nil check", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						key uuid
						name string?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("if r.Name != nil {").
					ToContain("if pb.Name != nil {")
			})

			It("Should deref the pointer on forward and rebind on backward for optional string enums", func(ctx SpecContext) {
				// Regression: a optional enum field was emitting
				// `pb.Type, err = TickTypeToPB(r.Type)` even though r.Type
				// is *TickType and TickTypeToPB takes a value. Backward
				// emitted `r.Type = TickTypeFromPB(pb.Type)` ignoring both
				// the returned error and the pointer target type. The fix
				// derefs in the forward call and uses the same val/&val
				// dance the optional struct branch already had.
				source := `
					@go output "core/test"
					@pb

					TickType enum {
						linear = "linear"
						time   = "time"
					}

					Axis struct {
						key   string
						type  TickType?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				content := ExpectContent(resp, "translator.gen.go")
				content.ToContain("val, err := TickTypeToPB(*r.Type)")
				content.ToContain("pb.Type = &val")
				content.ToContain("val, err := TickTypeFromPB(*pb.Type)")
				content.ToContain("r.Type = &val")
				content.ToNotContain("TickTypeToPB(r.Type)")
				content.ToNotContain("pb.Type, err = TickTypeToPB")
				content.ToNotContain("TickTypeFromPB(pb.Type)")
			})

			It("Should deref the pointer on forward and rebind on backward for optional integer enums", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Level enum {
						low    = 0
						medium = 1
						high   = 2
					}

					Item struct {
						key   string
						level Level?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				content := ExpectContent(resp, "translator.gen.go")
				content.ToContain("val, err := LevelToPB(*r.Level)")
				content.ToContain("pb.Level = &val")
				content.ToContain("val, err := LevelFromPB(*pb.Level)")
				content.ToContain("r.Level = &val")
				content.ToNotContain("LevelToPB(r.Level)")
				content.ToNotContain("pb.Level, err = LevelToPB")
				content.ToNotContain("LevelFromPB(pb.Level)")
			})
		})

		Context("generic struct with all-defaulted type params", func() {
			It("Should emit a non-generic translator call when every type param is defaulted and no args are provided", func(ctx SpecContext) {
				// Regression: a field whose type is a generic struct with a
				// fully-defaulted parameter list (e.g. Bounds<T extends
				// numeric = float64>) used to short-circuit to a raw
				// `pb.Bounds = r.Bounds` assignment because the generic
				// branch required at least one explicit type arg, and the
				// non-generic branch was guarded behind an early return.
				// The fix falls through to the regular non-generic
				// translator call, which matches the Go pb shape that
				// itself emits a concrete BoundsToPB signature for the
				// defaulted instantiation.
				source := `
					@go output "core/test"
					@pb

					Bounds struct<T extends numeric = float64> {
						lower T
						upper T
					}

					Axis struct {
						key    string
						bounds Bounds
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				content := ExpectContent(resp, "translator.gen.go")
				content.ToContain("BoundsToPB(r.Bounds)")
				content.ToContain("BoundsFromPB(pb.Bounds)")
				content.ToNotContain("pb.Bounds = r.Bounds")
				content.ToNotContain("r.Bounds = pb.Bounds")
			})
		})

		Context("struct reference fields", func() {
			It("Should call nested struct translators", func(ctx SpecContext) {
				source := `
					@go output "core/task"
					@pb

					Status struct {
						code int32
						message string
					}

					Task struct {
						key uuid
						status Status
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("StatusToPB(r.Status)").
					ToContain("StatusFromPB(pb.Status)")
			})

			It("Should handle array of struct references", func(ctx SpecContext) {
				source := `
					@go output "core/task"
					@pb

					Item struct {
						key uuid
						name string
					}

					Container struct {
						key uuid
						items Item[]
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("ItemsToPB(r.Items)").
					ToContain("ItemsFromPB(pb.Items)")
			})

			It("Should use List suffix for array of already-plural struct references", func(ctx SpecContext) {
				source := `
					@go output "arc/go/ir"
					@pb

					Authorities struct {
						default uint8?
					}

					Function struct {
						key uuid
						authorities Authorities[]
					}
				`
				resp := MustGenerate(ctx, source, "ir", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("AuthoritiesListToPB(r.Authorities)").
					ToContain("AuthoritiesListFromPB(pb.Authorities)")
			})
		})

		Context("nested array of struct", func() {
			It("Should delegate to generated helpers via IIFE when inner element is a struct", func(ctx SpecContext) {
				source := `
					@go output "arc/go/ir"
					@pb

					Member struct {
						key string
						value int32
					}

					Members = Member[]

					Strata Members[]

					Stage struct {
						key string
						strata Strata
					}
				`
				resp := MustGenerate(ctx, source, "ir", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						// Outer type matches the distinct named wrapper
						"func() (ir.Strata, error) {",
						"result := make(ir.Strata, len(pb.Strata))",
						// Forward IIFE delegates to MembersToPB per inner slice
						"func() ([]*MembersWrapper, error) {",
						"result := make([]*MembersWrapper, len(r.Strata))",
						"for i, inner := range r.Strata {",
						"vals, err := MembersToPB(inner)",
						"result[i] = &MembersWrapper{Values: vals}",
						// Backward IIFE delegates to MembersFromPB
						"for i, w := range pb.Strata {",
						"vals, err := MembersFromPB(w.Values)",
					).
					ToNotContain(
						// Must not fall back to the broken primitive lo.Map form
						"lo.Map(r.Strata, func(inner []string",
						"lo.Map(pb.Strata, func(w *MembersWrapper",
					)
			})
		})

		Context("generic struct translation", func() {
			It("Should generate generic translator functions", func(ctx SpecContext) {
				source := `
					@go output "core/container"
					@pb

					Container struct<T> {
						key uuid
						value T
					}
				`
				resp := MustGenerate(ctx, source, "container", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"func ContainerToPB[T any](",
						"translateT func(T) (*anypb.Any, error)",
					)
			})

			It("Should import anypb for generic types", func(ctx SpecContext) {
				source := `
					@go output "core/container"
					@pb

					Container struct<T> {
						value T
					}
				`
				resp := MustGenerate(ctx, source, "container", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("google.golang.org/protobuf/types/known/anypb")
			})

			It("Should propagate comparable constraint to translator functions", func(ctx SpecContext) {
				source := `
					@go output "core/control"
					@pb

					State struct<R> {
						resource R
					}

					Transfer struct<R extends comparable> {
						from State<R>?
						to   State<R>?
					}
				`
				resp := MustGenerate(ctx, source, "control", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func TransferToPB[R comparable](").
					ToContain("func TransferFromPB[R comparable](").
					ToContain("func TransfersToPB[R comparable](").
					ToContain("func TransfersFromPB[R comparable](").
					ToContain("StateToPB(*r.From, translateR)").
					ToContain("StateFromPB(pb.From, translateR)").
					ToContain("translateR")
			})

			It("Should forward type param args for array fields of generic structs", func(ctx SpecContext) {
				source := `
					@go output "core/control"
					@pb

					Transfer struct<R extends comparable> {
						key uuid
					}

					Update struct<R extends comparable> {
						transfers Transfer<R>[]
					}
				`
				resp := MustGenerate(ctx, source, "control", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("TransfersToPB[R](r.Transfers, translateR)").
					ToContain("TransfersFromPB[R](pb.Transfers, translateR)")
			})
		})

		Context("naming conventions", func() {
			It("Should convert snake_case to PascalCase", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						created_at timestamp
						time_range time_range
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("CreatedAt:").
					ToContain("r.CreatedAt").
					ToContain("TimeRange:")
			})
		})

		Context("import management", func() {
			It("Should not import context package", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						key uuid
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToNotContain(`"context"`)
			})

			It("Should import uuid package when uuid fields present", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						key uuid
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(`"github.com/google/uuid"`)
			})

			It("Should import lo package when array conversions needed", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						keys uuid[]
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(`"github.com/samber/lo"`)
			})
		})

		Context("omit directive", func() {
			It("Should skip structs with @pb omit", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Internal struct {
						secret string

						@pb omit
					}

					Public struct {
						key uuid
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func PublicToPB").
					ToNotContain("InternalToPB")
			})
		})

		Context("multiple structs", func() {
			It("Should generate translators for all structs", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					User struct {
						key uuid
						name string
					}

					Group struct {
						key uuid
						name string
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func UserToPB").
					ToContain("func UserFromPB").
					ToContain("func GroupToPB").
					ToContain("func GroupFromPB")
			})

			It("Should preserve declaration order", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					First struct { key uuid }
					Second struct { key uuid }
					Third struct { key uuid }
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToPreserveOrder(
						"func FirstToPB",
						"func SecondToPB",
						"func ThirdToPB",
					)
			})
		})

		Context("no pb directive", func() {
			It("Should not generate file when @pb is absent", func(ctx SpecContext) {
				source := `
					@go output "core/test"

					User struct {
						key uuid
						name string
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)
				Expect(resp.Files).To(BeEmpty())
			})
		})

		Context("output path", func() {
			It("Should generate file in pb subdirectory of go output", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/user"
					@pb

					User struct {
						key uuid
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, pbPlugin)
				Expect(resp.Files).To(HaveLen(1))
				Expect(resp.Files[0].Path).To(Equal("core/pkg/service/user/pb/translator.gen.go"))
			})
		})

		Context("timestamp and timespan conversions with telem import", func() {
			BeforeEach(func() {
				loader.Add("schemas/telem", `
					@go output "x/go/telem"
					@pb

					TimeStamp = uint64
					TimeSpan = int64
				`)
			})

			It("Should convert timestamp typedef via uint64", func(ctx SpecContext) {
				source := `
					import "schemas/telem"

					@go output "core/test"
					@pb

					Test struct {
						created_at telem.TimeStamp
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("uint64(r.CreatedAt)").
					ToContain("telem.TimeStamp(pb.CreatedAt)")
			})

			It("Should convert timespan typedef via int64", func(ctx SpecContext) {
				source := `
					import "schemas/telem"

					@go output "core/test"
					@pb

					Test struct {
						duration telem.TimeSpan
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("int64(r.Duration)").
					ToContain("telem.TimeSpan(pb.Duration)")
			})
		})

		Context("typedef (distinct type) conversions", func() {
			It("Should convert typedef with numeric base", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Key = uint32

					Test struct {
						rack Key
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("uint32(r.Rack)").
					ToContain("test.Key(pb.Rack)")
			})

			It("Should convert typedef with uuid base", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					ResourceKey = uuid

					Test struct {
						resource_key ResourceKey
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("r.ResourceKey.String()").
					ToContain("uuid.Parse(pb.ResourceKey)").
					ToContain("r.ResourceKey = test.ResourceKey(parsedResourceKey)")
			})
		})

		Context("key domain with numeric types", func() {
			It("Should convert key field with numeric key domain", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Key = uint32

					Test struct {
						@key
						rack Key
						name string
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("uint32(r.Rack)").
					ToContain("test.Key(pb.Rack)")
			})
		})

		Context("int8 conversion", func() {
			It("Should widen int8 to int32", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						priority int8
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("int32(r.Priority)").
					ToContain("int8(pb.Priority)")
			})
		})

		Context("@go name and @pb name annotations", func() {
			It("Should use custom Go name in translator", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@go name "CustomName"
					@pb

					Test struct {
						key uuid
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("test.CustomName")
			})

			It("Should use custom PB name in translator function", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb name "ProtoTest"

					Test struct {
						key uuid
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func ProtoTestToPB").
					ToContain("func ProtoTestFromPB")
			})
		})

		Context("@go hand enum handling", func() {
			It("Should generate against hand-written enum values when @go hand", func(ctx SpecContext) {
				source := `
					@go output "core/status"
					@go hand
					@pb

					Status enum {
						active = 0
						inactive = 1
					}

					Task struct {
						key uuid
						status Status
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("status.StatusActive").
					ToContain("status.StatusInactive")
			})
		})

		Context("@go omit handling", func() {
			It("Should generate no translator for a type omitted in Go", func(ctx SpecContext) {
				source := `
					@go output "core/status"
					@pb

					Status enum {
						@go omit
						active = 0
						inactive = 1
					}

					Task struct {
						key uuid
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func TaskToPB").
					ToNotContain("Status")
			})
		})

		Context("uint8 primitive conversion", func() {
			It("Should dereference optional uint8 pointer for conversion", func(ctx SpecContext) {
				source := `
					@go output "arc/go/ir"
					@pb

					Authorities struct {
						default  uint8?
						channels map<uint32, uint8>?
					}
				`
				resp := MustGenerate(ctx, source, "ir", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("uint32(*r.Default)").
					ToContain("uint8(*pb.Default)").
					ToContain("pb.Channels = make(map[uint32]uint32").
					ToContain("pb.Channels[k] = uint32(v)").
					ToContain("r.Channels = make(map[uint32]uint8").
					ToContain("r.Channels[k] = uint8(v)")
			})
		})

		Context("record field conversion", func() {
			It("Should handle record fields with structpb", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						data record
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("structpb.NewStruct(r.Data)").
					ToContain("pb.Data.AsMap()")
			})
		})

		Context("map with record value conversion", func() {
			It("Should generate error-returning forward loop for map<K, record>", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						bags map<uint32, record>?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("pb.Bags = make(map[uint32]*structpb.Struct").
					ToContain("converted, err := structpb.NewStruct(v)").
					ToContain("pb.Bags[k] = converted").
					ToContain("r.Bags = make(map[uint32]msgpack.EncodedJSON").
					ToContain("r.Bags[k] = msgpack.EncodedJSON(v.AsMap())")
			})
		})

		Context("map with struct value conversion", func() {
			It("Should generate error-returning loops in both directions for map<K, Struct>", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Inner struct {
						value uint32
					}

					Outer struct {
						items map<uint32, Inner>?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("pb.Items = make(map[uint32]*Inner").
					ToContain("converted, err := InnerToPB(v)").
					ToContain("pb.Items[k] = converted").
					ToContain("r.Items = make(map[uint32]test.Inner").
					ToContain("converted, err := InnerFromPB(v)").
					ToContain("r.Items[k] = converted")
			})
		})

		Context("any field conversion", func() {
			It("Should handle any fields with json.Marshal", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						value any
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("json.Marshal").
					ToContain("json.Unmarshal")
			})
		})
	})

	Describe("Check", func() {
		It("Should return nil (no-op)", func(ctx SpecContext) {
			req := MustGenerateRequest(ctx, `
				@go output "core/test"
				@pb

				Test struct { key uuid }
			`, "test", loader)
			Expect(pbPlugin.Check(req)).To(Succeed())
		})
	})

	Describe("primitive type conversions", func() {
		It("Should widen uint8 to uint32", func(ctx SpecContext) {
			source := `
				@go output "core/test"
				@pb

				Test struct {
					priority uint8
				}
			`
			resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

			ExpectContent(resp, "translator.gen.go").
				ToContain("uint32(r.Priority)").
				ToContain("uint8(pb.Priority)")
		})

		It("Should convert uint12 with types import", func(ctx SpecContext) {
			source := `
				@go output "core/test"
				@pb

				Test struct {
					value uint12
				}
			`
			resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

			ExpectContent(resp, "translator.gen.go").
				ToContain("uint32(r.Value)").
				ToContain("types.Uint12(pb.Value)")
		})

		It("Should convert uint20 with types import", func(ctx SpecContext) {
			source := `
				@go output "core/test"
				@pb

				Test struct {
					value uint20
				}
			`
			resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

			ExpectContent(resp, "translator.gen.go").
				ToContain("uint32(r.Value)").
				ToContain("types.Uint20(pb.Value)")
		})

		It("Should convert record field with structpb import", func(ctx SpecContext) {
			source := `
				@go output "core/test"
				@pb

				Test struct {
					metadata record
				}
			`
			resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

			ExpectContent(resp, "translator.gen.go").
				ToContain("structpb.NewStruct(r.Metadata)").
				ToContain("pb.Metadata.AsMap()")
		})
	})

	Describe("typedef delegation", func() {
		Context("typedef wrapping a struct", func() {
			It("Should generate delegation translator for typedef that wraps struct", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Base struct {
						key uuid
						name string
					}

					Custom = Base
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func BaseToPB")
			})
		})

		Context("cross-namespace typedef with numeric base", func() {
			BeforeEach(func() {
				loader.Add("schemas/ids", `
					@go output "core/ids"
					@pb

					NodeID = uint32
				`)
			})

			It("Should convert cross-namespace typedef with correct prefix", func(ctx SpecContext) {
				source := `
					import "schemas/ids"

					@go output "core/cluster"
					@pb

					Cluster struct {
						node_id ids.NodeID
					}
				`
				resp := MustGenerate(ctx, source, "cluster", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("uint32(r.NodeID)").
					ToContain("ids.NodeID(pb.NodeId)")
			})
		})
	})

	Describe("optional fields", func() {
		It("Should handle optional struct reference with pointer", func(ctx SpecContext) {
			source := `
				@go output "core/test"
				@pb

				Info struct {
					name string
				}

				Test struct {
					key uuid
					info Info?
				}
			`
			resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

			ExpectContent(resp, "translator.gen.go").
				ToContain("InfoToPB").
				ToContain("InfoFromPB")
		})

		It("Should wrap an optional struct array in a nullable wrapper message", func(ctx SpecContext) {
			source := `
				@go output "core/test"
				@pb

				Info struct {
					name string
				}

				Test struct {
					key uuid
					infos Info[]?
				}
			`
			resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

			ExpectContent(resp, "translator.gen.go").
				ToContain("if r.Infos != nil").
				ToContain("pb.Infos = &InfoList{Values:").
				ToContain("if pb.Infos != nil").
				ToContain("InfoListFromPB(pb.Infos.Values)")
		})

		It("Should wrap an optional array in a generic struct translator", func(ctx SpecContext) {
			source := `
				@go output "core/test"
				@pb

				Info struct {
					name string
				}

				Test struct {
					key uuid
					details D
					infos Info[]?
				}
			`
			resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

			ExpectContent(resp, "translator.gen.go").
				ToContain("pb.Infos = &InfoList{Values:").
				ToContain("InfoListFromPB(pb.Infos.Values)")
		})
	})

	Describe("Generate edge cases", func() {
		Context("type alias to struct", func() {
			It("Should handle alias pointing to struct type", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					User struct {
						key uuid
						name string
					}

					Person = User

					Container struct {
						key uuid
						person Person
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func UserToPB").
					ToContain("func ContainerToPB")
			})
		})

		Context("struct with extends", func() {
			It("Should handle struct that extends another", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Base struct {
						key uuid
						name string
					}

					Derived struct extends Base {
						extra string
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func BaseToPB").
					ToContain("func DerivedToPB").
					ToContain("Extra: r.Extra")
			})
		})

		Context("struct array references", func() {
			It("Should use slice translator for struct array with error handling", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Item struct {
						key uuid
					}

					List struct {
						key uuid
						items Item[]
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("ItemsToPB(r.Items)").
					ToContain("if err != nil")
			})
		})

		Context("generic struct with type args", func() {
			It("Should generate generic translator with converter functions", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Wrapper struct<T> {
						key uuid
						data T
					}

					User struct {
						key uuid
					}

					UserWrapper = Wrapper<User>
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("func WrapperToPB[T any]").
					ToContain("translateT func")
			})
		})

		Context("optional fields", func() {
			It("Should round-trip an optional scalar through a pointer", func(ctx SpecContext) {
				source := `
					@go output "core/test"
					@pb

					Test struct {
						key uuid
						name string?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"if r.Name != nil {",
						"pb.Name = r.Name",
						"if pb.Name != nil {",
						"r.Name = pb.Name",
					)
			})

			It("Should round-trip an optional struct as a nullable wire field", func(ctx SpecContext) {
				// A struct field with a "?" is nullable: its Go type is a
				// pointer and the proto field is `optional`. The translator
				// guards both directions on the pointer, converting only when
				// the value is present.
				source := `
					@go output "core/test"
					@pb

					Side enum {
						left  = "left"
						right = "right"
					}

					Anchor struct {
						side Side
					}

					Test struct {
						key    uuid
						anchor Anchor?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"if r.Anchor != nil {",
						"pb.Anchor, err = AnchorToPB(*r.Anchor)",
						"if pb.Anchor != nil {",
						"val, err := AnchorFromPB(pb.Anchor)",
						"r.Anchor = &val",
					).
					ToNotContain(
						"if r.Anchor != (test.Anchor{}) {",
					)
			})
		})

		Context("cross-namespace struct reference", func() {
			BeforeEach(func() {
				loader.Add("schemas/common", `
					@go output "core/common"
					@pb

					Info struct {
						key uuid
						description string
					}
				`)
			})

			It("Should import pb package for cross-namespace struct", func(ctx SpecContext) {
				source := `
					import "schemas/common"

					@go output "core/test"
					@pb

					Test struct {
						key uuid
						info common.Info
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("InfoToPB(r.Info)").
					ToContain("InfoFromPB(pb.Info)")
			})
		})

		Context("enum in different namespace", func() {
			BeforeEach(func() {
				loader.Add("schemas/status", `
					@go output "core/status"
					@pb

					Status enum {
						unknown = 0
						active = 1
					}
				`)
			})

			It("Should import pb package for cross-namespace enum", func(ctx SpecContext) {
				source := `
					import "schemas/status"

					@go output "core/task"
					@pb

					Task struct {
						key uuid
						status status.Status
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain("StatusToPB(r.Status)")
			})
		})

		Context("distinct type wrapping a primitive", func() {
			It("Should cast in and out of the distinct Go type via protoType", func(ctx SpecContext) {
				source := `
					@go output "core/task"
					@pb

					Priority uint16

					Task struct {
						key uuid
						priority Priority
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						// uint16 → uint32 on the wire, same-package alias prefix.
						"uint32(r.Priority)",
						"task.Priority(pb.Priority)",
					)
			})

			It("Should use uuid.Parse when distinct wraps a uuid", func(ctx SpecContext) {
				source := `
					@go output "core/task"
					@pb

					TaskID uuid

					Task struct {
						id TaskID
						name string
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						// Forward calls .String(); backward parses and casts to the
						// distinct wrapper.
						"r.ID.String()",
						"uuid.Parse(pb.Id)",
						"task.TaskID",
					)
			})

			It("Should emit lo.Map casts for array of same-namespace distinct primitive", func(ctx SpecContext) {
				source := `
					@go output "core/task"
					@pb

					Priority uint16

					Task struct {
						priorities Priority[]
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"lo.Map(r.Priorities, func(v task.Priority, _ int) uint32 { return uint32(v) })",
						"lo.Map(pb.Priorities, func(v uint32, _ int) task.Priority { return task.Priority(v) })",
					)
			})

			It("Should emit lo.Map casts with package alias for array of cross-namespace distinct primitive", func(ctx SpecContext) {
				loader.Add("schemas/channel", `
					@go output "core/channel"
					@pb

					Key uint32
				`)

				source := `
					import "schemas/channel"

					@go output "core/series"
					@pb

					Series struct {
						keys channel.Key[]
					}
				`
				resp := MustGenerate(ctx, source, "series", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"lo.Map(r.Keys, func(v channel.Key, _ int) uint32 { return uint32(v) })",
						"lo.Map(pb.Keys, func(v uint32, _ int) channel.Key { return channel.Key(v) })",
					)
			})
		})

		Context("fixed-size uint8 array", func() {
			It("Should delegate to the distinct type's Bytes/FromBytes helpers", func(ctx SpecContext) {
				source := `
					@go output "core/crypto"
					@pb

					Hash uint8[32]

					Digest struct {
						hash Hash
					}
				`
				resp := MustGenerate(ctx, source, "crypto", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"r.Hash.Bytes()",
						".FromBytes(pb.Hash)",
					)
			})
		})

		Context("delegation translator (distinct wrapping a struct)", func() {
			It("Should generate a delegating translator that calls the base type's ToPB/FromPB", func(ctx SpecContext) {
				loader.Add("schemas/ranger", `
					@go output "core/ranger"
					@pb output "core/ranger/pb"

					Range struct {
						key  uuid
						name string
					}
				`)

				source := `
					import "schemas/ranger"

					@go output "core/ranger/ts"
					@pb output "core/ranger/ts/pb"

					TSRange ranger.Range
				`
				resp := MustGenerate(ctx, source, "tsranger", loader, pbPlugin)

				// Delegation translator lives in the TSRange output path and
				// calls through to the base ranger.Range translator.
				ExpectContent(resp, "core/ranger/ts/pb/translator.gen.go").
					ToContain(
						"ranger_pb.RangeToPB(ranger.Range(r))",
						"ranger_pb.RangeFromPB",
						"ts.TSRange(result)",
					)
			})
		})

		Context("struct with @key numeric field", func() {
			It("Should cast through the Key type via protoType and namespace alias", func(ctx SpecContext) {
				loader.Add("schemas/ids", `
					@go output "core/ids"
					@pb

					Key uint32 {
						@doc value "is an identifier"
					}
				`)

				source := `
					import "schemas/ids"

					@go output "core/node"
					@pb

					Node struct {
						id ids.Key {
							@key
						}
						name string
					}
				`
				resp := MustGenerate(ctx, source, "node", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						"uint32(r.ID)",
						"ids.Key(pb.Id)",
					)
			})

			It("Should cast a bare numeric primitive @key field through the parent package alias", func(ctx SpecContext) {
				source := `
					@go output "core/node"
					@pb

					Node struct {
						id uint32 {
							@key
						}
						name string
					}
				`
				resp := MustGenerate(ctx, source, "node", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						// isNumericPrimitive branch: forward is protoType(goField),
						// backward casts the pb value through <pkg>.Key(...).
						"uint32(r.ID)",
						"node.Key(pb.Id)",
					)
			})
		})

		Context("generic struct instantiated with a struct type arg", func() {
			It("Should keep explicit instantiation only for backward conversion with nil converters", func(ctx SpecContext) {
				source := `
					@go output "core/control"
					@pb

					Status struct<D?> {
						key     string
						details D
					}

					ChannelStatus = Status<nil>

					Channel struct {
						status ChannelStatus?
					}
				`
				resp := MustGenerate(ctx, source, "control", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						// The typed argument pins the forward instantiation; the
						// nil converter cannot pin the backward one.
						"StatusToPB(*r.Status, nil)",
						"StatusFromPB[gotypes.Nil](pb.Status, nil)",
					)
			})

			It("Should emit ToPBAny/FromPBAny helpers for the struct-typed instantiation", func(ctx SpecContext) {
				source := `
					@go output "core/control"
					@pb

					Details struct {
						label string
					}

					Status struct<D> {
						key     string
						details D
					}

					Channel struct {
						status Status<Details>
					}
				`
				resp := MustGenerate(ctx, source, "control", loader, pbPlugin)

				ExpectContent(resp, "translator.gen.go").
					ToContain(
						// ensureAnyHelper appends a ToPBAny/FromPBAny helper pair
						// for each struct-typed generic instantiation.
						"DetailsToPBAny",
						"DetailsFromPBAny",
						// Generic struct conversion forwards the typed converters;
						// the argument and converters pin the instantiation.
						"StatusToPB(r.Status, DetailsToPBAny)",
						"StatusFromPB(pb.Status, DetailsFromPBAny)",
					)
			})
		})

		Context("cross-schema namespace collision", func() {
			It("Should not emit a non-@pb enum into another schema's pb translator when both schemas derive the same namespace", func(ctx SpecContext) {
				// schemas/text.oracle and schemas/arc/text.oracle both
				// derive namespace "text" via DeriveNamespace. The first
				// declares Level without @pb. The second is @pb with its
				// own type. Level must not bleed into arc/text/pb because
				// its declaring file is not opted into pb.
				loader.Add("schemas/text", `
					@go output "x/go/text"

					Level enum {
						h1 = "h1"
						h2 = "h2"
					}
				`)
				loader.Add("schemas/arc/text", `
					@go output "arc/go/text"
					@pb

					Text struct {
						content string
					}
				`)
				resp := MustGenerateMulti(ctx, loader, pbPlugin)

				var translatorFiles []string
				for _, f := range resp.Files {
					translatorFiles = append(translatorFiles, f.Path)
					if strings.HasSuffix(f.Path, "/translator.gen.go") {
						Expect(string(f.Content)).ToNot(
							ContainSubstring("Level"),
							"non-@pb Level enum bled into pb output at %s",
							f.Path,
						)
					}
				}
				Expect(translatorFiles).To(ConsistOf("arc/go/text/pb/translator.gen.go"))
			})
		})

		Context("enum-only @pb schema", func() {
			It("Should emit a translator file for a schema that declares only @pb enums", func(ctx SpecContext) {
				// Without this, a dependent schema's @pb struct that
				// references the enum has no foreign translator to call.
				loader.Add("schemas/text", `
					@go output "x/go/text"
					@pb

					Level enum {
						h1    = "h1"
						h2    = "h2"
						small = "small"
					}
				`)
				resp := MustGenerateMulti(ctx, loader, pbPlugin)

				Expect(resp.Files).To(HaveLen(1))
				Expect(resp.Files[0].Path).To(Equal("x/go/text/pb/translator.gen.go"))
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring("func LevelToPB"))
				Expect(content).To(ContainSubstring("func LevelFromPB"))
				Expect(content).To(ContainSubstring("text.LevelH1"))
			})

			It("Should call the foreign translator when an @pb struct references a cross-namespace @pb enum", func(ctx SpecContext) {
				loader.Add("schemas/text", `
					@go output "x/go/text"
					@pb

					Level enum {
						h1 = "h1"
						h2 = "h2"
					}
				`)
				loader.Add("schemas/lineplot", `
					import "schemas/text"

					@go output "core/lineplot"
					@pb

					Title struct {
						level text.Level
					}
				`)
				resp := MustGenerateMulti(ctx, loader, pbPlugin)

				var lineplotTranslator string
				for _, f := range resp.Files {
					if f.Path == "core/lineplot/pb/translator.gen.go" {
						lineplotTranslator = string(f.Content)
					}
				}
				Expect(lineplotTranslator).ToNot(BeEmpty())
				Expect(lineplotTranslator).To(ContainSubstring("textpb.LevelToPB"))
				Expect(lineplotTranslator).To(ContainSubstring("textpb.LevelFromPB"))
				Expect(lineplotTranslator).ToNot(ContainSubstring("func LevelToPB"))
			})
		})
	})
})

var _ = ShouldNotLeakGoroutinesPerSpec()
