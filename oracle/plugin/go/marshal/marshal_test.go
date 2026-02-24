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
			It("Should generate protowire codec", func() {
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
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"package test",
						"binary.BigEndian.AppendUint32",
						"TestCodec gorp.Codec",
					)
			})
		})

		Context("nested struct (inlined fields)", func() {
			It("Should inline nested struct fields in wire format", func() {
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
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "codec.gen.go").
					ToContain(
						"s.From.Type",
						"s.From.Key",
						"s.Name",
					)
			})
		})

		Context("hard optional field", func() {
			It("Should generate presence varint for pointer-based optional", func() {
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
				Expect(resp.Files).To(HaveLen(1))

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
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "codec.gen.go").
					ToContain("TestCodec gorp.Codec")
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
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "codec.gen.go").
					ToContain("GraphCodec gorp.Codec")
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
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "codec.gen.go").
					ToContain("TestCodec gorp.Codec")
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
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "codec.gen.go").
					ToContain("TestCodec gorp.Codec")
			})
		})
		Context("recursive struct (self-referencing optional fields)", func() {
			It("Should handle recursive type via length-prefixed sub-message", func() {
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
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "codec.gen.go").
					ToContain("ContainerCodec gorp.Codec").
					ToContain("marshalType")
			})
		})
	})
})
