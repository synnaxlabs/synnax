// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package marshal_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin/go/marshal"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

func TestGoMarshal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Go Marshal Suite")
}

var _ = Describe("Go Marshal Plugin", func() {
	var (
		ctx           context.Context
		loader        *MockFileLoader
		marshalPlugin *marshal.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = NewMockFileLoader()
		marshalPlugin = marshal.New(marshal.DefaultOptions())
	})

	Describe("Generate", func() {
		Context("simple struct with string and int fields", func() {
			It("Should generate EncodeOrc/DecodeOrc methods", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name string
						age int32
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"package test",
						"w.String(t.Name)",
						"w.Int32(int32(t.Age))",
						"func (t Test) EncodeOrc(w *orc.Writer",
						"func (t *Test) DecodeOrc(r *orc.Reader",
					)
			})
		})

		Context("nested struct (same package delegation)", func() {
			It("Should delegate to nested struct EncodeOrc/DecodeOrc methods", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Inner struct {
						type string
						key string

						@go omit
					}

					Outer struct {
						from Inner
						name string
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"o.From.EncodeOrc(w)",
						"o.From.DecodeOrc(r)",
						"o.Name",
					)
			})
		})

		Context("optional field", func() {
			It("Should generate presence flag for pointer-based optional", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name string
						description string?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain("if t.Description != nil {")
			})

			It("Should decode optional string into a non-shadowing temp var", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name        string
						description string?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain("var hv string")
				content.ToContain("t.Description = &hv")
			})

			It("Should compile a optional string-based enum without shadowing the outer pointer target", func() {
				// Regression: the decode template for a string-based enum
				// emitted "{ v, err := r.String(); v = TickType(v) }" which
				// shadows the outer "var v TickType" declared by the hard-
				// optional wrapper, breaks compilation, and leaves the outer v
				// at its zero value. The fix renames the wrapper's outer var so
				// the inner short-declaration cannot collide.
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					TickType enum {
						linear = "linear"
						time   = "time"
					}

					Axis struct {
						label string
						type  TickType?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain("var hv TickType")
				content.ToContain("hv = TickType(v)")
				content.ToContain("a.Type = &hv")
				content.ToNotContain("var v TickType")
			})

			It("Should compile a optional integer-based enum without shadowing the outer pointer target", func() {
				// Same regression class as the string case but exercising the
				// integer leaf decoder, which uses the same shared inner var
				// name and would have collided the same way under a hard-
				// optional wrapper.
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Level enum {
						low    = 0
						medium = 1
						high   = 2
					}

					Item struct {
						name  string
						level Level?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain("var hv Level")
				content.ToContain("iv.Level = &hv")
				content.ToContain("hv = Level(v)")
				content.ToNotContain("var v Level")
			})
		})

		Context("generic struct with nil type arg via alias", func() {
			It("Should skip nil-typed fields and resolve defaulted type params", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Variant enum {
						info    = "info"
						warning = "warning"
						error   = "error"
					}

					Status struct<Details?, V extends Variant = Variant> {
						key     string
						variant V
						details Details?
					}

					MyStatus = Status<nil>

					Test struct {
						name   string
						status MyStatus?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain("func (t Test) EncodeOrc(w *orc.Writer")
			})
		})

		Context("defaulted type param should encode as concrete type, not JSON fallback", func() {
			It("Should encode a defaulted enum type param as a string, not via JSON marshal", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Variant enum {
						info    = "info"
						warning = "warning"
						error   = "error"
					}

					Status struct<Details?, V extends Variant = Variant> {
						key     string
						variant V
						details Details?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"w.String(string(s.Variant))",
				)
				content.ToNotContain(
					"json.Marshal(s.Variant)",
				)
			})
		})

		Context("extending enum as a struct field", func() {
			It("Should encode/decode an extending enum field as a plain string enum", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

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
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain("w.String(string(p.AxisKey))")
				content.ToContain("p.AxisKey = AxisKey(v)")
			})
		})

		Context("non-optional array alias field", func() {
			It("Should handle a type alias that wraps an array", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Node struct {
						key  string
						type string

						@go omit
					}

					Nodes = Node[]

					Graph struct {
						nodes Nodes
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain("func (g Graph) EncodeOrc(w *orc.Writer")
			})
		})

		Context("nested array alias (alias to alias of array)", func() {
			It("Should handle Strata = Stratum[] where Stratum = string[]", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Stratum = string[]

					Strata = Stratum[]

					Test struct {
						name   string
						strata Strata?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain("func (t Test) EncodeOrc(w *orc.Writer")
			})
		})

		Context("generic struct with concrete type arg via alias", func() {
			It("Should inline the concrete type arg struct fields", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Wrapper struct<T?> {
						key   string
						value T?
					}

					Details struct {
						reason string

						@go omit
					}

					MyWrapper = Wrapper<Details>

					Test struct {
						name    string
						wrapper MyWrapper?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain("func (t Test) EncodeOrc(w *orc.Writer")
			})
		})

		Context("array field nil preservation", func() {
			It("Should generate a presence bit before the array length", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name  string
						items string[]
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"w.Bool(t.Items != nil)",
						"if t.Items != nil {",
						"present, err := r.Bool()",
						"if present {",
					)
			})
		})

		Context("map field nil preservation", func() {
			It("Should dispatch union map values through the wrapper codec", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					TankConfig struct { width float64 }

					ElementConfig union on variant {
						tank TankConfig
					}

					Test struct {
						configs map<string, ElementConfig>
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"if err := val.EncodeOrc(w); err != nil { return err }",
						"if err = val.DecodeOrc(r); err != nil { return err }",
					).
					ToNotContain("json.Marshal(val)")
			})

			It("Should generate a presence bit before the map length", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name   string
						labels map<string, string>
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"w.Bool(t.Labels != nil)",
						"if t.Labels != nil {",
						"present, err := r.Bool()",
						"if present {",
					)
			})
		})

		Context("bytes field nil preservation", func() {
			It("Should generate a presence bit before the byte slice length", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name string
						data bytes
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"w.Bool(t.Data != nil)",
						"if t.Data != nil {",
						"present, err := r.Bool()",
						"if present {",
					)
			})
		})

		Context("marshal omit on a field", func() {
			It("Should exclude the field from encoding and decoding", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name string
						data record? {
							@go marshal omit
						}
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain("t.Name")
				content.ToNotContain("Data")
			})
		})

		Context("marshal json_only on a type param field", func() {
			It("Should always use JSON encoding without SelfEncoder/SelfDecoder type assertions", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Status struct<Details?> {
						key     string
						details Details? {
							@go marshal json_only
						}
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"json.Marshal(s.Details)",
					"json.Unmarshal(b, &s.Details)",
				)
				content.ToNotContain(
					"orc.SelfEncoder",
					"orc.SelfDecoder",
				)
			})
		})

		Context("optional array field", func() {
			It("Should generate a single presence bit without a redundant inner nil check", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name  string
						items string[]?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"if t.Items != nil {",
					"w.Bool(true)",
					"w.Uint32(uint32(len(t.Items)))",
				)
				content.ToNotContain(
					"w.Bool(t.Items != nil)",
				)
			})
		})

		Context("optional map field", func() {
			It("Should generate a single presence bit without a redundant inner nil check", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name   string
						labels map<string, string>?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"if t.Labels != nil {",
					"w.Bool(true)",
					"w.Uint32(uint32(len(t.Labels)))",
				)
				content.ToNotContain(
					"w.Bool(t.Labels != nil)",
				)
			})
		})

		Context("optional struct-array field", func() {
			It("Should encode the slice in place without a pointer deref", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Inner struct {
						name string
						@go omit
					}

					Test struct {
						name  string
						items Inner[]?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"if t.Items != nil {",
					"w.Bool(true)",
					"w.Uint32(uint32(len(t.Items)))",
				)
				content.ToNotContain(
					"(*t.Items)",
				)
			})
		})

		Context("marshal flex on a distinct scalar type", func() {
			It("Should generate DecodeMsgpack and UnmarshalJSON methods", func() {
				source := `
					@go output "core/pkg/test"
					@pb

					Key uint64 {
						@go marshal flex
					}

					Inner struct {
						task Key
						@go marshal
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"func (kv *Key) DecodeMsgpack(dec *msgpack.Decoder) error",
					"xmsgpack.UnmarshalUint64",
					"func (kv *Key) UnmarshalJSON(b []byte) error",
					"xjson.UnmarshalStringUint64",
				)
			})

			It("Should generate uint32 helpers for uint32 base types", func() {
				source := `
					@go output "core/pkg/test"
					@pb

					Key uint32 {
						@go marshal flex
					}

					Inner struct {
						rack Key
						@go marshal
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"func (kv *Key) DecodeMsgpack(dec *msgpack.Decoder) error",
					"xmsgpack.UnmarshalUint32",
					"func (kv *Key) UnmarshalJSON(b []byte) error",
					"xjson.UnmarshalStringUint32",
				)
			})
		})

		Context("recursive struct (self-referencing optional fields)", func() {
			It("Should handle recursive type via delegation", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Type struct {
						name string
						elem Type?
					}

					Container struct {
						key  string
						type Type
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"func (c Container) EncodeOrc(w *orc.Writer",
						"c.Type.EncodeOrc(w)",
						"c.Type.DecodeOrc(r)",
					)
			})
		})

		Context("deterministic output ordering", func() {
			It("Should order codec methods alphabetically by qualified name", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Zebra struct {
						name string
					}

					Alpha struct {
						key string
					}

					Middle struct {
						id    uint64
						zebra Zebra
						alpha Alpha
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToPreserveOrder(
						"Alpha) EncodeOrc",
						"Alpha) DecodeOrc",
						"Middle) EncodeOrc",
						"Middle) DecodeOrc",
						"Zebra) EncodeOrc",
						"Zebra) DecodeOrc",
					)
			})

			It("Should emit the google/uuid import exactly once when uuid-typed fields are present", func() {
				// Regression: the test fixture generator both set
				// NeedsUUID (which the template renders as a hardcoded
				// `"github.com/google/uuid"` line) and registered the same
				// import under ExtraImports with an explicit "uuid" alias,
				// producing two import lines for the same path and breaking
				// the generated test file with a "uuid redeclared" compile
				// error.
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Entry struct {
						key  uuid
						name string
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec_gen_test.go")
				content.ToContain(`"github.com/google/uuid"`)
				content.ToNotContain(`uuid "github.com/google/uuid"`)
			})

			It("Should order test Describe blocks alphabetically by qualified name", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Zebra struct {
						name string
					}

					Alpha struct {
						key string
					}

					Middle struct {
						id    uint64
						zebra Zebra
						alpha Alpha
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec_gen_test.go").
					ToPreserveOrder(
						`Describe("Alpha"`,
						`Describe("Middle"`,
						`Describe("Zebra"`,
					)
			})

			It("Should order flex codec methods alphabetically", func() {
				source := `
					@go output "core/pkg/test"
					@pb

					Zulu uint64 {
						@go marshal flex
					}

					Bravo uint32 {
						@go marshal flex
					}

					Inner struct {
						task Zulu
						tag  Bravo
						@go marshal
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToPreserveOrder(
						"Bravo) DecodeMsgpack",
						"Bravo) UnmarshalJSON",
						"Zulu) DecodeMsgpack",
						"Zulu) UnmarshalJSON",
					)
			})

			It("Should order extra imports alphabetically", func() {
				source := `
					@go output "core/pkg/test"
					@pb

					Key uint64 {
						@go marshal flex
					}

					Inner struct {
						task Key
						@go marshal
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToPreserveOrder(
						`"github.com/synnaxlabs/x/encoding/json"`,
						`"github.com/synnaxlabs/x/encoding/msgpack"`,
						`"github.com/vmihailenco/msgpack/v5"`,
					)
			})
		})
	})
})

