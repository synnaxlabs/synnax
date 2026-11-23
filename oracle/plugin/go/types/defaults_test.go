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
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Cfg) SetDefaults() {",
				"c.Rolling = 1",
				"c.Scale = 1.5",
				`c.Name = "untitled"`,
				"c.Level = LevelH2",
			)
		},
	)

	It("Should generate Validate asserting enum membership", func() {
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
		resp := MustGenerate(source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			`"github.com/synnaxlabs/x/validate"`,
			"func (c Cfg) Validate() error {",
			`validate.New("Cfg")`,
			"!c.Level.IsValid()",
		)
	})

	It(
		"Should not generate SetDefaults when every default equals the zero value",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				name  string = ""
				count int32 = 0
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
		},
	)

	It(
		"Should key enum validation by the wire field name, not the Go name",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`v.Ternaryf("label_level"`,
				"!c.LabelLevel.IsValid()",
				"invalid label_level: %v",
			)
		},
	)

	It(
		"Should validate an optional enum only when it is present",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Cfg struct {
				level Level?
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c Cfg) Validate() error {",
				"if c.Level != nil {",
				`v.Ternaryf("level", !c.Level.IsValid(), "invalid level: %v", *c.Level)`,
			)
		},
	)

	It(
		"Should validate an optional enum of a union variant only when it is present",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Shape union on type {
				first { level Level? }
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToBeValidGoSource().ToContain(
				"func (f FirstShape) Validate() error {",
				"if f.Level != nil {",
				`v.Ternaryf("level", !f.Level.IsValid(), "invalid level: %v", *f.Level)`,
			)
		},
	)

	It(
		"Should validate each key of a map keyed by an enum",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Port enum {
				ain0 = "AIN0"
			}

			Ports = map<Port, uint32>

			Cfg struct {
				channels map<Port, uint32>
				aliased  Ports
			}

			Shape union on type {
				first { ports map<Port, uint32>? }
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToBeValidGoSource().ToContain(
				"func (c Cfg) Validate() error {",
				"for key := range c.Channels {",
				`v.Ternaryf("channels", !key.IsValid(), "invalid channels key: %v", key)`,
				"for key := range c.Aliased {",
				`v.Ternaryf("aliased", !key.IsValid(), "invalid aliased key: %v", key)`,
				"func (f FirstShape) Validate() error {",
				"for key := range f.Ports {",
				`v.Ternaryf("ports", !key.IsValid(), "invalid ports key: %v", key)`,
			)
		},
	)

	It(
		"Should widen the receiver when the type name would collide with the validator",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (vo Volume) Validate() error {",
				`v := validate.New("Volume")`,
				"!vo.Level.IsValid()",
			)
		},
	)

	It(
		"Should widen the receiver when the type name would collide with a loop index",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
			}

			Item struct {
				level Level
			}

			Index struct {
				items Item[]
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (in Index) Validate() error {",
				"in.Items[i].Validate()",
			)
		},
	)

	It(
		"Should generate SetDefaults on a struct that extends a base",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Base struct {
				rate float64 = 0.2
			}

			Cfg struct extends Base {
				mode string = "auto"
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (b Base) Validate() error {",
				"func (c Cfg) Validate() error {",
				"v.Exec(c.Base.Validate)",
			)
		},
	)

	It(
		"Should fill inherited defaults directly on a flattened extends struct",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Cfg) SetDefaults() {",
				"c.Rate = 0.2",
				`c.Mode = "auto"`,
			)
		},
	)

	It(
		"Should leave a channel list on an extends struct to the element decode",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (a *Axes) SetDefaults() {\n\ta.X1.SetDefaults()\n" +
						"\ta.X1.Key = AxisKeyX1\n}",
				)
			},
		)

		It("Should ignore an all-zero struct default", func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
		})
	})

	Describe("Array field defaults", func() {
		It("Should fill a non-empty array default", func() {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					patterns string[]  = ["^cRIO.*", "^nown.*"]
					limits   float64[] = [0, 1.5]
					flags    bool[]    = [false, true]
				}
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Cfg) SetDefaults() {",
				`c.Patterns = []string{"^cRIO.*", "^nown.*"}`,
				"c.Limits = []float64{0, 1.5}",
				"c.Flags = []bool{false, true}",
			)
		})

		It("Should fill an array of enum variants", func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"c.Levels = []Level{LevelH2, LevelH1}",
			)
		})

		It("Should fill an array default of objects", func() {
			source := `
				@go output "core/pkg/service/x"

				Tab struct { name string }

				Cfg struct { tabs Tab[] = [{ name = "a" }, { name = "b" }] }
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
				source := `
				@go output "core/pkg/service/x"

				Box struct<T extends int64> { value T? }

				Cfg struct { boxes Box<int64>[] = [{ value = 7 }] }
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"v0 := Box[int64]{}",
					"v0.Value = new(int64(7))",
					"c.Boxes = []Box[int64]{v0}",
				)
			},
		)

		It(
			"Should give a defaulted type parameter its default in an array default",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Box struct<T extends int64 = int32> { value T? }

				Cfg struct { boxes Box[] = [{ value = 7 }] }
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"Value *int32",
					"v0 := Box{}",
					"v0.Value = new(int32(7))",
				)
			},
		)

		It("Should ignore an empty array default", func() {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					patterns string[] = []
				}
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
		})

		It("Should fill an array component of a struct default", func() {
			source := `
				@go output "core/pkg/service/x"

				Filter struct {
					patterns string[]
				}

				Cfg struct {
					filter Filter = { patterns = ["^a.*"] }
				}
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`c.Filter.Patterns = []string{"^a.*"}`,
			)
		})
	})

	Describe("Recursion into nested types", func() {
		It(
			"Should recurse SetDefaults into a nested struct field",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct { inner Inner }
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (o *Outer) SetDefaults() {",
					"o.Inner.SetDefaults()",
				)
			},
		)

		It(
			"Should emit SetDefaults on a container that only nests a defaulted type",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Inner struct { rolling int32 = 1 }
				Outer struct {
					name  string
					inner Inner
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(
					resp,
					"types.gen.go",
				).ToContain("func (o *Outer) SetDefaults() {")
			},
		)

		It(
			"Should recurse Validate into a nested struct with a wire-name path segment",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Level enum { h1 = "h1" h2 = "h2" }
				Inner struct { level Level = LevelH2 }
				Outer struct { inner Inner }
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (o Outer) Validate() error {",
					`validate.PathedError(o.Inner.Validate(), "inner")`,
				)
			},
		)

		It(
			"Should index slice elements in a recursive Validate path",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Level enum { h1 = "h1" h2 = "h2" }
				Inner struct { level Level = LevelH2 }
				Outer struct { inners Inner[] }
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`"strconv"`,
					`validate.PathedError(o.Inners[i].Validate(), "inners", strconv.Itoa(i))`,
				)
			},
		)

		It(
			"Should not emit a method when a nested struct has no non-zero defaults",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Inner struct { name string = "" }
				Outer struct { inner Inner }
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToNotContain("SetDefaults")
			},
		)

		It(
			"Should terminate generation for a self-referential type",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Node struct {
					weight int32 = 1
					child  Node?
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
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
		It("Should emit NotEmptyString for a required string", func() {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					name string {
						@validate required
					}
				}
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c Cfg) Validate() error {",
				`v.NotEmptyString("name", c.Name)`,
			)
		})

		It(
			"Should classify a distinct numeric type by its primitive base",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Key uint32

				Cfg struct {
					rack Key {
						@validate min 1
					}
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.GreaterThanEq("rack", c.Rack, 1)`,
				)
			},
		)

		It(
			"Should emit NonZero for a required numeric (distinct) type",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Key uint32

				Cfg struct {
					rack Key {
						@validate required
					}
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.NonZero("rack", c.Rack)`,
				)
			},
		)

		It(
			"Should emit NonZero for a required uuid type",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Key = uuid

				Cfg struct {
					role Key {
						@validate required
					}
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToContain(`v.NonZero("role", c.Role)`).
					ToNotContain(`NotEmptyString`)
			},
		)

		It("Should apply the rules of a field's alias type", func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`v.GreaterThanEq("size", c.Size, 0)`,
				`v.LessThanEq("size", c.Size, 1)`,
			)
		})

		It(
			"Should apply the rules of each type in a distinct and alias chain",
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.GreaterThanEq("rate", c.Rate, 0)`,
					`v.LessThanEq("rate", c.Rate, 100)`,
				)
			},
		)

		It("Should refuse a non-finite value in a field with a finite rule", func() {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					value float64 { @validate finite }
					limit float64? { @validate finite }
				}
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`v.Finite("value", c.Value)`,
				"if c.Limit != nil {\n\tv.Finite(\"limit\", *c.Limit)",
			)
		})

		It(
			"Should merge a field's rules with its type's rules",
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToContain(`v.MinLength("name", c.Name, 3)`).
					ToNotContain(`v.NotEmptyString("name"`)
			},
		)

		It(
			"Should leave type rules on an inherited field to the embedded parent",
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToContain(
						`v.GreaterThanEq("rate", b.Rate, 0)`,
						"v.Exec(c.Base.Validate)",
					).
					ToNotContain(`v.GreaterThanEq("rate", c.Rate, 0)`)
			},
		)

		It("Should require a field whose type is required", func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(`v.NotEmptyString("name", c.Name)`).
				ToNotContain(`v.NotEmptyString("nickname"`)
		})

		It("Should emit GreaterThan for a numeric gt", func() {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					rate float64 {
						@validate gt 0.5
					}
				}
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain(`v.GreaterThan("rate", c.Rate, 0.5)`).
				ToNotContain(`GreaterThanEq`)
		})

		It("Should emit LessThanEq for a numeric max", func() {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					level int32 {
						@validate max 17
					}
				}
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				`v.LessThanEq("level", c.Level, 17)`,
			)
		})

		It(
			"Should bound an optional number only when it is present",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					level int32? {
						@validate min 0
						@validate max 17
					}
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.GreaterThanEq("level", *c.Level, 0)`,
					`v.LessThanEq("level", *c.Level, 17)`,
				)
			},
		)

		It(
			"Should not require an optional field to be present",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					name string? {
						@validate required
					}
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToNotContain("NotEmptyString")
			},
		)

		It(
			"Should accept an empty string for a zero minimum length",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					name string? {
						@validate min_length 0
					}
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToNotContain("Validate()")
			},
		)

		It(
			"Should check string lengths with the validator methods",
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.MinLength("name", c.Name, 2)`,
					`v.MaxLength("name", c.Name, 64)`,
					`v.MaxLength("alias", *c.Alias, 64)`,
				)
			},
		)

		It(
			"Should recurse into a nested type that only has a constraint",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Inner struct {
					name string {
						@validate required
					}
				}
				Outer struct { inner Inner }
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (l *LinearScale) SetDefaults() {",
					"l.LinearParams.SetDefaults()",
				)
			},
		)

		It(
			"Should emit Validate on the variant without a path segment for the promoted embed",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (l LinearScale) Validate() error {",
					"v.Exec(l.LinearParams.Validate)",
				)
			},
		)

		It(
			"Should give the union wrapper no SetDefaults",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToNotContain("func (u *Scale) SetDefaults()")
			},
		)

		It(
			"Should dispatch the wrapper Validate on the active variant",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (u Scale) Validate() error {",
					`v.Ternary("type", u.Variant == nil, "required")`,
					"case LinearScale:",
					"v.Exec(variant.Validate)",
				)
			},
		)

		It(
			"Should leave a union field to the union decode",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").
					ToNotContain("func (c *Container) SetDefaults()")
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

			Windows = Window[]

			Rate struct { hz int32 = 1 }

			Rates = Rate[]

			Holder struct { window Window }

			Shape union on kind {
				grid {
					cells Windows[]
					rates Rates[]
				}
				held Holder
			}

			Container struct {
				window    Window
				fallback  Window?
				windows   Window[]
				named     Windows
				grid      Windows[]
				by_name   map<string, Window>
				rates     Rates
				rate_grid Rates[]
			}
		`

		It(
			"Should reject a union without a variant when no variant validates",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (u Window) Validate() error {\n" +
						"\tv := validate.New(\"Window\")\n" +
						"\tv.Ternary(\"variant\", u.Variant == nil, \"required\")\n" +
						"\treturn v.Error()\n}",
				)
			},
		)

		It(
			"Should validate required, optional, array, and aliased array union fields",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c Container) Validate() error {",
					`validate.PathedError(c.Window.Validate(), "window")`,
					"if c.Fallback != nil {",
					`validate.PathedError(c.Fallback.Validate(), "fallback")`,
					`validate.PathedError(c.Windows[i].Validate(), "windows", `+
						`strconv.Itoa(i))`,
					`validate.PathedError(c.Named[i].Validate(), "named", `+
						`strconv.Itoa(i))`,
					"for j := range c.Grid[i] {",
					`validate.PathedError(c.Grid[i][j].Validate(), "grid", `+
						`strconv.Itoa(i), strconv.Itoa(j))`,
					`validate.PathedError(value.Validate(), "by_name", key)`,
				)
			},
		)

		It(
			"Should recurse nested arrays in an inline variant",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (g GridShape) Validate() error {",
					`validate.PathedError(g.Cells[i][j].Validate(), "cells", `+
						`strconv.Itoa(i), strconv.Itoa(j))`,
				)
			},
		)

		It(
			"Should validate a variant whose payload holds a union",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (h HeldShape) Validate() error {",
					"v.Exec(h.Holder.Validate)",
					"case HeldShape:",
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
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					`v.MinLength("name", s.Name, 2)`,
					`v.GreaterThan("rate", s.Rate, 0)`,
				)
			},
		)

		It(
			"Should dispatch the wrapper Validate the container recurses into",
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (i *Item) SetDefaults() {",
					"i.Cjc.Variant = ConstValCJC{}",
				)
			},
		)

		It(
			"Should recurse into the filled variant so its own defaults land",
			func() {
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
				resp := MustGenerate(withVariantDefault, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *ConstValCJC) SetDefaults() {\n\tc.Val = 25\n}",
					"v0 := ConstValCJC{}\n\t\tv0.SetDefaults()\n\t\ti.Cjc.Variant = v0",
				)
			},
		)

		It("Should fill the nil variant from an object literal", func() {
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
			resp := MustGenerate(literal, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0 := ConstValCJC{}",
				"v0.Val = 5",
				"i.Cjc.Variant = v0",
			)
		})

		It("Should fill a union literal nested in a variant", func() {
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
			resp := MustGenerate(nested, "x", loader, goPlugin)
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
			func() {
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
				resp := MustGenerate(optional, "x", loader, goPlugin)
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
			func() {
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
				resp := MustGenerate(objects, "x", loader, goPlugin)
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
			func() {
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
				resp := MustGenerate(inStruct, "x", loader, goPlugin)
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

	It(
		"Should guard each group apart and set ungrouped fields without a guard",
		func() {
			source := `
			@go output "core/pkg/service/x"

			TwoPointLin struct {
				first_electrical  float64 = 0 { @default group "electrical" }
				second_electrical float64 = 1 { @default group "electrical" }
				first_physical    float64 = 0 { @default group "physical" }
				second_physical   float64 = 2 { @default group "physical" }
				label             string  = "signal"
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (t *TwoPointLin) SetDefaults() {\n"+
					"\tt.SetUngroupedDefaults()\n\tt.DefaultGroups()\n}",
				"func (t *TwoPointLin) SetUngroupedDefaults() {\n"+
					"\tt.Label = \"signal\"\n}",
				"func (t *TwoPointLin) DefaultGroups() {\n"+
					"\tif t.FirstElectrical == 0 && t.SecondElectrical == 0 {\n"+
					"\t\tt.SecondElectrical = 1\n\t}\n"+
					"\tif t.FirstPhysical == 0 && t.SecondPhysical == 0 {\n"+
					"\t\tt.SecondPhysical = 2\n\t}\n}",
			)
		},
	)

	It("Should set grouped inline union variant fields", func() {
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
		resp := MustGenerate(source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			"func (m *MapScale) SetUngroupedDefaults() {\n\tm.Units = UnitsVolts\n}",
			"func (m *MapScale) DefaultGroups() {\n"+
				"\tif m.PreScaledMin == 0 && m.PreScaledMax == 0 {\n"+
				"\t\tm.PreScaledMax = 1\n\t}\n}",
		)
	})

	It(
		"Should emit no SetDefaults for a group whose members all default to zero",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				lower float64 = 0 { @default group "range" }
				upper float64 = 0 { @default group "range" }
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
			source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				rolling int32 = 1
				name    string = "untitled"
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (c *Cfg) SetDefaults() {\n\tc.Bounds.Upper = 100\n}").
				ToNotContain("func (b *Bounds) SetDefaults()", "c.Bounds.Lower =")
		},
	)

	It(
		"Should set the defaults of a required nested struct but not of an optional, "+
			"slice, or map one",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
			source := `
			@go output "core/pkg/service/x"

			Inner struct { rolling int32 = 1 }
			Outer struct { list Inner[] }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func(field string) {
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
			table, diag := Analyze(source, "x", loader)
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
		func() {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Page struct<T> extends Base { items T[] }
		`
			table, diag := Analyze(source, "x", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := &plugin.Request{Resolutions: table}
			Expect(goPlugin.Generate(req)).Error().To(MatchError(ContainSubstring(
				"Page.Base: Base has defaults and another struct embeds it",
			)))
		},
	)

	It(
		"Should reject a generic struct that instantiates itself without end",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Box struct<T> { item T }
			Node struct<T> { kids Node<Box<T>>[] }
		`
			table, diag := Analyze(source, "x", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := &plugin.Request{Resolutions: table}
			Expect(goPlugin.Generate(req)).Error().To(MatchError(ContainSubstring(
				"Node.kids: Node instantiates itself without end",
			)))
		},
	)

	It(
		"Should accept a generic struct that recurses into a fixed instantiation",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain("T Tree[Base]")
		},
	)

	It(
		"Should accept a generic struct nested in itself",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Box struct<T> { item T }
			Outer struct { items Box<Box<Base>> }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain("Items Box[Box[Base]]")
		},
	)

	It(
		"Should reject an embedded type with defaults in a required field of a "+
			"generic struct",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
			Page struct<T> { meta Base items T[] }
		`
			table, diag := Analyze(source, "x", loader)
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
		func(field string) {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
			Shape union on type {
				box { ` + field + ` }
				dot {}
			}
		`
			table, diag := Analyze(source, "x", loader)
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
		func() {
			source := `
			@go output "core/pkg/service/x"

			Base struct { name string }
			Cfg struct extends Base { mode string = "auto" }
			Outer struct { items Base[] }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain("Items []Base")
		},
	)

	It(
		"Should accept an embedded type with defaults in a required field",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate float64 = 0.2 }
			Cfg struct extends Base { mode string = "auto" }
			Outer struct { item Base }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (o *Outer) SetDefaults() {\n\to.Item.SetDefaults()\n}")
		},
	)

	It(
		"Should give decode hooks to a @create struct, whose New Go omits",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Doc struct {
				name string = "untitled"

				@create
			}
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (d *Doc) UnmarshalJSONFrom(").
				ToNotContain("type New ")
		},
	)

	Describe("Default groups", func() {
		It(
			"Should set a group only when every member is zero",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
					name  string  = "untitled"
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
				ExpectContent(resp, "types.gen.go").ToContain(
					"func (c *Cfg) SetDefaults() {\n\tc.SetUngroupedDefaults()\n"+
						"\tc.DefaultGroups()\n}",
					"func (c *Cfg) SetUngroupedDefaults() {\n\tc.Name = \"untitled\"\n}",
					"func (c *Cfg) DefaultGroups() {\n"+
						"\tif c.Lower == 0 && c.Upper == 0 {\n\t\tc.Upper = 1\n\t}\n}",
				)
			},
		)

		It("Should document the group rule on every grouped method", func() {
			source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
				}
			`
			resp := MustGenerate(source, "x", loader, goPlugin)
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

		It(
			"Should set the groups of a decoded value only when it started zero",
			func() {
				source := `
				@go output "core/pkg/service/x"

				Cfg struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
				}
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
				source := `
				@go output "core/pkg/service/x"

				Range struct {
					lower float64 = 0 { @default group "range" }
					upper float64 = 1 { @default group "range" }
					units string  = "volts"
				}

				Cfg struct { range Range }
			`
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
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
				resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (p *Panel) SetDefaults() {\n\tp.Root.Variant = LeafNode{}\n}",
			)
		},
	)

	It(
		"Should set the defaults of a defaulted union variant in its own block",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (p *Panel) SetDefaults() {\n\t{\n\t\tv0 := LeafNode{}\n" +
					"\t\tv0.SetDefaults()\n\t\tp.Root.Variant = v0\n\t}\n}",
			)
		},
	)

	It(
		"Should write a zero default that overrides an inherited default",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Parent struct { rate float64 = 0.2 }
			Child struct extends Parent { rate float64 = 0 }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
			source := `
			@go output "core/pkg/service/x"

			XY struct {
				x int32
				y int32
			}
			Parent struct { pos XY = { x = 5 } }
			Child struct extends Parent { pos XY = { x = 0, y = 3 } }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Child) SetDefaults() {\n\tc.Parent.SetDefaults()\n\t{\n" +
					"\t\tv0 := XY{}\n\t\tv0.X = 0\n\t\tv0.Y = 3\n\t\tc.Pos = v0\n\t}\n}",
			)
		},
	)

	It(
		"Should set a restated literal's groups after the decode",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0 := MapScale{}\n\t\tv0.SetUngroupedDefaults()\n" +
					"\t\tv0.MinVal = -10\n\t\tv0.DefaultGroups()\n",
			)
		},
	)

	It(
		"Should clear a list that an enclosing default filled",
		func() {
			source := `
			@go output "core/pkg/service/x"

			A struct { items int32[] }
			T struct { a A = { items = [1] } }
			Parent struct { t T = {} }
			Child struct extends Parent { t T = { a = { items = [] } } }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"\t\tv0 := T{}\n\t\tv0.SetDefaults()\n\t\tv0.A.Items = nil\n" +
					"\t\tc.T = v0\n",
			)
		},
	)

	It(
		"Should set the defaults of a required field typed by an alias",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Inner struct { rate int32 = 5 }
			Alias = Inner
			Outer struct { child Alias }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").
				ToContain("func (o *Outer) SetDefaults() {\n\to.Child.SetDefaults()\n}")
		},
	)

	It(
		"Should set the defaults of a parent named through an alias",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Base struct { rate int32 = 10 }
			Alias = Base
			Child struct extends Alias { mode string = "auto" }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"func (c *Child) SetDefaults() {\n\tc.Alias.SetDefaults()\n" +
					"\tc.Mode = \"auto\"\n}",
			)
		},
	)

	It(
		"Should write a zero leaf that overrides a nested type's own default",
		func() {
			source := `
			@go output "core/pkg/service/x"

			Inner struct {
				rate int32 = 5
				tags string[] = ["a"]
			}
			Outer struct { inner Inner = { rate = 0, tags = [] } }
		`
			resp := MustGenerate(source, "x", loader, goPlugin)
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
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0 := LeafNode{}\n\t\tv0.SetDefaults()\n",
			)
		},
	)

	It(
		"Should clear a variant's default list that a union literal sets empty",
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
			ExpectContent(resp, "types.gen.go").ToContain(
				"v0.SetDefaults()\n\t\tv0.Values = nil\n",
			)
		},
	)

	It(
		"Should give no decode hooks to a parent that another package embeds",
		func() {
			loader.Add("schemas/task", `
			@go output "core/pkg/service/task"

			ReadConfig struct { sample_rate float64 = 10 }
		`)
			source := `
			import "schemas/task"

			@go output "core/pkg/service/ni"

			Cfg struct extends task.ReadConfig { mode string = "auto" }
		`
			resp := MustGenerate(source, "ni", loader, goPlugin)
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
		func() {
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
			resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
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
			func() {
				resp := MustGenerate(source, "x", loader, goPlugin)
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
