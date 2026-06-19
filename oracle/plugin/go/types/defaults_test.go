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
	"github.com/synnaxlabs/oracle/plugin/go/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("ApplyDefaults and Validate generation", func() {
	var (
		loader   *MockFileLoader
		goPlugin *types.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		goPlugin = types.New(types.DefaultOptions())
	})

	It("Should generate ApplyDefaults filling non-zero static defaults", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Cfg struct {
				rolling int32   = 1
				scale   float64 = 1.5
				name    string  = "untitled"
				level   Level   = LevelH2
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			"func (c Cfg) ApplyDefaults() Cfg {",
			"c.Rolling = 1",
			"c.Scale = 1.5",
			`c.Name = "untitled"`,
			"c.Level = LevelH2",
			"return c",
		)
	})

	It("Should generate Validate asserting enum membership", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Cfg struct {
				level Level = LevelH2
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			`"github.com/synnaxlabs/x/validate"`,
			"func (c Cfg) Validate() error {",
			`validate.New("Cfg")`,
			"!c.Level.IsValid()",
		)
	})

	It("Should not generate ApplyDefaults when every default equals the zero value", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				name  string = ""
				count int32 = 0
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToNotContain("ApplyDefaults")
	})

	It("Should key enum validation by the wire field name, not the Go name", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Cfg struct {
				label_level Level = LevelH2
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			`v.Ternaryf("label_level"`,
			"!c.LabelLevel.IsValid()",
			"invalid label_level: %v",
		)
	})

	Describe("Recursion into nested types", func() {
		It("Should recurse ApplyDefaults into a nested struct field", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inner Inner }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (o Outer) ApplyDefaults() Outer {",
				"o.Inner = o.Inner.ApplyDefaults()",
			)
		})

		It("Should emit ApplyDefaults on a container with no own default that nests a defaulted type", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct {
					name  string
					inner Inner
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain("func (o Outer) ApplyDefaults() Outer {")
		})

		It("Should iterate a slice of structs in ApplyDefaults", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inners Inner[] }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"for i := range o.Inners {",
				"o.Inners[i] = o.Inners[i].ApplyDefaults()",
			)
		})

		It("Should nil-guard an optional struct field in ApplyDefaults", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inner Inner? }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"if o.Inner != nil {",
				"applied := o.Inner.ApplyDefaults()",
				"o.Inner = &applied",
			)
		})

		It("Should iterate map values in ApplyDefaults", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inners map<string, Inner> }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"for key, value := range o.Inners {",
				"o.Inners[key] = value.ApplyDefaults()",
			)
		})

		It("Should recurse Validate into a nested struct with a wire-name path segment", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Level enum { h1 = "h1" h2 = "h2" }
				Inner struct { level Level = LevelH2 }
				Outer struct { inner Inner }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (o Outer) Validate() error {",
				`validate.PathedError(o.Inner.Validate(), "inner")`,
			)
		})

		It("Should index slice elements in a recursive Validate path", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Level enum { h1 = "h1" h2 = "h2" }
				Inner struct { level Level = LevelH2 }
				Outer struct { inners Inner[] }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`"strconv"`,
				`validate.PathedError(o.Inners[i].Validate(), "inners", strconv.Itoa(i))`,
			)
		})

		It("Should not emit a method when a nested struct has no non-zero defaults", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Inner struct { name string = "" }
				Outer struct { inner Inner }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("ApplyDefaults")
		})

		It("Should terminate generation for a self-referential type", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Node struct {
					weight int32 = 1
					child  Node?
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (n Node) ApplyDefaults() Node {",
				"if n.Child != nil {",
				"applied := n.Child.ApplyDefaults()",
			)
		})
	})

	Describe("Recursion into union variants", func() {
		const source = `
			@go output "core/pkg/service/x"

			Notation enum { standard = "standard" scientific = "scientific" }

			LinearScale struct {
				slope    int32    = 1
				notation Notation = NotationStandard
			}
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none   NoneScale
			}

			Container struct { scale Scale }
		`

		It("Should emit ApplyDefaults on the variant carrying a defaulted payload", func(ctx SpecContext) {
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (s ScaleLinear) ApplyDefaults() ScaleLinear {",
				"s.LinearScale = s.LinearScale.ApplyDefaults()",
			)
		})

		It("Should emit Validate on the variant without a path segment for the promoted embed", func(ctx SpecContext) {
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (s ScaleLinear) Validate() error {",
				"v.Exec(s.LinearScale.Validate)",
			)
		})

		It("Should dispatch the wrapper ApplyDefaults on the active variant", func(ctx SpecContext) {
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (u Scale) ApplyDefaults() Scale {",
				"switch variant := u.Variant.(type) {",
				"case ScaleLinear:",
				"u.Variant = variant.ApplyDefaults()",
			)
		})

		It("Should dispatch the wrapper Validate on the active variant", func(ctx SpecContext) {
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (u Scale) Validate() error {",
				"return variant.Validate()",
			)
		})

		It("Should recurse a struct field whose type is a method-bearing union", func(ctx SpecContext) {
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c Container) ApplyDefaults() Container {",
				"c.Scale = c.Scale.ApplyDefaults()",
			)
		})
	})
})