var _ = Describe("Union Codecs", func() {
	var (
		loader        *MockFileLoader
		marshalPlugin *marshal.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		marshalPlugin = marshal.New(marshal.DefaultOptions())
	})

	It("Should generate a binary wrapper codec with base and payload delegation", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			BaseElement struct { key string }
			TankConfig struct { width float64 }
			ValveConfig struct {}

			ElementConfig union on variant extends BaseElement {
				tank  TankConfig
				valve ValveConfig
			}

			Test struct {
				config ElementConfig
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		ExpectContent(resp, "codec.gen.go").
			ToContain(
				"func (ec ElementConfig) EncodeOrc(w *orc.Writer) error {",
				"switch v := ec.Variant.(type) {",
				"case ElementConfigTank:",
				`w.String("tank")`,
				"if err := v.BaseElement.EncodeOrc(w); err != nil { return err }",
				"if err := v.TankConfig.EncodeOrc(w); err != nil { return err }",
				"case ElementConfigValve:",
				`w.String("valve")`,
				`return errors.Newf("ElementConfig: nil or unknown variant %T", ec.Variant)`,
				"func (ec *ElementConfig) DecodeOrc(r *orc.Reader) error {",
				"tag, err := r.String()",
				`case "tank":`,
				"var v ElementConfigTank",
				"if err := v.BaseElement.DecodeOrc(r); err != nil { return err }",
				"if err := v.TankConfig.DecodeOrc(r); err != nil { return err }",
				"ec.Variant = v",
				`return errors.Newf("ElementConfig: unknown variant %q", tag)`,
				"if err := t.Config.EncodeOrc(w); err != nil { return err }",
				`"github.com/synnaxlabs/x/errors"`,
			).
			ToNotContain("json.Marshal")
	})

	It("Should generate codecs for structs reachable only through a union", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			TankConfig struct { width float64 }

			ElementConfig union on variant {
				tank TankConfig
			}

			Test struct {
				config ElementConfig
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		ExpectContent(resp, "codec.gen.go").
			ToContain(
				"func (tc TankConfig) EncodeOrc(w *orc.Writer) error {",
				"w.Float64(float64(tc.Width))",
			)
	})

	It("Should generate round-trip, benchmark, and fuzz harnesses covering every variant", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			BaseElement struct { key string }
			TankConfig struct { width float64 }
			ValveConfig struct { open bool }

			ElementConfig union on variant extends BaseElement {
				tank  TankConfig
				valve ValveConfig
			}

			Test struct {
				config ElementConfig
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		ExpectContent(resp, "codec_gen_test.go").
			ToBeValidGoSource().
			ToContain(
				`Describe("ElementConfig"`,
				`Entry("tank variant"`,
				`Entry("valve variant"`,
				"test.ElementConfigTank{",
				"test.ElementConfigValve{",
				"BaseElement: fullyPopulatedBaseElement",
				"func BenchmarkEncodeDecodeElementConfig(b *testing.B) {",
				"func FuzzDecodeElementConfig(f *testing.F) {",
			)
	})

	It("Should encode inline variant fields directly in the union codec", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			TabBase struct { key string }

			Tab union on variant extends TabBase {
				view {
					type string
				}
				empty {}
			}

			Test struct {
				tab Tab
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		content := ExpectContent(resp, "codec.gen.go")
		content.ToBeValidGoSource()
		content.ToContain(
			`case TabView:`,
			"if err := v.TabBase.EncodeOrc(w); err != nil { return err }",
			"w.String(v.Type)",
			"if err := v.TabBase.DecodeOrc(r); err != nil { return err }",
			"v.Type, err = r.String()",
		)
		content.ToNotContain("TabViewPayload")
		ExpectContent(resp, "codec_gen_test.go").
			ToBeValidGoSource().
			ToContain(
				`Entry("view variant"`,
				`Entry("empty variant"`,
				"TabBase: fullyPopulatedTabBase",
				"Type:",
			)
	})

	It("Should share fully populated fixtures between union entries and type tables", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			TankConfig struct { width float64 }

			ElementConfig union on variant {
				tank       TankConfig
				other_tank TankConfig
			}

			Test struct {
				config ElementConfig
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		ExpectContent(resp, "codec_gen_test.go").
			ToBeValidGoSource().
			ToContain(
				"fullyPopulatedTankConfig = test.TankConfig{",
				"TankConfig: fullyPopulatedTankConfig",
				`Entry("fully populated", fullyPopulatedTankConfig)`,
			).
			ToNotContain("TankConfig: test.TankConfig{Width: 1.5}")
	})
})

