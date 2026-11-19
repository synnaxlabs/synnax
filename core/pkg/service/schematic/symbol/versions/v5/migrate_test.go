// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	color "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/set"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	red   = color.Color{R: 255, A: 255}
	green = color.Color{G: 255, A: 255}
	black = color.Color{A: 255}
)

func region(key string, selectors ...string) v4.Region {
	return v4.Region{Key: key, Name: key, Selectors: selectors}
}

func stroked(r v4.Region, c color.Color) v4.Region {
	r.StrokeColor = &c
	return r
}

func filled(r v4.Region, c color.Color) v4.Region {
	r.FillColor = &c
	return r
}

func state(key string, regions ...v4.Region) v4.State {
	return v4.State{Key: key, Name: key, Regions: regions}
}

func colorPaint(c color.Color) *v5.Paint {
	return &v5.Paint{Variant: v5.ColorPaint{Value: servicecolor.Ref{
		Variant: servicecolor.FixedRef{Value: c},
	}}}
}

func equals(values ...float64) *v5.Condition {
	return &v5.Condition{
		Variant: v5.EqualsCondition{State: "active", Values: values},
	}
}

func svg(markup string) v5.Drawing {
	return v5.Drawing{Variant: v5.SvgDrawing{Markup: markup}}
}

var activeState = v5.State{Variant: v5.EnumerationState{
	BaseState: v5.BaseState{Key: "active", Name: "Active"},
	Match:     v5.Matcher{Variant: v5.ExactMatcher{}},
}}

var _ = Describe("MigrateSpec", func() {
	migrate := func(ctx SpecContext, svg string, states ...v4.State) v5.Spec {
		GinkgoHelper()
		return MustSucceed(v5.MigrateSpec(ctx, v4.Spec{
			SVG: svg, Variant: "static", Scale: 1, States: states,
		}))
	}

	It("Should turn each region into a part and each color into a look", func(
		ctx SpecContext,
	) {
		spec := migrate(ctx, `<svg><rect id="body"/></svg>`,
			state("off", stroked(region("body", "#body"), red)),
			state("on", stroked(region("body", "#body"), green)),
		)
		Expect(spec.Drawing).To(Equal(svg(`<svg><rect id="body"/></svg>`)))
		Expect(spec.Parts).To(Equal([]v5.Part{
			{Key: "body", Name: "body", Selectors: []string{"#body"}},
		}))
		Expect(spec.States).To(Equal([]v5.State{activeState}))
		Expect(spec.Looks).To(Equal([]v5.Look{
			{
				Key:    "look-1",
				Part:   new("body"),
				When:   equals(0),
				Stroke: colorPaint(red),
			},
			{
				Key:    "look-2",
				Part:   new("body"),
				When:   equals(1),
				Stroke: colorPaint(green),
			},
		}))
		Expect(spec.Commands).To(BeEmpty())
		Expect(spec.Interactions).To(BeEmpty())
	})

	It("Should give no state to a symbol with no states", func(ctx SpecContext) {
		spec := migrate(ctx, "<svg/>")
		Expect(spec.States).To(BeEmpty())
		Expect(spec.Parts).To(BeEmpty())
		Expect(spec.Looks).To(BeEmpty())
	})

	It("Should keep each selector that names shapes", func(ctx SpecContext) {
		spec := migrate(ctx, `<svg><rect id="123" class="a b"/></svg>`,
			state("off", region("r", `#\31 23`, ".a.b", `[data-region-id="x"]`)),
		)
		Expect(spec.Parts[0].Selectors).
			To(Equal([]string{`#\31 23`, ".a.b", `[data-region-id="x"]`}))
	})

	It("Should name each shape that a selector picks by position", func(
		ctx SpecContext,
	) {
		spec := migrate(
			ctx,
			`<svg><g><path/><circle/><path d="M0"/></g><path/></svg>`,
			state("off", region("r", "g > path:nth-of-type(2)", "path")),
		)
		Expect(spec.Parts[0].Selectors).To(Equal([]string{
			`[data-region-id="region-1"]`,
			`[data-region-id="region-2"]`,
			`[data-region-id="region-3"]`,
		}))
		Expect(spec.Drawing).To(Equal(svg(
			`<svg><g><path data-region-id="region-2"/>` +
				`<circle/><path data-region-id="region-1" d="M0"/></g>` +
				`<path data-region-id="region-3"/></svg>`,
		)))
	})

	It("Should skip a data-region-id that the drawing holds", func(ctx SpecContext) {
		spec := migrate(
			ctx,
			`<svg><rect data-region-id="region-1"/><rect/></svg>`,
			state("off", region("r", "rect:nth-of-type(2)")),
		)
		Expect(spec.Parts[0].Selectors).
			To(Equal([]string{`[data-region-id="region-2"]`}))
		Expect(spec.Drawing).To(Equal(svg(
			`<svg><rect data-region-id="region-1"/><rect data-region-id="region-2"/></svg>`,
		)))
	})

	It("Should drop a selector outside the editor's grammar", func(ctx SpecContext) {
		spec := migrate(ctx, `<svg><rect/><circle/></svg>`,
			state("off", region("r", "rect + circle")),
		)
		Expect(spec.Parts[0].Selectors).To(BeEmpty())
	})

	It("Should keep the markup of a drawing that does not parse", func(
		ctx SpecContext,
	) {
		spec := migrate(ctx, `<svg><g>`, state("off", region("r", "g")))
		Expect(spec.Drawing).To(Equal(svg(`<svg><g>`)))
		Expect(spec.Parts[0].Selectors).To(BeEmpty())
	})

	It("Should give a repeated region key with other selectors its own part", func(
		ctx SpecContext,
	) {
		spec := migrate(ctx, `<svg><rect id="a"/><rect id="b"/></svg>`,
			state("off", region("r", "#a")),
			state("on", region("r", "#b")),
		)
		Expect(spec.Parts).To(Equal([]v5.Part{
			{Key: "r", Name: "r", Selectors: []string{"#a"}},
			{Key: "r-2", Name: "r", Selectors: []string{"#b"}},
		}))
	})

	It("Should give an actuator the commands Off and On and a click between them",
		func(ctx SpecContext) {
			spec := MustSucceed(v5.MigrateSpec(ctx, v4.Spec{
				SVG: "<svg/>", Variant: "actuator", Scale: 1,
			}))
			Expect(spec.States).To(Equal([]v5.State{{Variant: v5.EnumerationState{
				BaseState: v5.BaseState{Key: "active", Name: "Active"},
				Match: v5.Matcher{Variant: v5.WithinMatcher{
					Bounds: spatial.Bounds{Lower: 0.9, Upper: 1.1},
				}},
			}}}))
			Expect(spec.Commands).To(Equal([]v5.Command{
				{
					Key:   "deactivate",
					Name:  "Off",
					State: new("active"),
					Value: v5.CommandValue{Variant: v5.FixedCommandValue{Value: 0}},
				},
				{
					Key:   "activate",
					Name:  "On",
					State: new("active"),
					Value: v5.CommandValue{Variant: v5.FixedCommandValue{Value: 1}},
				},
			}))
			Expect(spec.Interactions).To(Equal([]v5.Interaction{{
				Variant: v5.NextInteraction{
					BaseInteraction: v5.BaseInteraction{Key: "next"},
					Commands:        []string{"deactivate", "activate"},
				},
			}}))
		},
	)
})

