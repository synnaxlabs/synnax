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

var _ = Describe("SetDefaults and Validate generation", func() {
	var (
		loader   *MockFileLoader
		goPlugin *types.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		goPlugin = types.New(types.DefaultOptions())
	})

	It(
		"Should generate SetDefaults setting non-zero static defaults",
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
				"func (c *Cfg) SetDefaults() {",
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
		"Should not generate SetDefaults when every default equals the zero value",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				name  string = ""
				count int32 = 0
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
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
		"Should generate SetDefaults on a struct that extends a base",
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
				"func (b *Base) SetDefaults() {",
				"b.Rate = 0.2",
				"func (c *Cfg) SetDefaults() {",
				"c.Base.SetDefaults()",
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
				"func (c *Cfg) SetDefaults() {",
				"c.Rate = 0.2",
				`c.Mode = "auto"`,
			)
		},
	)

	It(
		"Should leave a channel list on an extends struct to the element decode",
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
			ExpectContent(resp, "types.gen.go").
				ToContain(
					"func (c *Cfg) SetDefaults() {\n\tc.Base.SetDefaults()\n}",
					"func (c *Channel) UnmarshalJSONFrom(",
				).
				ToNotContain("c.Channels")
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
					`v.NotEmptyString("name", r.Name)`,
				)
				content.ToNotContain("r.Parent.Validate()", "r.Item.Validate()")
			},
		)
	})

	Describe("Struct-literal field defaults", func() {
		It(
			"Should set a nested component from a struct default after recursing",
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
					"func (a *Axes) SetDefaults() {\n\ta.X1.SetDefaults()\n" +
						"\ta.X1.Key = AxisKeyX1\n}",
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
			ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
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
				"func (c *Cfg) SetDefaults() {",
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

		It("Should fill an array default of objects", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Tab struct { name string }

				Cfg struct { tabs Tab[] = [{ name = "a" }, { name = "b" }] }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0 := Tab{}",
				`v0.Name = "a"`,
				"v1 := Tab{}",
				`v1.Name = "b"`,
				"c.Tabs = []Tab{v0, v1}",
			)
		})

		It(
			"Should bind a generic element's type arguments in an array default",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Box struct<T extends int64> { value T? }

				Cfg struct { boxes Box<int64>[] = [{ value = 7 }] }
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"v0 := Box[int64]{}",
					"v0.Value = new(int64(7))",
					"c.Boxes = []Box[int64]{v0}",
				)
			},
		)

		It(
			"Should keep a defaulted type parameter at its default in an array default",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Box struct<T extends int64 = int32> { value T? }

				Cfg struct { boxes Box<int64>[] = [{ value = 7 }] }
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"Value *int32",
					"v0 := Box{}",
					"v0.Value = new(int32(7))",
				)
			},
		)

		It("Should ignore an empty array default", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					patterns string[] = []
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
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
				`c.Filter.Patterns = []string{"^a.*"}`,
			)
		})
	})

	Describe("Recursion into nested types", func() {
		It(
			"Should recurse SetDefaults into a nested struct field",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inner Inner }
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (o *Outer) SetDefaults() {",
					"o.Inner.SetDefaults()",
				)
			},
		)

		It(
			"Should emit SetDefaults on a container that only nests a defaulted type",
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
				).ToContain("func (o *Outer) SetDefaults() {")
			},
		)

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
				ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
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
				ExpectContent(resp, "types.gen.go").
					ToContain(
						"func (n *Node) SetDefaults() {\n\tn.Weight = 1\n}",
						"func (n *Node) UnmarshalJSONFrom(",
					).
					ToNotContain("n.Child.SetDefaults()")
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
				`v.NotEmptyString("name", c.Name)`,
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
					`v.GreaterThanEq("rack", c.Rack, 1)`,
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
					`v.NonZero("rack", c.Rack)`,
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
				`v.LessThanEq("level", c.Level, 17)`,
			)
		})

		It(
			"Should bound an optional number only when it is present",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					level int32? {
						@validate min 0
						@validate max 17
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.GreaterThanEq("level", *c.Level, 0)`,
					`v.LessThanEq("level", *c.Level, 17)`,
				)
			},
		)

		It(
			"Should not require an optional field to be present",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					name string? {
						@validate required
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToNotContain("NotEmptyString")
			},
		)

		It(
			"Should accept an empty string for a zero minimum length",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					name string? {
						@validate min_length 0
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToNotContain("Validate()")
			},
		)

		It(
			"Should check string lengths with the validator methods",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					name string {
						@validate {
							min_length 2
							max_length 64
						}
					}
					alias string? {
						@validate max_length 64
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.MinLength("name", c.Name, 2)`,
					`v.MaxLength("name", c.Name, 64)`,
					`v.MaxLength("alias", *c.Alias, 64)`,
				)
			},
		)

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

			LinearParams struct {
				slope    int32    = 1
				notation Notation = NotationStandard
			}
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none   NoneParams
			}

			Container struct { scale Scale }
		`

		It(
			"Should emit SetDefaults on the variant carrying a defaulted payload",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (l *LinearScale) SetDefaults() {",
					"l.LinearParams.SetDefaults()",
				)
			},
		)

		It(
			"Should emit Validate on the variant without a path segment for the promoted embed",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (l LinearScale) Validate() error {",
					"v.Exec(l.LinearParams.Validate)",
				)
			},
		)

		It(
			"Should give the union wrapper no SetDefaults",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToNotContain("func (u *Scale) SetDefaults()")
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
			"Should leave a union field to the union decode",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToNotContain("func (c *Container) SetDefaults()")
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
					"func (s *StaticField) SetDefaults() {",
					"s.Kind = KindNumber",
					"func (s StaticField) Validate() error {",
					"!s.Kind.IsValid()",
				)
			},
		)

		It(
			"Should emit constraint checks for inline variant fields",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.MinLength("name", s.Name, 2)`,
				)
			},
		)

		It(
			"Should dispatch the wrapper Validate the container recurses into",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToContain("func (u Field) Validate() error {").
					ToNotContain("func (c *Container) SetDefaults()")
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
					"func (i *Item) SetDefaults() {",
					"i.Cjc.Variant = ConstValCJC{}",
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
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *ConstValCJC) SetDefaults() {\n\tc.Val = 25\n}",
					"v0 := ConstValCJC{}\n\t\tv0.SetDefaults()\n\t\ti.Cjc.Variant = v0",
				)
			},
		)

		It("Should fill the nil variant from an object literal", func(ctx SpecContext) {
			literal := `
				@go output "core/pkg/service/x"

				CJC union on source {
					built_in {}
					const_val {
						val float64 = 0
					}
				}

				Item struct { cjc CJC = { source = "const_val", val = 5 } }
			`
			resp := MustGenerate(ctx, literal, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0 := ConstValCJC{}",
				"v0.Val = 5",
				"i.Cjc.Variant = v0",
			)
		})

		It("Should fill a union literal nested in a variant", func(ctx SpecContext) {
			nested := `
				@go output "core/pkg/service/x"

				Window union on variant {
					dynamic {
						span int64 = 0
					}
					static {
						start int64 = 0
					}
				}

				Ref union on variant {
					key {
						key string
					}
					window {
						key    string
						window Window
					}
				}

				Log struct {
					range Ref = {
						variant = "window",
						key = "r7",
						window = { variant = "dynamic", span = 7 }
					}
				}
			`
			resp := MustGenerate(ctx, nested, "x", loader, goPlugin)
			content := MustContentOf(resp, "types.gen.go")
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0 := WindowRef{}",
				`v0.Key = "r7"`,
				"v1 := DynamicWindow{}",
				"v1.Span = 7",
				"v0.Window.Variant = v1",
				"l.Range.Variant = v0",
			)
			Expect(strings.Index(content, "v0.Window.Variant = v1")).
				To(BeNumerically("<", strings.Index(content, "l.Range.Variant = v0")))
		})

		It(
			"Should allocate optional fields a union literal sets",
			func(ctx SpecContext) {
				optional := `
				@go output "core/pkg/service/x"

				Meta struct { name string }

				Ref union on variant {
					key {
						key string
					}
					window {
						label string?
						span  int64?
						meta  Meta?
					}
				}

				Log struct {
					range Ref = {
						variant = "window",
						label = "x",
						span = 7,
						meta = { name = "m" }
					}
				}
			`
				resp := MustGenerate(ctx, optional, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"v0 := WindowRef{}",
					`v0.Label = new(string("x"))`,
					"v0.Span = new(int64(7))",
					"v0.Meta = &Meta{}",
					`v0.Meta.Name = "m"`,
					"l.Range.Variant = v0",
				)
			},
		)

		It(
			"Should build object array elements of a union literal",
			func(ctx SpecContext) {
				objects := `
				@go output "core/pkg/service/x"

				Tab struct { name string }

				Node union on variant {
					leaf {
						tabs Tab[]
					}
					split {
						size int64
					}
				}

				Panel struct {
					root Node = { variant = "leaf", tabs = [{ name = "first" }] }
				}
			`
				resp := MustGenerate(ctx, objects, "x", loader, goPlugin)
				content := MustContentOf(resp, "types.gen.go")
				ExpectContent(resp, "types.gen.go").ToContain(
					"v0 := LeafNode{}",
					"v1 := Tab{}",
					`v1.Name = "first"`,
					"v0.Tabs = []Tab{v1}",
					"p.Root.Variant = v0",
				)
				Expect(strings.Index(content, `v1.Name = "first"`)).
					To(BeNumerically("<", strings.Index(content, "v0.Tabs = []Tab{v1}")))
			},
		)

		It(
			"Should fill a union literal inside a struct default",
			func(ctx SpecContext) {
				inStruct := `
				@go output "core/pkg/service/x"

				CJC union on source {
					built_in {}
					const_val {
						val float64 = 0
					}
				}

				Item struct { cjc CJC }

				Cfg struct { item Item = { cjc = { source = "const_val", val = 5 } } }
			`
				resp := MustGenerate(ctx, inStruct, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *Cfg) SetDefaults() {",
					"v0 := ConstValCJC{}",
					"v0.Val = 5",
					"c.Item.Cjc.Variant = v0",
				)
			},
		)
	})
})