var _ = Describe("Recursive Codecs", func() {
	var (
		loader        *MockFileLoader
		marshalPlugin *marshal.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		marshalPlugin = marshal.New(marshal.DefaultOptions())
	})

	It("Should guard a recursive struct's decode with a depth limit", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			Node struct {
				name     string
				children Node[]
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		ExpectContent(resp, "codec.gen.go").
			ToBeValidGoSource().
			ToContain(
				`func (nv *Node) DecodeOrc(r *orc.Reader) error {
	if err := r.PushDepth(orc.MaxDecodeDepth); err != nil {
		return err
	}
	defer r.PopDepth()`,
			).
			ToNotContain("EncodeOrc(w *orc.Writer) error {\n\tif err := r.PushDepth")
	})

	It("Should guard a recursive union and its cycle members with a depth limit", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			LeafConfig struct { name string }
			GroupConfig struct { children ElementConfig[] }

			ElementConfig union on variant {
				leaf  LeafConfig
				group GroupConfig
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		ExpectContent(resp, "codec.gen.go").
			ToBeValidGoSource().
			ToContain(
				`func (ec *ElementConfig) DecodeOrc(r *orc.Reader) error {
	if err := r.PushDepth(orc.MaxDecodeDepth); err != nil {
		return err
	}
	defer r.PopDepth()
	tag, err := r.String()`,
				`func (gc *GroupConfig) DecodeOrc(r *orc.Reader) error {
	if err := r.PushDepth(orc.MaxDecodeDepth); err != nil {
		return err
	}
	defer r.PopDepth()`,
				`func (lc *LeafConfig) DecodeOrc(r *orc.Reader) error {
	var err error`,
			)
	})

	It("Should not guard non-recursive decodes", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			TankConfig struct { width float64 }

			ElementConfig union on variant {
				tank TankConfig
			}

			Test struct {
				config ElementConfig
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		ExpectContent(resp, "codec.gen.go").ToNotContain("PushDepth")
	})
})

