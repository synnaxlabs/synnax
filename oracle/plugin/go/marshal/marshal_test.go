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
			It("Should generate Writer/Reader codec functions", func() {
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
						"w.String(s.Name)",
						"w.Int32(int32(s.Age))",
						"TestCodec xencoding.Codec",
						"func EncodeTest(w *orc.Writer",
						"func DecodeTest(r *orc.Reader",
						"defer writerPool.Put(w)",
						"defer readerPool.Put(r)",
						"return nil, err",
						"return w.Copy(), nil",
						"return DecodeTest(r, s)",
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
			It("Should delegate to nested struct codec functions", func() {
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
						"EncodeInner(w, &s.From)",
						"DecodeInner(r, &s.From)",
						"s.Name",
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
					ToContain("if s.Description != nil {")
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
					ToContain("TestCodec xencoding.Codec")
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
					ToContain("GraphCodec xencoding.Codec")
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
					ToContain("TestCodec xencoding.Codec")
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
					ToContain("TestCodec xencoding.Codec")
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
						"w.Bool(s.Items != nil)",
						"if s.Items != nil {",
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
						"w.Bool(s.Labels != nil)",
						"if s.Labels != nil {",
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
						"w.Bool(s.Data != nil)",
						"if s.Data != nil {",
						"present, err := r.Bool()",
						"if present {",
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
						"ContainerCodec xencoding.Codec",
						"EncodeType(w, &s.Type)",
						"DecodeType(r, &s.Type)",
					)
			})
		})
	})
})
