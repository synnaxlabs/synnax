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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	It(
		"Should generate ApplyDefaults filling non-zero static defaults",
		func(ctx SpecContext) {
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
				"func (c *Cfg) ApplyDefaults() {",
				"c.Rolling = 1",
				"c.Scale = 1.5",
				`c.Name = "untitled"`,
				"c.Level = LevelH2",
			)
		},
	)

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

	It(
		"Should not generate ApplyDefaults when every default equals the zero value",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				name  string = ""
				count int32 = 0
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("ApplyDefaults")
		},
	)

	It(
		"Should key enum validation by the wire field name, not the Go name",
		func(ctx SpecContext) {
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
		},
	)

	It(
		"Should widen the receiver when the type name would collide with the validator",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Volume struct {
				level Level = LevelH2
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (vo Volume) Validate() error {",
				`v := validate.New("Volume")`,
				"!vo.Level.IsValid()",
			)
		},
	)

	It(
		"Should generate ApplyDefaults on a struct that extends a base",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct {
				rate float64 = 0.2
			}

			Cfg struct extends Base {
				mode string = "auto"
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (b *Base) ApplyDefaults() {",
				"b.Rate = 0.2",
				"func (c *Cfg) ApplyDefaults() {",
				"c.Base.ApplyDefaults()",
				`c.Mode = "auto"`,
			)
		},
	)

	It(
		"Should generate Validate on a struct that extends a base with an enum",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Base struct {
				level Level = LevelH2
			}

			Cfg struct extends Base {
				name string
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (b Base) Validate() error {",
				"func (c Cfg) Validate() error {",
				"v.Exec(c.Base.Validate)",
			)
		},
	)

	It(
		"Should fill inherited defaults directly on a flattened extends struct",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct {
				rate float64 = 0.2
				age  int32
			}

			Cfg struct extends Base {
				-age
				mode string = "auto"
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Cfg) ApplyDefaults() {",
				"c.Rate = 0.2",
				`c.Mode = "auto"`,
			)
		},
	)

	It(
		"Should recurse ApplyDefaults into a channel list on an extends struct",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct {
				rate float64 = 0.2
			}

			Channel struct {
				gain float64 = 1.5
			}

			Cfg struct extends Base {
				channels Channel[]
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Cfg) ApplyDefaults() {",
				"for i := range c.Channels {",
				"c.Channels[i].ApplyDefaults()",
			)
		},
	)

	Describe("@validate skip", func() {
		It(
			"Should exclude a reference field from Validate recursion",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Item struct {
					label string {
						@validate required
					}
				}

				Range struct {
					name string {
						@validate required
					}
					parent Range? {
						@validate skip
					}
					item Item {
						@validate skip
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				content := ExpectContent(resp, "types.gen.go")
				content.ToContain(
					"func (r Range) Validate() error {",
					`validate.NotEmptyString(v, "name", r.Name)`,
				)
				content.ToNotContain("r.Parent.Validate()", "r.Item.Validate()")
			},
		)
	})

	Describe("Struct-literal field defaults", func() {
		It(
			"Should fill a nested component from a struct default before recursing",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				AxisKey enum { x1 = "x1" }

				Axis struct {
					key  AxisKey
					tick float64 = 75
				}

				Axes struct {
					x1 Axis = { key = AxisKeyX1 }
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (a *Axes) ApplyDefaults() {",
					"if a.X1.Key == \"\" {",
					"a.X1.Key = AxisKeyX1",
					"a.X1.ApplyDefaults()",
				)
			},
		)

		It("Should ignore an all-zero struct default", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Bounds struct {
					lower float64
					upper float64
				}

				Cfg struct {
					bounds Bounds = { lower = 0, upper = 0 }
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("ApplyDefaults")
		})
	})

	Describe("Array field defaults", func() {
		It("Should fill a non-empty array default", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					patterns string[]  = ["^cRIO.*", "^nown.*"]
					limits   float64[] = [0, 1.5]
					flags    bool[]    = [false, true]
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Cfg) ApplyDefaults() {",
				"if c.Patterns == nil {",
				`c.Patterns = []string{"^cRIO.*", "^nown.*"}`,
				"c.Limits = []float64{0, 1.5}",
				"c.Flags = []bool{false, true}",
			)
		})

		It("Should fill an array of enum variants", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Level enum {
					h1 = "h1"
					h2 = "h2"
				}

				Cfg struct {
					levels Level[] = [LevelH2, LevelH1]
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"c.Levels = []Level{LevelH2, LevelH1}",
			)
		})

		It("Should ignore an empty array default", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					patterns string[] = []
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("ApplyDefaults")
		})

		It("Should fill an array component of a struct default", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Filter struct {
					patterns string[]
				}

				Cfg struct {
					filter Filter = { patterns = ["^a.*"] }
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"if c.Filter.Patterns == nil {",
				`c.Filter.Patterns = []string{"^a.*"}`,
			)
		})
	})

	Describe("Recursion into nested types", func() {
		It(
			"Should recurse ApplyDefaults into a nested struct field",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inner Inner }
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (o *Outer) ApplyDefaults() {",
					"o.Inner.ApplyDefaults()",
				)
			},
		)

		It(
			"Should emit ApplyDefaults on a container with no own default that nests a defaulted type",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct {
					name  string
					inner Inner
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(
					resp,
					"types.gen.go",
				).ToContain("func (o *Outer) ApplyDefaults() {")
			},
		)

		It("Should iterate a slice of structs in ApplyDefaults", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inners Inner[] }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"for i := range o.Inners {",
				"o.Inners[i].ApplyDefaults()",
			)
		})

		It(
			"Should nil-guard an optional struct field in ApplyDefaults",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inner Inner? }
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"if o.Inner != nil {",
					"o.Inner.ApplyDefaults()",
				)
			},
		)

		It("Should iterate map values in ApplyDefaults", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inners map<string, Inner> }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"for key, value := range o.Inners {",
				"value.ApplyDefaults()",
				"o.Inners[key] = value",
			)
		})

		It(
			"Should recurse Validate into a nested struct with a wire-name path segment",
			func(ctx SpecContext) {
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
			},
		)

		It(
			"Should index slice elements in a recursive Validate path",
			func(ctx SpecContext) {
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
			},
		)

		It(
			"Should not emit a method when a nested struct has no non-zero defaults",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Inner struct { name string = "" }
				Outer struct { inner Inner }
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToNotContain("ApplyDefaults")
			},
		)

		It(
			"Should terminate generation for a self-referential type",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Node struct {
					weight int32 = 1
					child  Node?
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (n *Node) ApplyDefaults() {",
					"if n.Child != nil {",
					"n.Child.ApplyDefaults()",
				)
			},
		)
	})

	Describe("@validate constraints", func() {
		It("Should emit NotEmptyString for a required string", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					name string {
						@validate required
					}
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c Cfg) Validate() error {",
				`validate.NotEmptyString(v, "name", c.Name)`,
			)
		})

		It(
			"Should classify a distinct numeric type by its primitive base",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Key uint32

				Cfg struct {
					rack Key {
						@validate min 1
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`validate.GreaterThanEq(v, "rack", c.Rack, 1)`,
				)
			},
		)

		It(
			"Should emit NonZero for a required numeric (distinct) type",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Key uint32

				Cfg struct {
					rack Key {
						@validate required
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`validate.NonZero(v, "rack", c.Rack)`,
				)
			},
		)

		It("Should emit LessThanEq for a numeric max", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					level int32 {
						@validate max 17
					}
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`validate.LessThanEq(v, "level", c.Level, 17)`,
			)
		})

		It(
			"Should recurse into a nested type that only has a constraint",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Inner struct {
					name string {
						@validate required
					}
				}
				Outer struct { inner Inner }
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (o Outer) Validate() error {",
					`validate.PathedError(o.Inner.Validate(), "inner")`,
				)
			},
		)
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

		It(
			"Should emit ApplyDefaults on the variant carrying a defaulted payload",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (s *ScaleLinear) ApplyDefaults() {",
					"s.LinearScale.ApplyDefaults()",
				)
			},
		)

		It(
			"Should emit Validate on the variant without a path segment for the promoted embed",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (s ScaleLinear) Validate() error {",
					"v.Exec(s.LinearScale.Validate)",
				)
			},
		)

		It(
			"Should dispatch the wrapper ApplyDefaults on the active variant",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (u *Scale) ApplyDefaults() {",
					"switch variant := u.Variant.(type) {",
					"case ScaleLinear:",
					"variant.ApplyDefaults()",
					"u.Variant = variant",
				)
			},
		)

		It(
			"Should dispatch the wrapper Validate on the active variant",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (u Scale) Validate() error {",
					"return variant.Validate()",
				)
			},
		)

		It(
			"Should recurse a struct field whose type is a method-bearing union",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *Container) ApplyDefaults() {",
					"c.Scale.ApplyDefaults()",
				)
			},
		)
	})

	Describe("Inline union variants with own defaults", func() {
		const source = `
			@go output "core/pkg/service/x"

			Kind enum { number = "number" text = "text" }

			Field union on type {
				static {
					kind Kind = KindNumber
					name string {
						@validate min_length 2
					}
				}
				generated {}
			}

			Container struct { fields Field[] }
		`

		It(
			"Should fill and validate inline fields on the variant",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (f *FieldStatic) ApplyDefaults() {",
					"f.Kind = KindNumber",
					"func (f FieldStatic) Validate() error {",
					"!f.Kind.IsValid()",
				)
			},
		)

		It(
			"Should emit constraint checks for inline variant fields",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.Ternaryf("name", len(f.Name) < 2, ` +
						`"must be at least 2 characters long")`,
				)
			},
		)

		It(
			"Should dispatch the wrapper methods the container recurses into",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (u *Field) ApplyDefaults() {",
					"func (u Field) Validate() error {",
					"func (c *Container) ApplyDefaults() {",
					"c.Fields[i].ApplyDefaults()",
				)
			},
		)
	})

	Describe("Union-typed field defaults", func() {
		const source = `
			@go output "core/pkg/service/x"

			CJC union on source {
				built_in {}
				const_val {
					val float64 = 0
				}
			}

			Item struct { cjc CJC = const_val }
		`

		It(
			"Should fill the nil variant with the defaulted one",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (i *Item) ApplyDefaults() {",
					"if i.Cjc.Variant == nil {",
					"i.Cjc.Variant = CJCConstVal{}",
				)
			},
		)

		It(
			"Should recurse into the filled variant so its own defaults land",
			func(ctx SpecContext) {
				withVariantDefault := `
					@go output "core/pkg/service/x"

					CJC union on source {
						built_in {}
						const_val {
							val float64 = 25
						}
					}

					Item struct { cjc CJC = const_val }
				`
				resp := MustGenerate(ctx, withVariantDefault, "x", loader, goPlugin)
				content := MustContentOf(resp, "types.gen.go")
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *CJCConstVal) ApplyDefaults() {",
					"c.Val = 25",
					"i.Cjc.ApplyDefaults()",
				)
				// The fill must precede the recursion: a nil variant has no
				// ApplyDefaults to dispatch to.
				Expect(strings.Index(content, "i.Cjc.Variant = CJCConstVal{}")).
					To(BeNumerically("<", strings.Index(content, "i.Cjc.ApplyDefaults()")))
			},
		)
	})
})
