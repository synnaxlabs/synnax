// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14_test

import (
	"context"
	"slices"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	enumerationv0 "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v14 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v14"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("ImportCustomNodes", func() {
	var (
		db          *gorp.DB
		src         symbol.Symbol
		enum        enumerationv0.Enumeration
		retrievable func(resource.ID) bool
		calls       []interned
		created     []symbol.Symbol
	)
	// states returns the states off and on with the shape a in the colors of each pair.
	states := func(pairs ...[2]*xcolor.Color) []symbolv4.State {
		out := make([]symbolv4.State, len(pairs))
		for i, p := range pairs {
			out[i] = symbolv4.State{
				Key:  []string{"off", "on"}[i],
				Name: []string{"Off", "On"}[i],
				Regions: []symbolv4.Region{{
					Key:         "a",
					Name:        "a",
					Selectors:   []string{".a"},
					StrokeColor: p[0],
					FillColor:   p[1],
				}},
			}
		}
		return out
	}
	stored := states(
		[2]*xcolor.Color{shade(1), shade(1)},
		[2]*xcolor.Color{shade(2), shade(2)},
	)
	// nodePaint is a look that paints the shape b in the color of the node.
	nodePaint := symbol.Look{
		Key:    "node",
		Part:   new("b"),
		Stroke: &symbol.Paint{Variant: symbol.NodePaint{}},
	}

	BeforeEach(func(ctx SpecContext) {
		calls, created = nil, nil
		retrievable = func(resource.ID) bool { return true }
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		old := symbolv4.Symbol{
			Key:   uuid.New(),
			Name:  "Valve",
			Group: new(uuid.New()),
			Data: symbolv4.Spec{
				Variant: "actuator",
				Scale:   1,
				SVG:     `<svg><rect class="a"/><rect class="b"/></svg>`,
				States:  stored,
			},
		}
		Expect(
			gorp.NewCreate[symbolv4.Key, symbolv4.Symbol]().Entry(&old).Exec(ctx, db),
		).
			To(Succeed())
		enums := MustOpen(
			enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}),
		)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[symbol.Key, symbol.Symbol]{
			DB: db,
			Migrations: []gorp.Migration{symbol.NewMigration(symbol.MigrationConfig{
				Intern: func(tx gorp.Tx) symbol.Intern {
					return enums.NewWriter(tx).NewInterner().Intern
				},
			})},
		}))
		Expect(gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
			Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](old.Key)).
			Entry(&src).
			Exec(ctx, db)).To(Succeed())
		src.Data.Parts = append(src.Data.Parts, symbol.Part{
			Key:       "b",
			Name:      "b",
			Selectors: []string{".b"},
		})
		src.Data.Looks = append(src.Data.Looks, nodePaint)
		Expect(gorp.NewCreate[symbol.Key, symbol.Symbol]().Entry(&src).Exec(ctx, db)).
			To(Succeed())
		state, ok := src.Data.States[0].Variant.(symbol.EnumerationState)
		Expect(ok).To(BeTrue())
		Expect(gorp.NewRetrieve[enumerationv0.Key, enumerationv0.Enumeration]().
			Where(gorp.MatchKeys[enumerationv0.Key, enumerationv0.Enumeration](
				*state.Enumeration,
			)).
			Entry(&enum).
			Exec(ctx, db)).To(Succeed())
	})

	// importNodes imports a schematic named Panel with a node for each of cfgs, and
	// returns its nodes.
	importNodes := func(
		ctx SpecContext,
		cfgs ...v13.NodeConfigVariant,
	) []v14.SymbolNodeConfig {
		GinkgoHelper()
		old := v13.Schematic{Name: "Panel"}
		for i, cfg := range cfgs {
			old.Nodes = append(old.Nodes, v13.Node{
				Key:    strconv.Itoa(i),
				Config: v13.NodeConfig{Variant: cfg},
			})
		}
		s := MustSucceed(v14.MigrateSchematic(ctx, old))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return v14.ImportCustomNodes(
				ctx,
				tx,
				&s,
				old,
				retrievable,
				func(
					_ context.Context,
					name string,
					entries []enumerationv0.Entry,
				) (enumerationv0.Key, error) {
					calls = append(calls, interned{name: name, entries: entries})
					return uuid.New(), nil
				},
				func(_ context.Context, sym *symbol.Symbol) error {
					sym.Key = uuid.New()
					created = append(created, *sym)
					return nil
				},
			)
		})).To(Succeed())
		nodes := make([]v14.SymbolNodeConfig, len(s.Nodes))
		for i, n := range s.Nodes {
			var ok bool
			nodes[i], ok = n.Config.Variant.(v14.SymbolNodeConfig)
			Expect(ok).To(BeTrue())
		}
		return nodes
	}
	// actuator returns a custom actuator, labeled Vent, that draws the symbol with key
	// with overrides.
	actuator := func(
		key symbol.Key,
		overrides []symbolv4.State,
	) v13.CustomActuatorNodeConfig {
		cfg := defaults.Fill(v13.CustomActuatorNodeConfig{SpecKey: key.String()})
		cfg.Label.Label, cfg.StateOverrides = "Vent", overrides
		return cfg
	}
	// run imports a schematic named Panel with the actuator of key and overrides, and
	// returns its node.
	run := func(
		ctx SpecContext,
		key symbol.Key,
		overrides []symbolv4.State,
	) v14.SymbolNodeConfig {
		GinkgoHelper()
		return importNodes(ctx, actuator(key, overrides))[0]
	}
	// stopActuating stores src with no commands or interactions.
	stopActuating := func(ctx SpecContext) {
		GinkgoHelper()
		src.Data.Commands, src.Data.Interactions = nil, nil
		Expect(gorp.NewCreate[symbol.Key, symbol.Symbol]().Entry(&src).Exec(ctx, db)).
			To(Succeed())
	}
	// newFill returns overrides with a fill that no entry carries.
	newFill := func() []symbolv4.State {
		return states(
			[2]*xcolor.Color{shade(3), shade(9)},
			[2]*xcolor.Color{shade(4), shade(9)},
		)
	}
	colors := func(entries []enumerationv0.Entry) []*servicecolor.Ref {
		out := make([]*servicecolor.Ref, len(entries))
		for i, e := range entries {
			out[i] = e.Color
		}
		return out
	}

	It("Should leave a node whose overrides equal its symbol on the symbol",
		func(ctx SpecContext) {
			n := run(ctx, src.Key, stored)
			Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: src.Key}))
			Expect(n.States[0].Enumeration).To(BeNil())
			Expect(calls).To(BeEmpty())
			Expect(created).To(BeEmpty())
		})

	It("Should give a node with new entry colors an enumeration in those colors",
		func(ctx SpecContext) {
			n := run(ctx, src.Key, states(
				[2]*xcolor.Color{shade(3), shade(3)},
				[2]*xcolor.Color{shade(4), shade(4)},
			))
			Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: src.Key}))
			Expect(n.States[0].Enumeration).ToNot(BeNil())
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].name).To(Equal("Valve: Vent"))
			Expect(
				colors(calls[0].entries),
			).To(Equal([]*servicecolor.Ref{fixed(3), fixed(4)}))
			Expect(created).To(BeEmpty())
		})

	It("Should give a node with a color that no entry carries a copy in the group of "+
		"its symbol, which keeps the paints of the node", func(ctx SpecContext) {
		n := run(ctx, src.Key, states(
			[2]*xcolor.Color{shade(3), shade(9)},
			[2]*xcolor.Color{shade(4), shade(9)},
		))
		Expect(created).To(HaveLen(1))
		c := created[0]
		Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: c.Key}))
		Expect(c.Name).To(Equal("Valve (Panel)"))
		Expect(c.Group).To(Equal(src.Group))
		Expect(c.Data.Looks).To(ContainElement(nodePaint))
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].name).To(Equal("Valve (Panel)"))
		state, ok := c.Data.States[0].Variant.(symbol.EnumerationState)
		Expect(ok).To(BeTrue())
		Expect(state.Enumeration).ToNot(BeNil())
	})

	It("Should keep each state of the symbol in a copy", func(ctx SpecContext) {
		level := symbol.State{Variant: symbol.NumberState{
			BaseState: symbol.BaseState{Key: "level", Name: "Level"},
		}}
		src.Data.States = append(src.Data.States, level)
		Expect(gorp.NewCreate[symbol.Key, symbol.Symbol]().Entry(&src).Exec(ctx, db)).
			To(Succeed())
		run(ctx, src.Key, states(
			[2]*xcolor.Color{shade(3), shade(9)},
			[2]*xcolor.Color{shade(4), shade(9)},
		))
		Expect(created).To(HaveLen(1))
		Expect(created[0].Data.States).To(HaveLen(2))
		Expect(created[0].Data.States[1]).To(Equal(level))
	})

	It("Should give no copy for a part that selects no shape", func(ctx SpecContext) {
		src.Data.Parts = append(src.Data.Parts, symbol.Part{
			Key:       "ghost",
			Name:      "ghost",
			Selectors: []string{},
		})
		src.Data.Looks = append([]symbol.Look{{
			Key:  "ghost",
			Part: new("ghost"),
			Fill: &symbol.Paint{Variant: symbol.EntryPaint{State: "active"}},
		}}, src.Data.Looks...)
		Expect(gorp.NewCreate[symbol.Key, symbol.Symbol]().Entry(&src).Exec(ctx, db)).
			To(Succeed())
		run(ctx, src.Key, states(
			[2]*xcolor.Color{shade(3), shade(3)},
			[2]*xcolor.Color{shade(4), shade(4)},
		))
		Expect(created).To(BeEmpty())
		Expect(calls).To(HaveLen(1))
		Expect(colors(calls[0].entries)).
			To(Equal([]*servicecolor.Ref{fixed(3), fixed(4)}))
	})

	It("Should draw override i on value i when the entries are out of order",
		func(ctx SpecContext) {
			slices.Reverse(enum.Entries)
			Expect(gorp.NewCreate[enumerationv0.Key, enumerationv0.Enumeration]().
				Entry(&enum).
				Exec(ctx, db)).To(Succeed())
			run(ctx, src.Key, states(
				[2]*xcolor.Color{shade(3), shade(3)},
				[2]*xcolor.Color{shade(4), shade(4)},
			))
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].entries).To(HaveLen(2))
			Expect(calls[0].entries[0].Value).To(Equal(1.0))
			Expect(colors(calls[0].entries)).
				To(Equal([]*servicecolor.Ref{fixed(4), fixed(3)}))
		})

	DescribeTable("Should leave a node on its symbol when its user cannot retrieve",
		func(ctx SpecContext, rejected func() resource.ID) {
			retrievable = func(id resource.ID) bool { return id != rejected() }
			n := run(ctx, src.Key, states(
				[2]*xcolor.Color{shade(3), shade(9)},
				[2]*xcolor.Color{shade(4), shade(9)},
			))
			Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: src.Key}))
			Expect(n.States[0].Enumeration).To(BeNil())
			Expect(calls).To(BeEmpty())
			Expect(created).To(BeEmpty())
		},
		Entry("the symbol", func() resource.ID { return src.ResourceID() }),
		Entry("the enumeration of the symbol", func() resource.ID {
			return enum.ResourceID()
		}),
	)

	It("Should leave a node on its symbol when its user cannot retrieve an "+
		"enumeration that its copy keeps", func(ctx SpecContext) {
		mode := enumerationv0.Enumeration{Key: uuid.New(), Name: "Mode"}
		Expect(gorp.NewCreate[enumerationv0.Key, enumerationv0.Enumeration]().
			Entry(&mode).
			Exec(ctx, db)).To(Succeed())
		src.Data.States = append(src.Data.States, symbol.State{
			Variant: symbol.EnumerationState{
				BaseState:   symbol.BaseState{Key: "mode", Name: "Mode"},
				Enumeration: &mode.Key,
				Match:       symbol.Matcher{Variant: symbol.ExactMatcher{}},
			},
		})
		Expect(gorp.NewCreate[symbol.Key, symbol.Symbol]().Entry(&src).Exec(ctx, db)).
			To(Succeed())
		retrievable = func(id resource.ID) bool { return id != mode.ResourceID() }
		n := run(ctx, src.Key, newFill())
		Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: src.Key}))
		Expect(calls).To(BeEmpty())
		Expect(created).To(BeEmpty())
	})

	It("Should leave a custom actuator with no overrides on a symbol that actuates",
		func(ctx SpecContext) {
			n := run(ctx, src.Key, nil)
			Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: src.Key}))
			Expect(created).To(BeEmpty())
		})

	It("Should give a custom actuator a copy that actuates when its symbol does not",
		func(ctx SpecContext) {
			stopActuating(ctx)
			n := run(ctx, src.Key, nil)
			Expect(created).To(HaveLen(1))
			c := created[0]
			Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: c.Key}))
			Expect(c.Name).To(Equal("Valve (Panel)"))
			Expect(c.Group).To(Equal(src.Group))
			Expect(c.Data).To(Equal(symbol.Actuate(src.Data)))
			Expect(calls).To(BeEmpty())
		})

	It("Should give a custom actuator with new entry colors a copy that actuates, "+
		"when its symbol does not", func(ctx SpecContext) {
		stopActuating(ctx)
		n := run(ctx, src.Key, states(
			[2]*xcolor.Color{shade(3), shade(3)},
			[2]*xcolor.Color{shade(4), shade(4)},
		))
		Expect(created).To(HaveLen(1))
		c := created[0]
		Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: c.Key}))
		Expect(c.Data.Commands).To(Equal(symbol.Actuate(src.Data).Commands))
		Expect(calls).To(HaveLen(1))
		Expect(colors(calls[0].entries)).
			To(Equal([]*servicecolor.Ref{fixed(3), fixed(4)}))
	})

	It("Should give a custom actuator on a symbol with one state a copy that draws "+
		"value 1 as value 0 and reads it as the entry On", func(ctx SpecContext) {
		e := enumerationv0.Enumeration{
			Key:     uuid.New(),
			Name:    "Tank",
			Entries: []enumerationv0.Entry{{Key: "base", Value: 0, Name: "Base"}},
		}
		Expect(gorp.NewCreate[enumerationv0.Key, enumerationv0.Enumeration]().
			Entry(&e).
			Exec(ctx, db)).To(Succeed())
		on := func(values ...float64) *symbol.Condition {
			return &symbol.Condition{
				Variant: symbol.EqualsCondition{State: "active", Values: values},
			}
		}
		tank := symbol.Symbol{Key: uuid.New(), Name: "Tank", Data: symbol.Spec{
			Drawing: symbol.Drawing{Variant: symbol.SvgDrawing{Markup: "<svg/>"}},
			States: []symbol.State{{Variant: symbol.EnumerationState{
				BaseState:   symbol.BaseState{Key: "active", Name: "Active"},
				Enumeration: &e.Key,
				Match:       symbol.Matcher{Variant: symbol.ExactMatcher{}},
			}}},
			Looks:   []symbol.Look{{Key: "look-1", When: on(0)}},
			Variant: "static",
			Scale:   1,
		}}
		Expect(gorp.NewCreate[symbol.Key, symbol.Symbol]().Entry(&tank).Exec(ctx, db)).
			To(Succeed())
		run(ctx, tank.Key, nil)
		Expect(created).To(HaveLen(1))
		Expect(created[0].Data.Looks).To(Equal([]symbol.Look{
			{Key: "look-1", When: on(0, 1)},
		}))
		Expect(calls).To(Equal([]interned{{
			name: "Tank (Panel)",
			entries: []enumerationv0.Entry{
				{Key: "base", Value: 0, Name: "Base"},
				{Key: "on", Value: 1, Name: "On"},
			},
		}}))
	})

	It("Should give a custom static node and a custom actuator with the same "+
		"overrides their own copies", func(ctx SpecContext) {
		stopActuating(ctx)
		static := defaults.Fill(v13.CustomStaticNodeConfig{SpecKey: src.Key.String()})
		static.StateOverrides = newFill()
		nodes := importNodes(ctx, static, actuator(src.Key, newFill()))
		Expect(created).To(HaveLen(2))
		Expect(nodes[0].Spec.Variant).To(Equal(symbol.CustomRef{Key: created[0].Key}))
		Expect(nodes[1].Spec.Variant).To(Equal(symbol.CustomRef{Key: created[1].Key}))
		Expect(created[0].Data.Commands).To(BeEmpty())
		Expect(created[1].Data.Commands).To(HaveLen(2))
	})

	It("Should leave a node whose symbol the Core lacks on that symbol",
		func(ctx SpecContext) {
			key := uuid.New()
			n := run(ctx, key, stored)
			Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: key}))
			Expect(calls).To(BeEmpty())
			Expect(created).To(BeEmpty())
		})
})
