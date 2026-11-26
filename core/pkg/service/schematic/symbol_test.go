// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

// heater returns a custom symbol with an enumeration state active, a number state
// level, a command activate that writes to active, and a command reset with no state.
func heater() symbol.Symbol {
	active := "active"
	return defaults.Fill(symbol.Symbol{
		Name: "Heater",
		Data: symbol.Spec{
			Drawing: symbol.Drawing{Variant: symbol.SvgDrawing{Markup: "<svg/>"}},
			Variant: "static",
			States: []symbol.State{
				{Variant: symbol.EnumerationState{
					BaseState: symbol.BaseState{Key: "active", Name: "Active"},
				}},
				{Variant: symbol.NumberState{
					BaseState: symbol.BaseState{Key: "level", Name: "Level"},
					Bounds:    spatial.ClosedBounds{Lower: 0, Upper: 100},
				}},
			},
			Commands: []symbol.Command{
				{
					Key:   "activate",
					Name:  "On",
					State: &active,
					Value: symbol.CommandValue{
						Variant: symbol.FixedCommandValue{Value: 1},
					},
				},
				{
					Key:  "reset",
					Name: "Reset",
					Value: symbol.CommandValue{
						Variant: symbol.FixedCommandValue{Value: 0},
					},
				},
			},
		},
	})
}

// createSymbol creates sym and deletes it when the spec ends.
func createSymbol(ctx SpecContext, sym *symbol.Symbol) {
	GinkgoHelper()
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Symbol.NewWriter(tx).Create(ctx, sym)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Symbol.NewWriter(tx).Delete(ctx, sym.Key)
		})).To(Succeed())
	})
}

// createSchematic creates s and deletes it when the spec ends.
func createSchematic(ctx SpecContext, s *schematic.Schematic) {
	GinkgoHelper()
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.NewWriter(tx).Create(ctx, s)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, s.Key)
		})).To(Succeed())
	})
}

func symbolNode(key string, cfg schematic.SymbolNodeConfig) schematic.Node {
	return configured(
		node(key, 0, 0),
		defaults.Fill(schematic.NodeConfig{Variant: cfg}),
	)
}

func customRef(sym symbol.Symbol) symbol.Ref {
	return symbol.Ref{Variant: symbol.CustomRef{Key: sym.Key}}
}

func builtinRef(name string) symbol.Ref {
	return symbol.Ref{Variant: symbol.BuiltinRef{Name: name}}
}

