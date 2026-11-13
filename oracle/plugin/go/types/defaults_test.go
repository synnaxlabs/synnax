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
	"github.com/synnaxlabs/oracle/plugin"
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
					`v.NotEmptyString("name", r.Name)`,
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

		It("Should fill an array default of objects", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Tab struct { name string }

				Cfg struct { tabs Tab[] = [{ name = "a" }, { name = "b" }] }
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"if c.Tabs == nil {",
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

		It(
			"Should emit NonZero for a required uuid type",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Key = uuid

				Cfg struct {
					role Key {
						@validate required
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToContain(`v.NonZero("role", c.Role)`).
					ToNotContain(`NotEmptyString`)
			},
		)

		It("Should apply the rules of a field's alias type", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Decimal = float64 {
					@validate min 0
					@validate max 1
				}

				Cfg struct {
					size Decimal
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`v.GreaterThanEq("size", c.Size, 0)`,
				`v.LessThanEq("size", c.Size, 1)`,
			)
		})

		It(
			"Should apply the rules of each type in a distinct and alias chain",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Rate float64 {
					@validate min 0
				}

				Hz = Rate {
					@validate max 100
				}

				Cfg struct {
					rate Hz
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.GreaterThanEq("rate", c.Rate, 0)`,
					`v.LessThanEq("rate", c.Rate, 100)`,
				)
			},
		)

		It(
			"Should merge a field's rules with its type's rules",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Decimal = float64 {
					@validate min 0
					@validate max 1
				}

				Cfg struct {
					size Decimal {
						@validate gt 0
					}
					loose Decimal {
						@validate max 5
					}
					tight Decimal {
						@validate max 0.5
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.GreaterThan("size", c.Size, 0)`,
					`v.LessThanEq("size", c.Size, 1)`,
					`v.LessThanEq("loose", c.Loose, 1)`,
					`v.LessThanEq("tight", c.Tight, 0.5)`,
				).ToNotContain(
					`v.GreaterThanEq("size", c.Size, 0)`,
					`v.LessThanEq("loose", c.Loose, 5)`,
					`v.LessThanEq("tight", c.Tight, 1)`,
				)
			},
		)

		It(
			"Should keep a field's minimum length when its type is required",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Name = string {
					@validate required
				}

				Cfg struct {
					name Name {
						@validate min_length 3
					}
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToContain(`v.MinLength("name", c.Name, 3)`).
					ToNotContain(`v.NotEmptyString("name"`)
			},
		)

		It(
			"Should leave type rules on an inherited field to the embedded parent",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Rate float64 {
					@validate min 0
				}

				Base struct {
					rate Rate = 1
				}

				Cfg struct extends Base {
					rate Rate = 5
				}
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToContain(
						`v.GreaterThanEq("rate", b.Rate, 0)`,
						"v.Exec(c.Base.Validate)",
					).
					ToNotContain(`v.GreaterThanEq("rate", c.Rate, 0)`)
			},
		)

		It("Should require a field whose type is required", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Name = string {
					@validate required
				}

				Cfg struct {
					name     Name
					nickname Name?
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(`v.NotEmptyString("name", c.Name)`).
				ToNotContain(`v.NotEmptyString("nickname"`)
		})

		It("Should emit GreaterThan for a numeric gt", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					rate float64 {
						@validate gt 0.5
					}
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(`v.GreaterThan("rate", c.Rate, 0.5)`).
				ToNotContain(`GreaterThanEq`)
		})

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
					"if c.Level != nil {",
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
			"Should emit ApplyDefaults on the variant carrying a defaulted payload",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (l *LinearScale) ApplyDefaults() {",
					"l.LinearParams.ApplyDefaults()",
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
			"Should dispatch the wrapper ApplyDefaults on the active variant",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (u *Scale) ApplyDefaults() {",
					"switch variant := u.Variant.(type) {",
					"case LinearScale:",
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
					`v.Ternary("type", u.Variant == nil, "required")`,
					"case LinearScale:",
					"v.Exec(variant.Validate)",
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

	Describe("Union without a variant", func() {
		const source = `
			@go output "core/pkg/service/x"

			Window union on variant {
				tumbling {}
				sliding {}
			}

			Container struct {
				window   Window
				fallback Window?
				windows  Window[]
			}
		`

		It(
			"Should reject a union without a variant when no variant validates",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (u Window) Validate() error {",
					`v.Ternary("variant", u.Variant == nil, "required")`,
				)
				ExpectContent(resp, "types.gen.go").ToNotContain(
					"switch variant := u.Variant.(type) {",
				)
			},
		)

		It(
			"Should decode a missing discriminator to no variant",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"switch disc.Type {\n\tcase \"\":\n\t\tu.Variant = nil",
				)
			},
		)

		It(
			"Should validate required, optional, and array union fields",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c Container) Validate() error {",
					`validate.PathedError(c.Window.Validate(), "window")`,
					"if c.Fallback != nil {",
					`validate.PathedError(c.Fallback.Validate(), "fallback")`,
					`validate.PathedError(c.Windows[i].Validate(), "windows", `+
						`strconv.Itoa(i))`,
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
					rate float64 {
						@validate gt 0
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
					"func (s *StaticField) ApplyDefaults() {",
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
					`v.GreaterThan("rate", s.Rate, 0)`,
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
				content := MustContentOf(resp, "types.gen.go")
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *ConstValCJC) ApplyDefaults() {",
					"c.Val = 25",
					"i.Cjc.ApplyDefaults()",
				)
				// The fill must precede the recursion: a nil variant has no
				// ApplyDefaults to dispatch to.
				Expect(strings.Index(content, "i.Cjc.Variant = ConstValCJC{}")).
					To(BeNumerically("<", strings.Index(content, "i.Cjc.ApplyDefaults()")))
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
				"if i.Cjc.Variant == nil {",
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
				"if l.Range.Variant == nil {",
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
					"if p.Root.Variant == nil {",
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
					"func (c *Cfg) ApplyDefaults() {",
					"if c.Item.Cjc.Variant == nil {",
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

	It("Should guard a group's fill on every member being zero", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			MinMaxVal struct {
				min_val float64 = 0 { @default group "range" }
				max_val float64 = 1 { @default group "range" }
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			"func (m *MinMaxVal) ApplyDefaults() {",
			"if m.MinVal == 0 && m.MaxVal == 0 {",
			"m.MaxVal = 1",
		)
	})

	It("Should emit one guard per group", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			TwoPointLin struct {
				first_electrical  float64 = 0 { @default group "electrical" }
				second_electrical float64 = 1 { @default group "electrical" }
				first_physical    float64 = 0 { @default group "physical" }
				second_physical   float64 = 2 { @default group "physical" }
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			"if t.FirstElectrical == 0 && t.SecondElectrical == 0 {",
			"t.SecondElectrical = 1",
			"if t.FirstPhysical == 0 && t.SecondPhysical == 0 {",
			"t.SecondPhysical = 2",
		)
	})

	It(
		"Should leave an ungrouped field on the struct filling independently",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				min_val float64 = 0 { @default group "range" }
				max_val float64 = 1 { @default group "range" }
				label   string  = "signal"
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`if c.Label == "" {`,
				"if c.MinVal == 0 && c.MaxVal == 0 {",
			)
		},
	)

	It("Should group inline union variant fields", func(ctx SpecContext) {
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
					scaled_min     float64 = 0 { @default group "scaled" }
					scaled_max     float64 = 1 { @default group "scaled" }
					units          Units   = volts
				}
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			"func (m *MapScale) ApplyDefaults() {",
			"m.Units = UnitsVolts",
			"if m.PreScaledMin == 0 && m.PreScaledMax == 0 {",
			"m.PreScaledMax = 1",
			"if m.ScaledMin == 0 && m.ScaledMax == 0 {",
			"m.ScaledMax = 1",
		)
	})

	It(
		"Should omit a group whose members all default to zero",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				lower float64 = 0 { @default group "range" }
				upper float64 = 0 { @default group "range" }
			}
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("ApplyDefaults")
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

	DescribeTable(
		"Should reject an embedded type with defaults that a field holds",
		func(ctx SpecContext, field string) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
			Bases = Base[]
			BaseMap = map<string, Base>
			DistinctBases Base[]
			Page struct<T> { items T[] }
			Box struct<T> { item T }
			Name struct { value string }
			Pair struct<A, B> { a A b B }
			Outer struct { ` + field + ` }
		`
			table, diag := Analyze(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := &plugin.Request{Resolutions: table}
			Expect(goPlugin.Generate(req)).Error().To(MatchError(ContainSubstring(
				"Outer.items: Base has defaults and another struct embeds it, so it " +
					"cannot be held in a slice, a map, an optional field, or a " +
					"generic struct",
			)))
		},
		Entry("in a slice", "items Base[]"),
		Entry("in a map value", "items map<string, Base>"),
		Entry("in an optional field", "items Base?"),
		Entry("in a named slice", "items Bases"),
		Entry("in a named map", "items BaseMap"),
		Entry("in a distinct slice", "items DistinctBases"),
		Entry("in a slice of a generic struct", "items Page<Base>"),
		Entry("in a type parameter field of a generic struct", "items Box<Base>"),
		Entry("in a generic struct inside a slice", "items Box<Base>[]"),
		Entry(
			"in a second instantiation of a generic struct",
			"items Pair<Box<Name>, Box<Base>>",
		),
	)

	It(
		"Should reject an embedded type with defaults that a generic struct extends",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Page struct<T> extends Base { items T[] }
		`
			table, diag := Analyze(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := &plugin.Request{Resolutions: table}
			Expect(goPlugin.Generate(req)).Error().To(MatchError(ContainSubstring(
				"Page.Base: Base has defaults and another struct embeds it",
			)))
		},
	)

	It(
		"Should reject a generic struct that instantiates itself without end",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Box struct<T> { item T }
			Node struct<T> { kids Node<Box<T>>[] }
		`
			table, diag := Analyze(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := &plugin.Request{Resolutions: table}
			Expect(goPlugin.Generate(req)).Error().To(MatchError(ContainSubstring(
				"Node.kids: Node instantiates itself without end",
			)))
		},
	)

	It(
		"Should accept a generic struct that recurses into a fixed instantiation",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Wrapper struct { name string }
			Tree struct<T> {
				value T
				kids  Tree<Wrapper>[]
			}
			Outer struct { t Tree<Base> }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain("T Tree[Base]")
		},
	)

	It(
		"Should accept a generic struct nested in itself",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Box struct<T> { item T }
			Outer struct { items Box<Box<Base>> }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain("Items Box[Box[Base]]")
		},
	)

	It(
		"Should reject an embedded type with defaults in a required field of a "+
			"generic struct",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
			Page struct<T> { meta Base items T[] }
		`
			table, diag := Analyze(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := &plugin.Request{Resolutions: table}
			Expect(goPlugin.Generate(req)).Error().To(MatchError(ContainSubstring(
				"Page.meta: Base has defaults and another struct embeds it",
			)))
		},
	)

	DescribeTable(
		"Should reject an embedded type with defaults that an inline variant field "+
			"holds",
		func(ctx SpecContext, field string) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
			Shape union on type {
				box { ` + field + ` }
				dot {}
			}
		`
			table, diag := Analyze(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := &plugin.Request{Resolutions: table}
			Expect(goPlugin.Generate(req)).Error().To(MatchError(ContainSubstring(
				"BoxShape.items: Base has defaults and another struct embeds it, so " +
					"it cannot be held in a slice, a map, an optional field, or a " +
					"generic struct",
			)))
		},
		Entry("in a slice", "items Base[]"),
		Entry("in a map value", "items map<string, Base>"),
		Entry("in an optional field", "items Base?"),
	)

	It(
		"Should accept an embedded type without defaults that a field holds",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { name string }
			Cfg struct extends Base { mode string = "auto" }
			Outer struct { items Base[] }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain("Items []Base")
		},
	)

	It(
		"Should accept an embedded type with defaults in a required field",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
			Outer struct { item Base }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (o *Outer) SetDefaults() {\n\to.Item.SetDefaults()\n}")
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

		It("Should document the group rule on every grouped method", func(
			ctx SpecContext,
		) {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"// SetDefaults sets every field outside a default group to its "+
					"schema default. Each\n// default group takes its defaults only "+
					"when all of its members are zero.\nfunc (c *Cfg) SetDefaults() {",
				"// DefaultGroups sets the defaults of each default group whose "+
					"members are all zero, in\n// this type and in the types it "+
					"embeds or holds. A group with a non-zero member keeps\n// its "+
					"values.\nfunc (c *Cfg) DefaultGroups() {",
			)
		})

		It("Should document the group rule on a union of grouped variants", func(
			ctx SpecContext,
		) {
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
				"// ApplyDefaults fills the active variant's zero-valued fields with " +
					"their\n// schema-declared defaults. A default group takes its " +
					"defaults only when all of its\n// members are zero.\nfunc (u " +
					"*Scale) ApplyDefaults() {",
			)
		})

		It("Should document the group rule on a grouped ApplyDefaults", func(
			ctx SpecContext,
		) {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
				}
			`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"// ApplyDefaults fills zero-valued fields with their " +
					"schema-declared " +
					"defaults. A default\n// group takes its defaults only when all " +
					"of its members are zero.\nfunc (c *Cfg) ApplyDefaults() {",
			)
		})

		It(
			"Should set the groups of a decoded value only when it started zero",
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
					"// UnmarshalJSONFrom decodes the input over the value. A zero value "+
						"first takes its\n// defaults outside its default groups.",
					"// DecodeMsgpack decodes the input over the value, as "+
						"UnmarshalJSONFrom does.",
					"\tfresh := reflect.ValueOf(*c).IsZero()\n\tif fresh {\n"+
						"\t\tc.SetUngroupedDefaults()\n\t}\n"+
						"\tif err := json.UnmarshalDecode(dec, (*plain)(c)); err != nil {"+
						"\n\t\treturn err\n\t}\n"+
						"\tif fresh {\n\t\tc.DefaultGroups()\n\t}"+
						"\n\treturn nil\n}",
					"\tfresh := reflect.ValueOf(*c).IsZero()\n\tif fresh {\n"+
						"\t\tc.SetUngroupedDefaults()\n\t}\n"+
						"\tif err := dec.Decode((*plain)(c)); err != nil {"+
						"\n\t\treturn err\n\t}\n"+
						"\tif fresh {\n\t\tc.DefaultGroups()\n\t}"+
						"\n\treturn nil\n}",
				)
			},
		)

		It(
			"Should set a nested field's groups from the parent that prefilled it",
			func(ctx SpecContext) {
				source := `
				@go output "core/pkg/service/x"

				Range struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
					units string  = "volts"
				}

				Cfg struct { range Range }
			`
				resp := MustGenerate(ctx, source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *Cfg) SetUngroupedDefaults() {\n"+
						"\tc.Range.SetUngroupedDefaults()\n}",
					"func (c *Cfg) DefaultGroups() {\n\tc.Range.DefaultGroups()\n}",
					"func (r *Range) UnmarshalJSONFrom(",
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
		"Should set a restated literal's groups after the decode",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			MinMax struct {
				min_val float64 = 0 { @default group "range" }
				max_val float64 = 1 { @default group "range" }
				label   string  = "signal"
			}
			Parent struct { range MinMax = {} }
			Child struct extends Parent { range MinMax = { min_val = -10 } }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(
					"\t\tv0 := MinMax{}\n\t\tv0.SetUngroupedDefaults()\n"+
						"\t\tv0.MinVal = -10\n\t\tc.Range = v0\n",
					"func (c *Child) DefaultGroups() {\n\tc.Parent.DefaultGroups()\n"+
						"\tc.Range.DefaultGroups()\n}",
				).
				ToNotContain("v0.DefaultGroups()")
		},
	)

	It(
		"Should set a union literal's groups after its members",
		func(ctx SpecContext) {
			source := `
			@go output "core/pkg/service/x"

			Scale union on type {
				map {
					min_val float64 = 0 { @default group "range" }
					max_val float64 = 1 { @default group "range" }
				}
				none {}
			}
			Cfg struct { scale Scale = { type = "map", min_val = -10 } }
		`
			resp := MustGenerate(ctx, source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0 := MapScale{}\n\t\tv0.SetUngroupedDefaults()\n" +
					"\t\tv0.MinVal = -10\n\t\tv0.DefaultGroups()\n",
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
