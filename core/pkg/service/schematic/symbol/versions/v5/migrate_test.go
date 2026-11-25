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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/selector"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	color "github.com/synnaxlabs/x/color/versions/v2"
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

	DescribeTable("Should name the shapes of a name that the checker refuses",
		func(ctx SpecContext, markup, sel, named string) {
			spec := migrate(ctx, markup, state("off", region("r", sel)))
			Expect(spec.Parts[0].Selectors).
				To(Equal([]string{`[data-region-id="region-1"]`}))
			Expect(spec.Parts[0].Selectors).To(HaveEach(Satisfy(selector.IsName)))
			Expect(spec.Drawing).To(Equal(svg(named)))
		},
		Entry(
			"an id that starts with a digit",
			`<svg><rect id="1a"/></svg>`,
			"#1a",
			`<svg><rect data-region-id="region-1" id="1a"/></svg>`,
		),
		Entry(
			"an id with a comma",
			`<svg><rect id="a,b"/></svg>`,
			"#a,b",
			`<svg><rect data-region-id="region-1" id="a,b"/></svg>`,
		),
		Entry(
			"a class that starts with a digit",
			`<svg><rect class="1a"/></svg>`,
			".1a",
			`<svg><rect data-region-id="region-1" class="1a"/></svg>`,
		),
		Entry(
			"an id that starts with a hyphen and a digit",
			`<svg><rect id="-1"/></svg>`,
			"#-1",
			`<svg><rect data-region-id="region-1" id="-1"/></svg>`,
		),
	)

	It("Should drop a name that the checker refuses and that selects no shape", func(
		ctx SpecContext,
	) {
		spec := migrate(
			ctx,
			`<svg><rect id="a"/></svg>`,
			state("off", region("r", "#a,b")),
		)
		Expect(spec.Parts[0].Selectors).To(BeEmpty())
		Expect(spec.Drawing).To(Equal(svg(`<svg><rect id="a"/></svg>`)))
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
	migrate := func(ctx SpecContext) {
		GinkgoHelper()
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Symbol]{
			DB: db,
			Migrations: []gorp.Migration{v5.NewMigration(v5.MigrationConfig{
				Intern: func(tx gorp.Tx) v5.Intern {
					return enumSvc.NewWriter(tx).NewInterner().Intern
				},
			})},
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

	It("Should give an actuator with one state the entry On and its looks on value 1",
		func(ctx SpecContext) {
			old := create(ctx, v4.Spec{
				SVG: `<svg><rect id="b"/></svg>`, Variant: "actuator",
				States: []v4.State{state("off", filled(region("b", "#b"), red))},
			})
			migrate(ctx)
			sym := retrieve(ctx, old.Key)
			Expect(enumerationOf(ctx, sym).Entries).To(Equal([]enumeration.Entry{
				{Key: "off", Value: 0, Name: "off"},
				{Key: "on", Value: 1, Name: "On"},
			}))
			Expect(sym.Data.Looks).To(Equal([]v5.Look{{
				Key:  "look-1",
				Part: new("b"),
				When: equals(0, 1),
				Fill: colorPaint(red),
			}}))
		},
	)

	It("Should give a symbol with no states no enumeration", func(ctx SpecContext) {
		old := create(ctx, v4.Spec{SVG: "<svg/>", Variant: "static"})
		migrate(ctx)
		Expect(retrieve(ctx, old.Key).Data.States).To(BeEmpty())
	})
})

var _ = Describe("Actuate", func() {
	within := v5.Matcher{Variant: v5.WithinMatcher{
		Bounds: spatial.Bounds{Lower: 0.9, Upper: 1.1},
	}}
	It("Should give a spec with no states the state active", func() {
		spec := v5.Actuate(v5.Spec{})
		Expect(spec.States).To(Equal([]v5.State{{Variant: v5.EnumerationState{
			BaseState: v5.BaseState{Key: "active", Name: "Active"},
			Match:     within,
		}}}))
		Expect(spec.Commands).To(HaveLen(2))
		Expect(spec.Interactions).To(HaveLen(1))
	})

	It("Should keep the enumeration of the state active", func() {
		e := uuid.New()
		in := v5.Spec{States: []v5.State{{Variant: v5.EnumerationState{
			BaseState:   v5.BaseState{Key: "active", Name: "Active"},
			Enumeration: &e,
			Match:       v5.Matcher{Variant: v5.ExactMatcher{}},
		}}}}
		spec := v5.Actuate(in)
		Expect(spec.States).To(Equal([]v5.State{{Variant: v5.EnumerationState{
			BaseState:   v5.BaseState{Key: "active", Name: "Active"},
			Enumeration: &e,
			Match:       within,
		}}}))
		st, ok := in.States[0].Variant.(v5.EnumerationState)
		Expect(ok).To(BeTrue())
		Expect(st.Match).To(Equal(v5.Matcher{Variant: v5.ExactMatcher{}}))
	})

	It("Should make a spec an actuator that holds its looks on value 0 on value 1, "+
		"when no look holds on value 1", func() {
		spec := v5.Actuate(v5.Spec{Variant: "static", Looks: []v5.Look{
			{Key: "a", When: equals(0)},
			{Key: "b", When: equals(2)},
			{Key: "c"},
		}})
		Expect(spec.Variant).To(Equal("actuator"))
		Expect(spec.Looks).To(Equal([]v5.Look{
			{Key: "a", When: equals(0, 1)},
			{Key: "b", When: equals(2)},
			{Key: "c"},
		}))
	})

	It("Should keep the looks when a look holds on value 1", func() {
		looks := []v5.Look{{Key: "a", When: equals(0)}, {Key: "b", When: equals(1)}}
		Expect(v5.Actuate(v5.Spec{Looks: looks}).Looks).To(Equal(looks))
	})
})

var _ = Describe("ActuatorEntries", func() {
	It("Should add the entry On with a free key when no entry has value 1", func() {
		entries, added := v5.ActuatorEntries([]enumeration.Entry{
			{Key: "on", Value: 0, Name: "Base"},
		})
		Expect(added).To(BeTrue())
		Expect(entries).To(Equal([]enumeration.Entry{
			{Key: "on", Value: 0, Name: "Base"},
			{Key: "1", Value: 1, Name: "On"},
		}))
	})

	DescribeTable("Should add no entry", func(entries []enumeration.Entry) {
		next, added := v5.ActuatorEntries(entries)
		Expect(added).To(BeFalse())
		Expect(next).To(Equal(entries))
	},
		Entry("to no entries", []enumeration.Entry(nil)),
		Entry("to entries with value 1", []enumeration.Entry{{Key: "a", Value: 1}}),
	)
})
