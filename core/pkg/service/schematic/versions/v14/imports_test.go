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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
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

var _ = Describe("ImportOverrides", func() {
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
		spec, entries := MustSucceed2(symbol.MigrateStoredSpec(ctx, symbolv4.Spec{
			Variant: "actuator",
			Scale:   1,
			SVG:     `<svg><rect class="a"/><rect class="b"/></svg>`,
			States:  stored,
		}))
		spec.Parts = append(spec.Parts, symbol.Part{
			Key:       "b",
			Name:      "b",
			Selectors: []string{".b"},
		})
		spec.Looks = append(spec.Looks, nodePaint)
		enum = enumerationv0.Enumeration{
			Key:     uuid.New(),
			Name:    "Valve",
			Entries: entries,
		}
		state, ok := spec.States[0].Variant.(symbol.EnumerationState)
		Expect(ok).To(BeTrue())
		state.Enumeration = &enum.Key
		spec.States[0].Variant = state
		src = symbol.Symbol{
			Key:   uuid.New(),
			Name:  "Valve",
			Data:  spec,
			Group: new(uuid.New()),
		}
		Expect(gorp.NewCreate[enumerationv0.Key, enumerationv0.Enumeration]().
			Entry(&enum).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.NewCreate[symbol.Key, symbol.Symbol]().Entry(&src).Exec(ctx, db)).
			To(Succeed())
	})

	// run imports a schematic named Panel with one custom node, labeled Vent, that
	// draws the symbol with key with overrides, and returns that node.
	run := func(
		ctx SpecContext,
		key symbol.Key,
		overrides []symbolv4.State,
	) v14.SymbolNodeConfig {
		GinkgoHelper()
		cfg := defaults.Fill(v13.CustomActuatorNodeConfig{SpecKey: key.String()})
		cfg.Label.Label, cfg.StateOverrides = "Vent", overrides
		old := v13.Schematic{
			Name:  "Panel",
			Nodes: []v13.Node{{Key: "a", Config: v13.NodeConfig{Variant: cfg}}},
		}
		s := MustSucceed(v14.MigrateSchematic(ctx, old))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return v14.ImportOverrides(
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
		n, ok := s.Nodes[0].Config.Variant.(v14.SymbolNodeConfig)
		Expect(ok).To(BeTrue())
		return n
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

	It("Should leave a node whose symbol the Core lacks on that symbol",
		func(ctx SpecContext) {
			key := uuid.New()
			n := run(ctx, key, stored)
			Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: key}))
			Expect(calls).To(BeEmpty())
			Expect(created).To(BeEmpty())
		})
})
