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

		Context("struct with uuid field", func() {
			It("Should use assignment not declaration for r.Read", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						key uuid
						name string
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"w.Write(s.Key[:])",
						"if _, err = r.Read(s.Key[:]); err != nil",
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
					"if s.Items != nil {",
					"w.Bool(true)",
					"w.Uint32(uint32(len(s.Items)))",
				)
				content.ToNotContain(
					"w.Bool(s.Items != nil)",
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
					"if s.Labels != nil {",
					"w.Bool(true)",
					"w.Uint32(uint32(len(s.Labels)))",
				)
				content.ToNotContain(
					"w.Bool(s.Labels != nil)",
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

		Context("generic struct test generation", func() {
			It("Should generate tests for generic structs with concrete type substitution", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Status struct<Details?> {
						key     string
						message string
						details Details?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec_gen_test.go").
					ToContain(
						"Describe(\"Status\"",
						"Status[string]",
						"EncodeStatus",
						"DecodeStatus",
					)
			})

			It("Should generate tests for generic structs with defaulted and non-defaulted type params", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Variant enum {
						info    = "info"
						warning = "warning"
						error   = "error"
					}

					Inner struct {
						name string
						@go omit
					}

					Status struct<Details?, V extends Variant = Variant> {
						key         string
						variant     V
						message     string
						description string?
						time        int64
						details     Details?
						items       Inner[]?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec_gen_test.go").
					ToContain(
						"Describe(\"Status\"",
						"Status[string]",
						"EncodeStatus",
						"DecodeStatus",
						"BenchmarkEncodeDecodeStatus",
						"FuzzDecodeStatus",
					)
			})

			It("Should generate benchmark and fuzz tests for generic structs", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Container struct<T?> {
						name  string
						value T?
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				ExpectContent(resp, "codec_gen_test.go").
					ToContain(
						"BenchmarkEncodeDecodeContainer",
						"FuzzDecodeContainer",
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
					"if s.Items != nil {",
					"w.Bool(true)",
					"w.Uint32(uint32(len((*s.Items))))",
				)
				content.ToNotContain(
					"w.Bool((*s.Items) != nil)",
				)
			})
		})

		Context("hard optional map field", func() {
			It("Should generate a single presence bit without a redundant inner nil check", func() {
				source := `
					@go output "core/pkg/test"
					@go marshal
					@pb

					Test struct {
						name   string
						labels map<string, string>??
					}
				`
				resp := MustGenerate(ctx, source, "test", loader, marshalPlugin)
				content := ExpectContent(resp, "codec.gen.go")
				content.ToContain(
					"if s.Labels != nil {",
					"w.Bool(true)",
					"w.Uint32(uint32(len((*s.Labels))))",
				)
				content.ToNotContain(
					"w.Bool((*s.Labels) != nil)",
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
	})
})
