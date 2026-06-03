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

var _ = Describe("Validation Rules", func() {
	var (
		loader *MockFileLoader
		p      *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = types.New(types.DefaultOptions())
	})

	It("Should emit regex with custom message when pattern provides one", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				slug string @validate {
					pattern "^[a-z]+$" "must be lowercase letters"
				}
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.regex(/^[a-z]+$/, "must be lowercase letters")`)
	})

	It("Should emit regex without message for pattern only", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				slug string @validate {
					pattern "^[a-z]+$"
				}
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.regex(/^[a-z]+$/)`)
	})

	It("Should emit min/max for number fields", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Range struct {
				low  int32 @validate { min 0 }
				high int32 @validate { max 100 }
			}
		`
		resp := MustGenerate(ctx, source, "range", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(".min(0)"))
		Expect(content).To(ContainSubstring(".max(100)"))
	})

	It("Should emit float min/max for float fields", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Range struct {
				low  float64 @validate { min 0.5 }
				high float64 @validate { max 99.5 }
			}
		`
		resp := MustGenerate(ctx, source, "range", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(".min(0.500000)"))
		Expect(content).To(ContainSubstring(".max(99.500000)"))
	})

	It("Should emit string default", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				name string = "untitled"
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.default("untitled")`)
	})

	It("Should emit supplementary-plane characters as surrogate pairs, not \\U", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				name string = "a😀b"
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring("\\uD83D\\uDE00"))
		Expect(content).ToNot(ContainSubstring("\\U"))
	})

	It("Should emit int default", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				count int32 = 5
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.default(5)`)
	})

	It("Should emit float default", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				ratio float64 = 1.5
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.default(1.500000)`)
	})

	It("Should emit bool default", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				active bool = true
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.default(true)`)
	})

	It("Should emit an object-literal struct default", func(ctx SpecContext) {
		source := `
			@ts output "out"

			TimestampConfig struct {
				format string
				tz     string
			}

			Item struct {
				timestamp TimestampConfig = { format "preciseDate", tz "local" }
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.default({ format: "preciseDate", tz: "local" })`)
	})

	It("Should emit id.create() default for string keys with create ident", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				key string = create
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.default(() => id.create())`)
	})

	It("Should rely on nullishToEmpty for an empty array default, not a misplaced element default", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				vals float64[] = []
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(`vals: array.nullishToEmpty(z.number()),`))
		Expect(content).ToNot(ContainSubstring(`z.number().default`))
	})

	It("Should apply a populated array default to the wrapped array, not the element", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				vals float64[] = [1.5, 2.5]
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(`vals: array.nullishToEmpty(z.number()).default([1.500000, 2.500000]),`))
		Expect(content).ToNot(ContainSubstring(`z.number().default`))
	})

	It("Should emit min/max length for string fields", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				name string @validate {
					min_length 1
					max_length 64
				}
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(".min(1)"))
		Expect(content).To(ContainSubstring(".max(64)"))
	})

	It("Should emit min(1, ...) for required string", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				name string @validate { required }
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.min(1, "Name is required")`)
	})

	It("Should leave fields without validation untouched", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				name string
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).ToNot(ContainSubstring(".min("))
		Expect(content).ToNot(ContainSubstring(".max("))
		Expect(content).ToNot(ContainSubstring(".regex("))
		Expect(content).ToNot(ContainSubstring(".default("))
	})
})