var _ = Describe("Default group generation", func() {
	var (
		loader   *MockFileLoader
		goPlugin *types.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		goPlugin = types.New(types.DefaultOptions())
	})

	It("Should set grouped inline union variant fields", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Units enum {
				volts = "Volts"
				amps  = "Amps"
			}

			Scale union on type {
				map {
					pre_scaled_min float64 = 0 { @default group "pre_scaled" }
					pre_scaled_max float64 = 1 { @default group "pre_scaled" }
					units          Units   = volts
				}
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			"func (m *MapScale) SetUngroupedDefaults() {\n\tm.Units = UnitsVolts\n}",
			"func (m *MapScale) DefaultGroups() {\n"+
				"\tif m.PreScaledMin == 0 && m.PreScaledMax == 0 {\n"+
				"\t\tm.PreScaledMax = 1\n\t}\n}",
		)
	})

	It(
		"Should emit no SetDefaults for a group whose members all default to zero",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				lower float64 = 0 { @default group "range" }
				upper float64 = 0 { @default group "range" }
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
		},
	)
})

var _ = Describe("Decode-time default generation", func() {
	var (
		loader   *MockFileLoader
		goPlugin *types.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		goPlugin = types.New(types.DefaultOptions())
	})

	It(
		"Should set every default without a zero guard and decode over the defaults",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				rolling int32 = 1
				name    string = "untitled"
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`"encoding/json/jsontext"`,
				`vmsgpack "github.com/vmihailenco/msgpack/v5"`,
				"func (c *Cfg) SetDefaults() {\n\tc.Rolling = 1\n"+
					"\tc.Name = \"untitled\"\n}",
				"func (c *Cfg) UnmarshalJSONFrom(dec *jsontext.Decoder) error {\n"+
					"\ttype plain Cfg\n"+
					"\tif reflect.ValueOf(*c).IsZero() {\n\t\tc.SetDefaults()\n\t}\n"+
					"\treturn json.UnmarshalDecode(dec, (*plain)(c))\n}",
				"func (c *Cfg) DecodeMsgpack(dec *vmsgpack.Decoder) error {\n"+
					"\ttype plain Cfg\n"+
					"\tif reflect.ValueOf(*c).IsZero() {\n\t\tc.SetDefaults()\n\t}\n"+
					"\treturn dec.Decode((*plain)(c))\n}",
			)
		},
	)

	It(
		"Should set only the non-zero leaves of a struct default",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Bounds struct {
				lower float64
				upper float64
			}

			Cfg struct {
				bounds Bounds = { lower = 0, upper = 100 }
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (c *Cfg) SetDefaults() {\n\tc.Bounds.Upper = 100\n}").
				ToNotContain("func (b *Bounds) SetDefaults()", "c.Bounds.Lower =")
		},
	)

	It(
		"Should set the defaults of a required nested struct but not of an optional, "+
			"slice, or map one",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Inner struct { rolling int32 = 1 }
			Outer struct {
				inner  Inner
				maybe  Inner?
				list   Inner[]
				byName map<string, Inner>
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(
					"func (o *Outer) SetDefaults() {\n\to.Inner.SetDefaults()\n}",
					"func (i *Inner) UnmarshalJSONFrom(",
					"func (o *Outer) UnmarshalJSONFrom(",
				).
				ToNotContain(
					"o.Maybe.SetDefaults()",
					"o.List[i].SetDefaults()",
					"value.SetDefaults()",
				)
		},
	)

	It(
		"Should emit no SetDefaults on a struct that reaches defaults only in a slice",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Inner struct { rolling int32 = 1 }
			Outer struct { list Inner[] }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (i *Inner) SetDefaults() {").
				ToNotContain(
					"func (o *Outer) SetDefaults()",
					"func (o *Outer) UnmarshalJSONFrom(",
				)
		},
	)

	It(
		"Should give an embedded parent no decode hooks and set its defaults from the "+
			"embedder",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(
					"func (b *Base) SetDefaults() {\n\tb.Rate = 0.2\n}",
					"func (c *Cfg) SetDefaults() {\n\tc.Base.SetDefaults()\n"+
						"\tc.Mode = \"auto\"\n}",
					"func (c *Cfg) UnmarshalJSONFrom(",
				).
				ToNotContain(
					"func (b *Base) UnmarshalJSONFrom(",
					"func (b *Base) DecodeMsgpack(",
				)
		},
	)

	It(
		"Should give decode hooks to a @create struct, whose New Go omits",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Doc struct {
				name string = "untitled"

				@create
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (d *Doc) UnmarshalJSONFrom(").
				ToNotContain("type New ")
		},
	)

	Describe("Default groups", func() {
		It(
			"Should set a group only when every member is zero",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
					name  string  = "untitled"
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *Cfg) SetDefaults() {\n\tc.SetUngroupedDefaults()\n"+
						"\tc.DefaultGroups()\n}",
					"func (c *Cfg) SetUngroupedDefaults() {\n\tc.Name = \"untitled\"\n}",
					"func (c *Cfg) DefaultGroups() {\n"+
						"\tif c.Lower == 0 && c.Upper == 0 {\n\t\tc.Upper = 1\n\t}\n}",
				)
			},
		)

		It(
			"Should decode over the ungrouped defaults, then set the zero groups",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"\t\tc.SetUngroupedDefaults()\n\t}\n"+
						"\tif err := json.UnmarshalDecode(dec, (*plain)(c)); err != nil {"+
						"\n\t\treturn err\n\t}\n\tc.DefaultGroups()\n\treturn nil\n}",
					"\t\tc.SetUngroupedDefaults()\n\t}\n"+
						"\tif err := dec.Decode((*plain)(c)); err != nil {"+
						"\n\t\treturn err\n\t}\n\tc.DefaultGroups()\n\treturn nil\n}",
				)
			},
		)

		It(
			"Should set a child's restated group before its embedded parent's",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				MinMax struct {
					min_val float64 = 0 { @default group "range" }
					max_val float64 = 1 { @default group "range" }
				}

				Cfg struct extends MinMax {
					min_val = 2 { @default group "range" }
					max_val = 100 { @default group "range" }
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *Cfg) SetUngroupedDefaults() {\n"+
						"\tc.MinMax.SetUngroupedDefaults()\n}",
					"func (c *Cfg) DefaultGroups() {\n"+
						"\tif c.MinVal == 0 && c.MaxVal == 0 {\n"+
						"\t\tc.MinVal = 2\n\t\tc.MaxVal = 100\n\t}\n"+
						"\tc.MinMax.DefaultGroups()\n}",
				)
			},
		)

		It(
			"Should decode a grouped union variant over its ungrouped defaults",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Scale union on type {
					map {
						pre_scaled_min float64 = 0 { @default group "pre_scaled" }
						pre_scaled_max float64 = 1 { @default group "pre_scaled" }
					}
					none {}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"\t\tv.MapScale.SetUngroupedDefaults()\n" +
						"\t\tif err := json.Unmarshal(data, &v, opts); err != nil {\n" +
						"\t\t\treturn err\n\t\t}\n\t\tv.MapScale.DefaultGroups()\n",
				)
			},
		)
	})

	It(
		"Should scope the setup of a union literal default in its own block",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Leaf struct {}
			Split struct { size int32 }
			Node union on variant {
				leaf  Leaf
				split Split
			}
			Panel struct { root Node = leaf }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (p *Panel) SetDefaults() {\n\tp.Root.Variant = LeafNode{}\n}",
			)
		},
	)

	It(
		"Should set the defaults of a defaulted union variant in its own block",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Leaf struct { depth int32 = 3 }
			Split struct { size int32 }
			Node union on variant {
				leaf  Leaf
				split Split
			}
			Panel struct { root Node = leaf }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (p *Panel) SetDefaults() {\n\t{\n\t\tv0 := LeafNode{}\n" +
					"\t\tv0.SetDefaults()\n\t\tp.Root.Variant = v0\n\t}\n}",
			)
		},
	)

	It(
		"Should write a zero default that overrides an inherited default",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Parent struct { rate float64 = 0.2 }
			Child struct extends Parent { rate float64 = 0 }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(
					"func (c *Child) SetDefaults() {\n\tc.Parent.SetDefaults()\n" +
						"\tc.Rate = 0\n}",
				).
				ToNotContain("c.Rate == 0")
		},
	)

	It(
		"Should build a struct default that overrides an inherited one in full",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			XY struct {
				x int32
				y int32
			}
			Parent struct { pos XY = { x = 5 } }
			Child struct extends Parent { pos XY = { x = 0, y = 3 } }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Child) SetDefaults() {\n\tc.Parent.SetDefaults()\n\t{\n" +
					"\t\tv0 := XY{}\n\t\tv0.X = 0\n\t\tv0.Y = 3\n\t\tc.Pos = v0\n\t}\n}",
			)
		},
	)

	It(
		"Should clear a list that an enclosing default filled",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			A struct { items int32[] }
			T struct { a A = { items = [1] } }
			Parent struct { t T = {} }
			Child struct extends Parent { t T = { a = { items = [] } } }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"\t\tv0 := T{}\n\t\tv0.SetDefaults()\n\t\tv0.A.Items = nil\n" +
					"\t\tc.T = v0\n",
			)
		},
	)

	It(
		"Should set the defaults of a required field typed by an alias",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Inner struct { rate int32 = 5 }
			Alias = Inner
			Outer struct { child Alias }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (o *Outer) SetDefaults() {\n\to.Child.SetDefaults()\n}")
		},
	)

	It(
		"Should set the defaults of a parent named through an alias",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate int32 = 10 }
			Alias = Base
			Child struct extends Alias { mode string = "auto" }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Child) SetDefaults() {\n\tc.Alias.SetDefaults()\n" +
					"\tc.Mode = \"auto\"\n}",
			)
		},
	)

	It(
		"Should write a zero leaf that overrides a nested type's own default",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Inner struct {
				rate int32 = 5
				tags string[] = ["a"]
			}
			Outer struct { inner Inner = { rate = 0, tags = [] } }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(
					"func (o *Outer) SetDefaults() {\n\to.Inner.SetDefaults()\n"+
						"\to.Inner.Rate = 0\n\to.Inner.Tags = nil\n}",
				).
				ToNotContain("o.Inner.Rate == 0", "o.Inner.Tags == nil")
		},
	)

	It(
		"Should set a union base's defaults on a defaulted variant",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { weight int32 = 2 }
			Leaf struct { name string? }
			Split struct { size int32 }
			Node union on variant extends Base {
				leaf  Leaf
				split Split
			}
			Panel struct { root Node = leaf }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0 := LeafNode{}\n\t\tv0.SetDefaults()\n",
			)
		},
	)

	It(
		"Should clear a variant's default list that a union literal sets empty",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Leaf struct { values int32[] = [1] }
			Split struct { size int32 }
			Node union on variant {
				leaf  Leaf
				split Split
			}
			Panel struct { root Node = { variant = "leaf", values = [] } }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0.SetDefaults()\n\t\tv0.Values = nil\n",
			)
		},
	)

	It(
		"Should give no decode hooks to a parent that another package embeds",
		func(ctx SpecContext) {
			loader.Add("schemas/task", `
			@go output "core/pkg/service/task"

			ReadConfig struct { sample_rate float64 = 10 }
		`)
			source := `
			import "schemas/task"

			@go output "core/pkg/service/ni"

			Cfg struct extends task.ReadConfig { mode string = "auto" }
		`
			resp := MustGenerate(ctx, source, "ni", loader, goPlugin)
			ExpectContent(resp, "task/types.gen.go").
				ToContain("func (r *ReadConfig) SetDefaults()").
				ToNotContain(
					"func (r *ReadConfig) UnmarshalJSONFrom(",
					"func (r *ReadConfig) DecodeMsgpack(",
				)
		},
	)

	It(
		"Should give no decode hooks to a type whose versioned copy is embedded",
		func(ctx SpecContext) {
			loader.Add("schemas/xv1", `
			@go output "core/pkg/service/x/versions/v1"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
		`)
			source := `
			import "schemas/xv1"

			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Doc struct { cfg xv1.Cfg }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "service/x/types.gen.go").
				ToContain(
					"func (b *Base) SetDefaults()",
					"func (d *Doc) UnmarshalJSONFrom(",
				).
				ToNotContain("func (b *Base) UnmarshalJSONFrom(")
		},
	)

	Describe("Union variants", func() {
		const source = `
			@go output "core/pkg/service/x"

			LinearParams struct { slope int32 = 1 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none   NoneParams
			}
		`

		It(
			"Should give a variant SetDefaults but no decode hooks",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToContain(
						"func (l *LinearScale) SetDefaults() {\n"+
							"\tl.LinearParams.SetDefaults()\n}",
						"func (l *LinearParams) SetDefaults() {\n\tl.Slope = 1\n}",
					).
					ToNotContain(
						"func (l *LinearScale) UnmarshalJSONFrom(",
						"func (l *LinearParams) UnmarshalJSONFrom(",
						"func (n *NoneScale) SetDefaults()",
					)
			},
		)

		It(
			"Should decode a variant over its defaults",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"\t\t\tLinearScale\n\t\t}\n\t\tv.LinearScale.SetDefaults()\n"+
						"\t\tif err := json.Unmarshal(data, &v, opts); err != nil {",
					"\t\t\tNoneScale\n\t\t}\n"+
						"\t\tif err := json.Unmarshal(data, &v, opts); err != nil {",
				)
			},
		)
	})
})
