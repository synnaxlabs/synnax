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
				active bool = false
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.default(false)`)
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

	It("Should emit uuid.create() default for uuid keys with create ident", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				key uuid = create
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`.default(() => uuid.create())`)
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

	It("Should emit a struct default as a typed object literal", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Level enum {
				low  = "low"
				high = "high"
			}

			Point struct {
				x     int32
				y     int32
				level Level
			}

			Item struct {
				p Point = { x = 1, y = 2, level = LevelHigh }
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(`p: pointZ.prefault({ x: 1, y: 2, level: "high" }),`))
	})

	It("Should emit nested struct and array values in a struct default", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Inner struct {
				tags string[]
			}

			Mid struct {
				inner Inner
			}

			Item struct {
				m Mid = { inner = { tags = ["a", "b"] } }
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(`m: midZ.prefault({ inner: { tags: ["a", "b"] } }),`))
	})

	It("Should emit .prefault() for a struct override in an extending input struct, not .partial()", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Title struct {
				level   int32
				visible bool
			}

			Plot struct {
				key   string
				title Title
			}

			NewPlot struct extends Plot {
				title Title = { level = 1, visible = false }
				@ts use_input
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(`titleZ.prefault({ level: 1, visible: false })`))
		Expect(content).ToNot(ContainSubstring(`.partial({ title: true })`))
	})

	It("Should resolve enum-extends variants and generics in a nested struct default via typeless override", func(ctx SpecContext) {
		source := `
			@ts output "out"

			XAxisKey enum {
				x1 = "x1"
				x2 = "x2"
			}

			YAxisKey enum {
				y1 = "y1"
			}

			AxisKey enum extends XAxisKey, YAxisKey {}

			Bounds struct<T extends numeric = float64> {
				lower T
				upper T
			}

			Axis struct {
				key          AxisKey
				bounds       Bounds
				tick_spacing float64
			}

			Axes struct {
				x1 Axis
				y1 Axis
			}

			Plot struct {
				axes Axes
			}

			NewPlot struct extends Plot {
				axes = {
					x1 = { key = AxisKeyX1, bounds = { lower = 0, upper = 0 }, tick_spacing = 75 },
					y1 = { key = AxisKeyY1, bounds = { lower = 0, upper = 0 }, tick_spacing = 75 }
				}
				@ts use_input
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring(`axesZ.prefault({ x1: { key: "x1", bounds: { lower: 0, upper: 0 }, tickSpacing: 75 }, y1: { key: "y1", bounds: { lower: 0, upper: 0 }, tickSpacing: 75 } })`))
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
			ToContain(`.min(1, "name is required")`)
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