var _ = Describe("MigrateSymbol", func() {
	It("Should keep the key, name, and group and migrate the spec", func(
		ctx SpecContext,
	) {
		data := v4.Spec{
			SVG:     `<svg><rect id="body"/></svg>`,
			Variant: "static",
			Scale:   1,
			States: []v4.State{
				state("off", stroked(region("body", "#body"), red)),
				state("on", stroked(region("body", "#body"), green)),
			},
		}
		old := v4.Symbol{
			Key:   uuid.New(),
			Name:  "Valve",
			Data:  data,
			Group: new(uuid.New()),
		}
		sym := MustSucceed(v5.MigrateSymbol(ctx, old))
		Expect(sym.Key).To(Equal(old.Key))
		Expect(sym.Name).To(Equal("Valve"))
		Expect(sym.Group).To(Equal(old.Group))
		Expect(sym.Data).To(Equal(MustSucceed(v5.MigrateSpec(ctx, data))))
		Expect(sym.Data.Parts).To(ConsistOf(HaveField("Key", "body")))
	})
})

var _ = Describe("Migration", func() {
	var (
		db      *gorp.DB
		enumSvc *enumeration.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		enumSvc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{
			DB: db,
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Symbol]{DB: db}))
	})
	create := func(ctx SpecContext, data v4.Spec) v4.Symbol {
		GinkgoHelper()
		data.Scale = 1
		sym := v4.Symbol{Key: uuid.New(), Name: "Valve", Data: data}
		Expect(gorp.NewCreate[v4.Key, v4.Symbol]().Entry(&sym).Exec(ctx, db)).
			To(Succeed())
		return sym
	}
	migrate := func(ctx SpecContext, actuators ...v4.Key) {
		GinkgoHelper()
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Symbol]{
			DB: db,
			Migrations: []gorp.Migration{v5.NewMigration(
				func(tx gorp.Tx) v5.Intern {
					return enumSvc.NewWriter(tx).NewInterner().Intern
				},
				func(context.Context, gorp.Tx) (set.Set[v5.Key], error) {
					return set.New(actuators...), nil
				},
			)},
		}))
	}
	retrieve := func(ctx SpecContext, key v4.Key) v5.Symbol {
		GinkgoHelper()
		var sym v5.Symbol
		Expect(gorp.NewRetrieve[v5.Key, v5.Symbol]().
			Where(gorp.MatchKeys[v5.Key, v5.Symbol](key)).
			Entry(&sym).Exec(ctx, db)).To(Succeed())
		return sym
	}
	enumerationOf := func(ctx SpecContext, sym v5.Symbol) enumeration.Enumeration {
		GinkgoHelper()
		st, ok := sym.Data.States[0].Variant.(v5.EnumerationState)
		Expect(ok).To(BeTrue())
		Expect(st.Enumeration).ToNot(BeNil())
		var e enumeration.Enumeration
		Expect(enumSvc.NewRetrieve().
			Where(enumeration.MatchKeys(*st.Enumeration)).
			Entry(&e).Exec(ctx, db)).To(Succeed())
		return e
	}
	fixed := func(c color.Color) *servicecolor.Ref {
		return &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: c}}
	}
	valve := func() v4.Spec {
		return v4.Spec{
			SVG: `<svg><rect id="b"/></svg>`, Variant: "static",
			States: []v4.State{
				state("off", stroked(filled(region("b", "#b"), red), black)),
				state("on", stroked(filled(region("b", "#b"), green), black)),
			},
		}
	}

	It("Should give the state an enumeration with an entry for each old state",
		func(ctx SpecContext) {
			old := create(ctx, valve())
			migrate(ctx)
			sym := retrieve(ctx, old.Key)
			e := enumerationOf(ctx, sym)
			Expect(e.Name).To(Equal("Valve"))
			Expect(e.Entries).To(Equal([]enumeration.Entry{
				{Key: "off", Value: 0, Name: "off", Color: fixed(red)},
				{Key: "on", Value: 1, Name: "on", Color: fixed(green)},
			}))
			Expect(sym.Data.Looks).To(Equal([]v5.Look{
				{
					Key:  "entry-b-fill",
					Part: new("b"),
					When: equals(0, 1),
					Fill: &v5.Paint{Variant: v5.EntryPaint{State: "active"}},
				},
				{
					Key:    "look-1",
					Part:   new("b"),
					When:   equals(0),
					Stroke: colorPaint(black),
				},
				{
					Key:    "look-2",
					Part:   new("b"),
					When:   equals(1),
					Stroke: colorPaint(black),
				},
			}))
		},
	)

	It("Should give entries with no color when no paint changes", func(
		ctx SpecContext,
	) {
		old := create(ctx, v4.Spec{SVG: "<svg/>", Variant: "static", States: []v4.State{
			state("off", stroked(region("b", "#b"), black)),
			state("", stroked(region("b", "#b"), black)),
			state("off"),
		}})
		migrate(ctx)
		sym := retrieve(ctx, old.Key)
		Expect(enumerationOf(ctx, sym).Entries).To(Equal([]enumeration.Entry{
			{Key: "off", Value: 0, Name: "off"},
			{Key: "1", Value: 1},
			{Key: "2", Value: 2, Name: "off"},
		}))
		Expect(sym.Data.Looks).To(HaveLen(2))
	})

	It("Should share one enumeration between symbols with equal states", func(
		ctx SpecContext,
	) {
		first, second := create(ctx, valve()), create(ctx, valve())
		other := create(ctx, v4.Spec{
			SVG: "<svg/>", Variant: "static",
			States: []v4.State{state("closed"), state("open")},
		})
		migrate(ctx)
		shared := enumerationOf(ctx, retrieve(ctx, first.Key)).Key
		Expect(enumerationOf(ctx, retrieve(ctx, second.Key)).Key).To(Equal(shared))
		Expect(enumerationOf(ctx, retrieve(ctx, other.Key)).Key).ToNot(Equal(shared))
	})

	It("Should give a symbol that an actuator node draws the commands of an actuator",
		func(ctx SpecContext) {
			drawn, plain := create(ctx, valve()), create(ctx, valve())
			migrate(ctx, drawn.Key)
			Expect(retrieve(ctx, drawn.Key).Data.Commands).To(HaveLen(2))
			Expect(retrieve(ctx, drawn.Key).Data.Interactions).To(HaveLen(1))
			Expect(retrieve(ctx, plain.Key).Data.Commands).To(BeEmpty())
		},
	)

	It("Should give a symbol with no states no enumeration", func(ctx SpecContext) {
		old := create(ctx, v4.Spec{SVG: "<svg/>", Variant: "static"})
		migrate(ctx)
		Expect(retrieve(ctx, old.Key).Data.States).To(BeEmpty())
	})
})