var _ = Describe("Symbol nodes", func() {
	It("Should refuse the delete of a custom symbol that a node draws", func(
		ctx SpecContext,
	) {
		sym := heater()
		createSymbol(ctx, &sym)
		s := schematic.Schematic{
			Project: &proj.Key,
			Name:    "symbols",
			Nodes: []schematic.Node{
				symbolNode("n", schematic.SymbolNodeConfig{Spec: customRef(sym)}),
			},
		}
		createSchematic(ctx, &s)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Symbol.NewWriter(tx).Delete(ctx, sym.Key)
		})).To(MatchError(gorp.RestrictError{
			Target:    gorp.NewTarget("schematic_symbol", sym.Key),
			Holder:    gorp.NewTarget("schematic", s.Key),
			Reference: "nodes.config.spec.key",
		}))
	})

	It("Should return the schematics whose symbol nodes draw the symbol as uses", func(
		ctx SpecContext,
	) {
		sym := heater()
		createSymbol(ctx, &sym)
		s := schematic.Schematic{
			Project: &proj.Key,
			Name:    "symbols",
			Nodes: []schematic.Node{
				symbolNode("n", schematic.SymbolNodeConfig{Spec: customRef(sym)}),
			},
		}
		createSchematic(ctx, &s)
		legacy := schematic.Schematic{
			Project: &proj.Key,
			Name:    "custom statics",
			Nodes: []schematic.Node{configured(
				node("n", 0, 0),
				defaults.Fill(schematic.NodeConfig{
					Variant: schematic.CustomStaticNodeConfig{
						SpecKey: sym.Key.String(),
					},
				}),
			)},
		}
		createSchematic(ctx, &legacy)
		Expect(svc.Symbol.Uses(ctx, db, sym.Key)).
			To(Equal([]resource.ID{schematic.ResourceID(s.Key)}))
	})

	Describe("State enumerations", func() {
		var (
			used, other enumeration.Enumeration
			s           schematic.Schematic
		)
		BeforeEach(func(ctx SpecContext) {
			created := []enumeration.Enumeration{{Name: "Breaker"}, {Name: "Valve"}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return enumSvc.NewWriter(tx).CreateMany(ctx, created)
			})).To(Succeed())
			used, other = created[0], created[1]
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return enumSvc.NewWriter(tx).Delete(ctx, used.Key, other.Key)
				})).To(Succeed())
			})
			s = schematic.Schematic{
				Project: &proj.Key,
				Name:    "symbols",
				Nodes: []schematic.Node{symbolNode("n", schematic.SymbolNodeConfig{
					Spec: builtinRef("valve"),
					States: []schematic.StateBinding{
						{Key: "active", Enumeration: &used.Key},
					},
				})},
			}
			createSchematic(ctx, &s)
		})

		It("Should refuse the delete of an enumeration that a state uses", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return enumSvc.NewWriter(tx).Delete(ctx, used.Key)
			})).To(MatchError(gorp.RestrictError{
				Target:    gorp.NewTarget("enumeration", used.Key),
				Holder:    gorp.NewTarget("schematic", s.Key),
				Reference: "nodes.config.states.enumeration",
			}))
		})

		It("Should move each state to the replacing enumeration", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return enumSvc.NewWriter(tx).Replace(ctx, used.Key, other.Key)
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			cfg := as[schematic.SymbolNodeConfig](res.Nodes[0].Config.Variant)
			Expect(cfg.States[0].Enumeration).To(Equal(&other.Key))
		})
	})

	Describe("Import", func() {
		exportOf := func(ctx SpecContext, s schematic.Schematic) imex.Envelope {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			env := MustSucceed(svc.Export(ctx, db, schematic.ResourceID(s.Key)))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
			return WireRoundTrip(env)
		}
		drawing := func(sym symbol.Symbol) schematic.Schematic {
			return schematic.Schematic{
				Project: &proj.Key,
				Name:    "exported",
				Nodes: []schematic.Node{
					symbolNode("a", schematic.SymbolNodeConfig{
						Spec:   customRef(sym),
						States: []schematic.StateBinding{{Key: "active"}},
						Commands: []schematic.CommandBinding{
							{Key: "reset", Channel: ch1},
						},
					}),
					symbolNode(
						"b",
						schematic.SymbolNodeConfig{Spec: builtinRef("valve")},
					),
				},
				Edges: []schematic.Edge{pipe("e", "a", "o", "b", "i", "")},
			}
		}

		It("Should draw a missing symbol for a custom symbol that the Core lacks", func(
			ctx SpecContext,
		) {
			gone := heater()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Symbol.NewWriter(tx).Create(ctx, &gone)
			})).To(Succeed())
			env := exportOf(ctx, drawing(gone))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Symbol.NewWriter(tx).Delete(ctx, gone.Key)
			})).To(Succeed())
			res := importSchematic(ctx, env)
			Expect(res.Nodes).To(HaveLen(2))
			Expect(res.Edges).To(HaveLen(1))
			cfg := as[schematic.SymbolNodeConfig](res.Nodes[0].Config.Variant)
			Expect(cfg.Spec).To(Equal(symbol.Ref{
				Variant: symbol.MissingRef{Key: gone.Key.String()},
			}))
			Expect(cfg.States).To(HaveLen(1))
			Expect(cfg.States[0].Key).To(Equal("active"))
			Expect(cfg.Commands).To(Equal([]schematic.CommandBinding{
				{Key: "reset", Channel: ch1},
			}))
		})

		It("Should keep a custom symbol that the Core has", func(ctx SpecContext) {
			sym := heater()
			createSymbol(ctx, &sym)
			res := importSchematic(ctx, exportOf(ctx, drawing(sym)))
			cfg := as[schematic.SymbolNodeConfig](res.Nodes[0].Config.Variant)
			Expect(cfg.Spec).To(Equal(customRef(sym)))
		})
	})

	It("Should store no format for a state that sets none", func(ctx SpecContext) {
		var s schematic.Schematic
		Expect(json.Unmarshal([]byte(
			`{"name":"symbols","nodes":[{"key":"n","config":{"variant":"symbol",`+
				`"spec":{"type":"builtin","name":"valve"},"states":[{"key":"active"}]}}]}`,
		), &s)).To(Succeed())
		s.Project = &proj.Key
		createSchematic(ctx, &s)
		var res schematic.Schematic
		Expect(svc.NewRetrieve().
			Where(schematic.MatchKeys(s.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		cfg := as[schematic.SymbolNodeConfig](res.Nodes[0].Config.Variant)
		Expect(cfg.States[0].FormatConfig).To(BeZero())
	})
})

var _ = Describe("Symbol node channel bindings", func() {
	// bound draws a heater, which commands reset with no state, from the channels.
	bound := func(
		ctx SpecContext,
		state, command, written channel.Key,
	) schematic.Schematic {
		sym := heater()
		createSymbol(ctx, &sym)
		return schematic.Schematic{
			Project: &proj.Key,
			Name:    "bound",
			Nodes: []schematic.Node{symbolNode("n", schematic.SymbolNodeConfig{
				Spec: customRef(sym),
				States: []schematic.StateBinding{{
					Key: "active",
					Source: &schematic.Source{Variant: schematic.ChannelSource{
						Key:     &state,
						Command: &command,
					}},
				}},
				Commands: []schematic.CommandBinding{{Key: "reset", Channel: written}},
			})},
		}
	}

	It("Should list a node as a reader of its state's channel and a writer of its "+
		"command channels", func(ctx SpecContext) {
		s := bound(ctx, ch1, ch2, ch3)
		createSchematic(ctx, &s)
		holder := gorp.NewTarget("schematic", s.Key)
		Expect(channelSvc.Uses(ctx, db, []channel.Key{ch1, ch2, ch3})).To(Equal(
			[][]channel.Use{
				{{Holder: holder, Direction: channel.DirectionRead}},
				{{Holder: holder, Direction: channel.DirectionWrite}},
				{{Holder: holder, Direction: channel.DirectionWrite}},
			},
		))
	})

	It("Should keep the binding keys of a node when their channels are deleted", func(
		ctx SpecContext,
	) {
		keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 3))
		s := bound(ctx, keys[0], keys[1], keys[2])
		createSchematic(ctx, &s)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).DeleteMany(ctx, keys, false)
		})).To(Succeed())
		var stored schematic.Schematic
		Expect(svc.NewRetrieve().
			Where(schematic.MatchKeys(s.Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		cfg := as[schematic.SymbolNodeConfig](stored.Nodes[0].Config.Variant)
		source := as[schematic.ChannelSource](cfg.States[0].Source.Variant)
		Expect(*source.Key).To(Equal(keys[0]))
		Expect(*source.Command).To(Equal(keys[1]))
		Expect(cfg.Commands[0].Channel).To(Equal(keys[2]))
	})
})
