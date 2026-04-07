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
	"github.com/synnaxlabs/oracle/plugin/go/marshal"
	. "github.com/synnaxlabs/oracle/testutil"
)

func TestGoMarshal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Go Marshal Plugin Suite")
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

		Context("hard optional field", func() {
			It("Should generate presence flag for pointer-based optional", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name string
						description string??
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain("if t.Description != nil {")
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
						status MyStatus??
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
						wrapper MyWrapper??
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

		Context("marshal skip on a field", func() {
			It("Should exclude the field from encoding and decoding", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name string
						data record? {
							@go marshal skip
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

		Context("soft optional array field", func() {
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

		Context("soft optional map field", func() {
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

		Context("hard optional array field", func() {
			It("Should generate a single presence bit without a redundant inner nil check", func() {
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
						items Inner[]??
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"if t.Items != nil {",
					"w.Bool(true)",
					"w.Uint32(uint32(len((*t.Items))))",
				)
				content.ToNotContain(
					"w.Bool((*t.Items) != nil)",
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
						elem Type??
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