var _ = Describe("Version-Laid-Out Packages", func() {
	var (
		ctx           context.Context
		loader        *MockFileLoader
		marshalPlugin *marshal.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = NewMockFileLoader()
		marshalPlugin = marshal.New(marshal.DefaultOptions())
	})

	It("Should emit the codec and its test into types/vN", func() {
		source := `
			@go output "out"
			@go version 3
			@pb

			Entry struct {
				key uuid @key
				name string
				@go marshal
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		ExpectContent(resp, "out/types/v3/codec.gen.go").
			ToBeValidGoSource().
			ToContain(
				"package v3",
				"func (e Entry) EncodeOrc(w *orc.Writer",
			)
		ExpectContent(resp, "out/types/v3/codec_gen_test.go").
			ToBeValidGoSource().
			ToContain("package v3_test")
	})

	It("Should leave non-versioned packages at the package root", func() {
		source := `
			@go output "core/pkg/test"
			@go marshal
			@pb

			Test struct {
				key uuid @key
				name string
			}
		`
		resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
		Expect(resp.Files[0].Path).To(Equal("core/pkg/test/codec.gen.go"))
		ExpectContent(resp, "core/pkg/test/codec.gen.go").
			ToContain("package test").
			ToNotContain("types/v")
	})
})

// The rt* fixtures mirror the exact shape the generator emits for a
// discriminated union wrapper codec (binary discriminator tag followed by the
// variant's base and payload codecs), locking the runtime semantics the
// substring tests above cannot: round-trip equality, the nil-variant encode
// error, and the unknown-tag decode error.
type rtBase struct{ Key string }

func (b rtBase) EncodeOrc(w *orc.Writer) error {
	w.String(b.Key)
	return nil
}

func (b *rtBase) DecodeOrc(r *orc.Reader) error {
	var err error
	if b.Key, err = r.String(); err != nil {
		return err
	}
	return nil
}

type rtTank struct{ Width float64 }

func (t rtTank) EncodeOrc(w *orc.Writer) error {
	w.Float64(t.Width)
	return nil
}

func (t *rtTank) DecodeOrc(r *orc.Reader) error {
	var err error
	if t.Width, err = r.Float64(); err != nil {
		return err
	}
	return nil
}

type rtValve struct{ Open bool }

func (v rtValve) EncodeOrc(w *orc.Writer) error {
	w.Bool(v.Open)
	return nil
}

func (v *rtValve) DecodeOrc(r *orc.Reader) error {
	var err error
	if v.Open, err = r.Bool(); err != nil {
		return err
	}
	return nil
}

type rtConfigVariant interface{ isRTConfigVariant() }

type rtConfigTank struct {
	rtBase
	rtTank
}

func (rtConfigTank) isRTConfigVariant() {}

type rtConfigValve struct {
	rtBase
	rtValve
}

func (rtConfigValve) isRTConfigVariant() {}

type rtConfig struct{ Variant rtConfigVariant }

func (c rtConfig) EncodeOrc(w *orc.Writer) error {
	switch v := c.Variant.(type) {
	case rtConfigTank:
		w.String("tank")
		if err := v.rtBase.EncodeOrc(w); err != nil {
			return err
		}
		if err := v.rtTank.EncodeOrc(w); err != nil {
			return err
		}
	case rtConfigValve:
		w.String("valve")
		if err := v.rtBase.EncodeOrc(w); err != nil {
			return err
		}
		if err := v.rtValve.EncodeOrc(w); err != nil {
			return err
		}
	default:
		return errors.Newf("rtConfig: nil or unknown variant %T", c.Variant)
	}
	return nil
}

func (c *rtConfig) DecodeOrc(r *orc.Reader) error {
	tag, err := r.String()
	if err != nil {
		return err
	}
	switch tag {
	case "tank":
		var v rtConfigTank
		if err := v.rtBase.DecodeOrc(r); err != nil {
			return err
		}
		if err := v.rtTank.DecodeOrc(r); err != nil {
			return err
		}
		c.Variant = v
	case "valve":
		var v rtConfigValve
		if err := v.rtBase.DecodeOrc(r); err != nil {
			return err
		}
		if err := v.rtValve.DecodeOrc(r); err != nil {
			return err
		}
		c.Variant = v
	default:
		return errors.Newf("rtConfig: unknown variant %q", tag)
	}
	return nil
}

var _ = Describe("Union Codec Round Trip", func() {
	DescribeTable("should round-trip each variant through the orc wire form",
		func(in rtConfig) {
			w := orc.NewWriter(0)
			Expect(in.EncodeOrc(w)).To(Succeed())
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			var out rtConfig
			Expect(out.DecodeOrc(r)).To(Succeed())
			Expect(out).To(Equal(in))
		},
		Entry("tank variant", rtConfig{Variant: rtConfigTank{
			rtBase{Key: "t1"}, rtTank{Width: 1.5},
		}}),
		Entry("valve variant", rtConfig{Variant: rtConfigValve{
			rtBase{Key: "v1"}, rtValve{Open: true},
		}}),
	)

	It("Should reject encoding a nil variant", func() {
		w := orc.NewWriter(0)
		Expect(rtConfig{}.EncodeOrc(w)).To(MatchError(ContainSubstring("nil or unknown variant")))
	})

	It("Should reject decoding an unknown discriminator tag", func() {
		w := orc.NewWriter(0)
		w.String("bogus")
		r := orc.NewReader(nil)
		r.ResetBytes(w.Bytes())
		var out rtConfig
		Expect(out.DecodeOrc(r)).To(MatchError(ContainSubstring(`unknown variant "bogus"`)))
	})
})

// rtNode mirrors the depth guard the generator emits for recursive types,
// locking the runtime semantics: input nested past orc.MaxDecodeDepth fails
// with ErrRecursionDepth instead of growing the stack without bound.
type rtNode struct{ Children []rtNode }

func (n rtNode) EncodeOrc(w *orc.Writer) error {
	w.Uint32(uint32(len(n.Children)))
	for _, c := range n.Children {
		if err := c.EncodeOrc(w); err != nil {
			return err
		}
	}
	return nil
}

func (n *rtNode) DecodeOrc(r *orc.Reader) error {
	if err := r.PushDepth(orc.MaxDecodeDepth); err != nil {
		return err
	}
	defer r.PopDepth()
	count, err := r.CollectionLen()
	if err != nil || count == 0 {
		return err
	}
	n.Children = make([]rtNode, count)
	for i := range n.Children {
		if err := n.Children[i].DecodeOrc(r); err != nil {
			return err
		}
	}
	return nil
}

var _ = Describe("Recursive Codec Depth Guard", func() {
	It("Should round-trip nesting within the depth limit", func() {
		in := rtNode{Children: []rtNode{{Children: []rtNode{{}}}, {}}}
		w := orc.NewWriter(0)
		Expect(in.EncodeOrc(w)).To(Succeed())
		r := orc.NewReader(nil)
		r.ResetBytes(w.Bytes())
		var out rtNode
		Expect(out.DecodeOrc(r)).To(Succeed())
		Expect(out).To(Equal(in))
	})

	It("Should reject input nested past the depth limit", func() {
		w := orc.NewWriter(0)
		for range orc.MaxDecodeDepth {
			w.Uint32(1)
		}
		w.Uint32(0)
		r := orc.NewReader(nil)
		r.ResetBytes(w.Bytes())
		var out rtNode
		Expect(out.DecodeOrc(r)).To(MatchError(orc.ErrRecursionDepth))
	})
})

var _ = Describe("Predecessor Aliasing", func() {
	It("Should skip codecs for types aliased to the predecessor version", func(ctx SpecContext) {
		loader := NewMockFileLoader()
		oldSource := `
			@go output "out"
			@go version 0
			Inner struct { value int32 }
			Entry struct {
				name string
				inner Inner
				@go marshal
			}
		`
		newSource := `
			@go output "out"
			@go version 1
			Inner struct { value int32 }
			Entry struct {
				name string
				label string
				inner Inner
				@go marshal
			}
		`
		req := MustGenerateRequest(ctx, newSource, "test", loader)
		req.SnapshotVersion = 56
		req.LoadSnapshot = func(version int) (*resolution.Table, error) {
			if version != 56 {
				return nil, nil
			}
			table, diag := analyzer.AnalyzeSource(
				ctx, oldSource, "test", NewMockFileLoader(),
			)
			if diag != nil && !diag.Ok() {
				return nil, diag
			}
			return table, nil
		}
		resp := MustSucceed(marshal.New(marshal.DefaultOptions()).Generate(req))
		ExpectContent(resp, "out/types/v1/codec.gen.go").
			ToContain("func (e Entry) EncodeOrc").
			ToNotContain("func (i Inner) EncodeOrc")
	})
})

var _ = ShouldNotLeakGoroutinesPerSpec()
