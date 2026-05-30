// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Edge Cases", func() {
	var (
		loader *MockFileLoader
		p      *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = types.New(types.DefaultOptions())
	})

	Describe("Ontology key zero values", func() {
		It("Should emit a string zero for uuid keys", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Item struct {
					key uuid @key
					@ontology type "item"
				}
			`
			resp := MustGenerate(ctx, source, "item", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring(`""`))
		})

		It("Should emit a numeric zero for int32 keys", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Item struct {
					key int32 @key
					@ontology type "item"
				}
			`
			resp := MustGenerate(ctx, source, "item", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring(`ontologyID(0)`))
		})

		It("Should emit a boolean zero for bool keys", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Item struct {
					key bool @key
					@ontology type "item"
				}
			`
			resp := MustGenerate(ctx, source, "item", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring(`ontologyID(false)`))
		})
	})

	Describe("Type parameter constraints", func() {
		It("Should produce a typeof enum fallback for type-param fields constrained on an enum", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Variant enum {
					primary  = "primary"
					secondary = "secondary"
				}

				Holder struct<V extends Variant> {
					value V
				}
			`
			resp := MustGenerate(ctx, source, "holder", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring("variantZ"))
		})

	})

	Describe("Auto-generated key defaults", func() {
		It("Should emit id.create() default for string key fields with @key generate", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Item struct {
					key string {
						@key generate
					}
					name string
				}
			`
			resp := MustGenerate(ctx, source, "item", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring("id.create()"))
		})

		It("Should emit uuid.create() default for uuid key fields with @key generate", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Item struct {
					key uuid {
						@key generate
					}
					name string
				}
			`
			resp := MustGenerate(ctx, source, "item", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring("uuid.create()"))
		})
	})
})
