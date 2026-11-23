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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Symbol node bindings", func() {
	var sym symbol.Symbol
	BeforeEach(func(ctx SpecContext) {
		sym = heater()
		createSymbol(ctx, &sym)
	})

	// fits returns a node of sym that binds each of its states and its command with no
	// state.
	fits := func() schematic.SymbolNodeConfig {
		return schematic.SymbolNodeConfig{
			Spec: customRef(sym),
			States: []schematic.StateBinding{
				{Key: "active", Source: &schematic.Source{
					Variant: schematic.ChannelSource{Key: new(channel.Key(1))},
				}},
				{Key: "level", Bounds: &spatial.Bounds{Lower: 10, Upper: 20}},
			},
			Commands: []schematic.CommandBinding{{Key: "reset", Channel: 2}},
		}
	}
	diagram := func(nodes ...schematic.Node) schematic.Schematic {
		return schematic.Schematic{Project: &proj.Key, Name: "symbols", Nodes: nodes}
	}
	create := func(ctx SpecContext, s *schematic.Schematic) error {
		GinkgoHelper()
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, s)
		})
		if err == nil {
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, s.Key)
				})).To(Succeed())
			})
		}
		return err
	}
	// importNode imports a file that holds one node with the symbol and bindings of
	// cfg, and returns the config that the import stores.
	importNode := func(
		ctx SpecContext,
		cfg schematic.SymbolNodeConfig,
	) schematic.SymbolNodeConfig {
		GinkgoHelper()
		s := diagram(symbolNode("n", schematic.SymbolNodeConfig{Spec: cfg.Spec}))
		Expect(create(ctx, &s)).To(Succeed())
		env := WireRoundTrip(
			MustSucceed(svc.Export(ctx, db, schematic.ResourceID(s.Key))),
		)
		file := MustSucceed(env.Decode[schematic.Schematic]())
		node := as[schematic.SymbolNodeConfig](file.Nodes[0].Config.Variant)
		node.States, node.Commands = cfg.States, cfg.Commands
		file.Nodes[0].Config.Variant = node
		Expect(env.Encode(file)).To(Succeed())
		res := importSchematic(ctx, WireRoundTrip(env))
		return as[schematic.SymbolNodeConfig](res.Nodes[0].Config.Variant)
	}

	It("Should save a node whose bindings fit its custom symbol", func(
		ctx SpecContext,
	) {
		s := diagram(symbolNode("n", fits()))
		Expect(create(ctx, &s)).To(Succeed())
	})

	It("Should save a node whose bindings fit its built-in symbol", func(
		ctx SpecContext,
	) {
		s := diagram(symbolNode("n", schematic.SymbolNodeConfig{
			Spec: builtinRef("valve"),
			States: []schematic.StateBinding{{
				Key:     "active",
				Match:   &symbol.Matcher{Variant: symbol.ExactMatcher{}},
				Default: new(1.0),
			}},
		}))
		Expect(create(ctx, &s)).To(Succeed())
	})

	It("Should save the bindings of a missing symbol unchecked", func(ctx SpecContext) {
		s := diagram(symbolNode("n", schematic.SymbolNodeConfig{
			Spec:     symbol.Ref{Variant: symbol.MissingRef{Key: "gone"}},
			States:   []schematic.StateBinding{{Key: "anything"}},
			Commands: []schematic.CommandBinding{{Key: "anything", Channel: 2}},
		}))
		Expect(create(ctx, &s)).To(Succeed())
	})

	DescribeTable(
		"Should refuse a node that its symbol does not allow",
		func(ctx SpecContext, edit func(*schematic.SymbolNodeConfig), msg string) {
			cfg := fits()
			edit(&cfg)
			s := diagram(symbolNode("n", cfg))
			Expect(create(ctx, &s)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry(
			"a built-in name that the Core lacks",
			func(c *schematic.SymbolNodeConfig) {
				c.Spec = builtinRef("warp_drive")
			},
			`nodes.0.config.spec.name: "warp_drive" names no built-in symbol`,
		),
		Entry("a state that the built-in lacks", func(c *schematic.SymbolNodeConfig) {
			c.Spec = builtinRef("valve")
		}, `nodes.0.config.states.1.key: "level" names no state of the symbol`),
		Entry("a repeated state binding", func(c *schematic.SymbolNodeConfig) {
			c.States = append(c.States, c.States[0])
		}, `nodes.0.config.states.2.key: another state binding has the key "active"`),
		Entry("bounds on an enumeration state", func(c *schematic.SymbolNodeConfig) {
			c.States[0].Bounds = &spatial.Bounds{Lower: 0, Upper: 1}
		}, `nodes.0.config.states.0.bounds: enumeration state "active" has no bounds`),
		Entry("an empty match on an enumeration state", func(
			c *schematic.SymbolNodeConfig,
		) {
			c.States[0].Match = &symbol.Matcher{Variant: symbol.WithinMatcher{
				Bounds: spatial.Bounds{Lower: 1, Upper: 1},
			}}
		}, `nodes.0.config.states.0.match.bounds: lower 1 must be below upper 1`),
		Entry("an enumeration on a number state", func(c *schematic.SymbolNodeConfig) {
			c.States[1].Enumeration = new(enumeration.ValveKey)
		}, `nodes.0.config.states.1.enumeration: number state "level" has no enumeration`),
		Entry("a match on a number state", func(c *schematic.SymbolNodeConfig) {
			c.States[1].Match = &symbol.Matcher{Variant: symbol.ExactMatcher{}}
		}, `nodes.0.config.states.1.match: number state "level" has no match`),
		Entry("a default on a number state", func(c *schematic.SymbolNodeConfig) {
			c.States[1].Default = new(0.0)
		}, `nodes.0.config.states.1.default: number state "level" has no default`),
		Entry("empty bounds on a number state", func(c *schematic.SymbolNodeConfig) {
			c.States[1].Bounds = &spatial.Bounds{Lower: 5, Upper: 2}
		}, `nodes.0.config.states.1.bounds: lower 5 must be below upper 2`),
		Entry("a repeated command binding", func(c *schematic.SymbolNodeConfig) {
			c.Commands = append(c.Commands, c.Commands[0])
		}, `nodes.0.config.commands.1.key: another command binding has the key "reset"`),
		Entry("a command that the symbol lacks", func(c *schematic.SymbolNodeConfig) {
			c.Commands[0].Key = "stop"
		}, `nodes.0.config.commands.0.key: "stop" names no command of the symbol`),
		Entry("a command that has a state", func(c *schematic.SymbolNodeConfig) {
			c.Commands[0].Key = "activate"
		}, `nodes.0.config.commands.0.key: command "activate" writes to the channel `+
			`of state "active"`),
	)

	It(
		"Should refuse a node whose custom symbol does not exist",
		func(ctx SpecContext) {
			cfg := fits()
			gone := uuid.New()
			cfg.Spec = symbol.Ref{Variant: symbol.CustomRef{Key: gone}}
			s := diagram(symbolNode("n", cfg))
			Expect(create(ctx, &s)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"nodes.config.spec.key: schematic_symbol %s does not exist",
					gone,
				)),
			))
		},
	)

	It("Should check the bindings that an action sets", func(ctx SpecContext) {
		s := diagram(symbolNode("n", fits()))
		Expect(create(ctx, &s)).To(Succeed())
		cfg := fits()
		cfg.States[0].Key = "speed"
		Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{
			schematic.NewSetNodeAction(schematic.SetNodePayload{
				Node: symbolNode("n", cfg),
			}),
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				`nodes.0.config.states.0.key: "speed" names no state of the symbol`,
			)),
		))
	})

	Describe("A node that a symbol change outside a save leaves stale", func() {
		var s schematic.Schematic
		BeforeEach(func(ctx SpecContext) {
			s = diagram(
				symbolNode("stale", fits()),
				symbolNode("other", schematic.SymbolNodeConfig{
					Spec: builtinRef("valve"),
				}),
			)
			Expect(create(ctx, &s)).To(Succeed())
			// A raw update skips the symbol Writer, so no save removes the level
			// binding. A built-in that a new Core changes leaves a node stale the same
			// way.
			Expect(gorp.NewUpdate[symbol.Key, symbol.Symbol]().
				Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](sym.Key)).
				Change(func(_ gorp.Context, sym symbol.Symbol) symbol.Symbol {
					sym.Data.States = sym.Data.States[:1]
					return sym
				}).
				Exec(ctx, db)).To(Succeed())
		})
		refused := func(index string) types.GomegaMatcher {
			return SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					`nodes.0.config.states.%s.key: "level" names no state of the symbol`,
					index,
				)),
			)
		}

		It("Should copy a schematic that holds the stale node", func(ctx SpecContext) {
			var cpy schematic.Schematic
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Copy(ctx, s.Key, "Copy", &cpy)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, cpy.Key)
				})).To(Succeed())
			})
			Expect(cpy.Nodes).To(Equal(s.Nodes))
		})

		It("Should import an export of the schematic without the stale binding", func(
			ctx SpecContext,
		) {
			env := MustSucceed(svc.Export(ctx, db, schematic.ResourceID(s.Key)))
			res := importSchematic(ctx, WireRoundTrip(env))
			cfg := as[schematic.SymbolNodeConfig](res.Nodes[0].Config.Variant)
			Expect(cfg.States).To(HaveExactElements(HaveField("Key", "active")))
			Expect(cfg.Commands).To(Equal(fits().Commands))
		})

		DescribeTable(
			"Should check the stale node only when an action rebinds it",
			func(
				ctx SpecContext,
				act func() schematic.Action,
				match types.GomegaMatcher,
			) {
				Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{act()})).
					To(match)
			},
			Entry("a label of another node", func() schematic.Action {
				return schematic.NewSetAction(schematic.SetPayload{
					Path:  []string{"nodes", "other", "config", "label", "label"},
					Value: "Inlet",
				})
			}, Succeed()),
			Entry("a label of the stale node", func() schematic.Action {
				return schematic.NewSetAction(schematic.SetPayload{
					Path:  []string{"nodes", "stale", "config", "label", "label"},
					Value: "Heater",
				})
			}, Succeed()),
			Entry("a position of the stale node", func() schematic.Action {
				return schematic.NewSetNodePositionAction(
					schematic.SetNodePositionPayload{Key: "stale"},
				)
			}, Succeed()),
			Entry("a remove of the stale binding", func() schematic.Action {
				return schematic.NewRemoveAction(schematic.RemovePayload{
					Path: []string{"nodes", "stale", "config", "states"},
					Key:  "level",
				})
			}, Succeed()),
			Entry("a set into the states of the stale node", func() schematic.Action {
				return schematic.NewSetAction(schematic.SetPayload{
					Path: []string{
						"nodes", "stale", "config", "states", "active", "units",
					},
					Value: "C",
				})
			}, refused("1")),
			Entry(
				"a remove from the commands of the stale node",
				func() schematic.Action {
					return schematic.NewRemoveAction(schematic.RemovePayload{
						Path: []string{"nodes", "stale", "config", "commands"},
						Key:  "reset",
					})
				},
				refused("1"),
			),
			Entry(
				"an insert into the states of the stale node",
				func() schematic.Action {
					return schematic.NewInsertAction(schematic.InsertPayload{
						Path:  []string{"nodes", "stale", "config", "states"},
						Value: map[string]any{"key": "active"},
					})
				},
				refused("1"),
			),
			Entry("a move in the states of the stale node", func() schematic.Action {
				return schematic.NewMoveAction(schematic.MovePayload{
					Path: []string{"nodes", "stale", "config", "states"},
					Key:  "level",
				})
			}, refused("0")),
			Entry("a set of the spec of the stale node", func() schematic.Action {
				return schematic.NewSetAction(schematic.SetPayload{
					Path:  []string{"nodes", "stale", "config", "spec"},
					Value: map[string]any{"type": "builtin", "name": "valve"},
				})
			}, refused("1")),
			Entry("a set of the config of the stale node", func() schematic.Action {
				return schematic.NewSetAction(schematic.SetPayload{
					Path:  []string{"nodes", "stale", "config"},
					Value: fields(s.Nodes[0].Config),
				})
			}, refused("1")),
			Entry("a set of the stale node", func() schematic.Action {
				return schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: s.Nodes[0],
				})
			}, refused("1")),
		)
	})

	Describe("A symbol save", func() {
		var (
			s    schematic.Schematic
			twin symbol.Symbol
		)
		// bound returns the bindings of fits, with a default on the active state.
		bound := func() schematic.SymbolNodeConfig {
			cfg := fits()
			cfg.States[0].Default = new(1.0)
			return cfg
		}
		BeforeEach(func(ctx SpecContext) {
			twin = heater()
			createSymbol(ctx, &twin)
			other := bound()
			other.Spec = customRef(twin)
			s = diagram(
				symbolNode("n", bound()),
				symbolNode("twin", other),
				symbolNode("valve", schematic.SymbolNodeConfig{
					Spec: builtinRef("valve"),
				}),
			)
			Expect(create(ctx, &s)).To(Succeed())
		})
		save := func(ctx SpecContext, act symbol.Action) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Symbol.NewWriter(tx).Apply(ctx, sym.Key, act)
			})).To(Succeed())
		}
		stored := func(ctx SpecContext) schematic.Schematic {
			GinkgoHelper()
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}
		removeLevel := symbol.NewRemoveAction(symbol.RemovePayload{
			Path: []string{"data", "states"},
			Key:  "level",
		})

		DescribeTable(
			"Should remove the bindings that the save orphans",
			func(
				ctx SpecContext,
				act func() symbol.Action,
				states types.GomegaMatcher,
				commands types.GomegaMatcher,
			) {
				save(ctx, act())
				cfg := as[schematic.SymbolNodeConfig](
					stored(ctx).Nodes[0].Config.Variant,
				)
				Expect(cfg.States).To(states)
				Expect(cfg.Commands).To(commands)
			},
			Entry(
				"a binding of a removed state",
				func() symbol.Action { return removeLevel },
				HaveExactElements(HaveField("Key", "active")),
				Equal(bound().Commands),
			),
			Entry(
				"the bounds of a state that becomes an enumeration",
				func() symbol.Action {
					return symbol.NewSetAction(symbol.SetPayload{
						Path: []string{"data", "states", "level"},
						Value: fields(symbol.State{Variant: symbol.EnumerationState{
							BaseState: symbol.BaseState{Key: "level", Name: "Level"},
							Match:     symbol.Matcher{Variant: symbol.ExactMatcher{}},
						}}),
					})
				},
				HaveExactElements(
					Equal(bound().States[0]),
					SatisfyAll(HaveField("Key", "level"), HaveField("Bounds", BeNil())),
				),
				Equal(bound().Commands),
			),
			Entry(
				"the default of a state that becomes a number",
				func() symbol.Action {
					return symbol.NewSetAction(symbol.SetPayload{
						Path: []string{"data", "states", "active"},
						Value: fields(symbol.State{Variant: symbol.NumberState{
							BaseState: symbol.BaseState{Key: "active", Name: "Active"},
							Bounds:    spatial.Bounds{Lower: 0, Upper: 1},
						}}),
					})
				},
				HaveExactElements(
					Equal(fits().States[0]),
					Equal(bound().States[1]),
				),
				Equal(bound().Commands),
			),
			Entry(
				"a binding of a removed command",
				func() symbol.Action {
					return symbol.NewRemoveAction(symbol.RemovePayload{
						Path: []string{"data", "commands"},
						Key:  "reset",
					})
				},
				Equal(bound().States),
				BeEmpty(),
			),
			Entry(
				"a binding of a command that gains a state",
				func() symbol.Action {
					return symbol.NewSetAction(symbol.SetPayload{
						Path:  []string{"data", "commands", "reset", "state"},
						Value: "active",
					})
				},
				Equal(bound().States),
				BeEmpty(),
			),
		)

		It("Should send a set node action for each node that it changes", func(
			ctx SpecContext,
		) {
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			save(ctx, removeLevel)
			res := stored(ctx)
			Expect(res.Nodes[1:]).To(Equal(s.Nodes[1:]))
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(s.Key))
			Expect(seen[0].Actions).To(HaveExactElements(
				schematic.NewSetNodeAction(
					schematic.SetNodePayload{Node: res.Nodes[0]},
				),
			))
		})

		It("Should change no schematic when the save orphans no binding", func(
			ctx SpecContext,
		) {
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			save(ctx, symbol.NewSetAction(symbol.SetPayload{
				Path:  []string{"data", "states", "level", "name"},
				Value: "Fill",
			}))
			Expect(stored(ctx).Nodes).To(Equal(s.Nodes))
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})

	DescribeTable(
		"Should drop on import the bindings that the symbol orphans",
		func(
			ctx SpecContext,
			cfg func(ctx SpecContext) schematic.SymbolNodeConfig,
			states types.GomegaMatcher,
			commands types.GomegaMatcher,
		) {
			res := importNode(ctx, cfg(ctx))
			Expect(res.States).To(states)
			Expect(res.Commands).To(commands)
		},
		Entry(
			"a binding of a state that the symbol lacks",
			func(SpecContext) schematic.SymbolNodeConfig {
				return schematic.SymbolNodeConfig{
					Spec:   customRef(sym),
					States: []schematic.StateBinding{{Key: "gone"}, {Key: "active"}},
				}
			},
			HaveExactElements(HaveField("Key", "active")),
			BeEmpty(),
		),
		Entry(
			"a binding of a state that a built-in lacks",
			func(SpecContext) schematic.SymbolNodeConfig {
				return schematic.SymbolNodeConfig{
					Spec:   builtinRef("valve"),
					States: []schematic.StateBinding{{Key: "gone"}, {Key: "active"}},
				}
			},
			HaveExactElements(HaveField("Key", "active")),
			BeEmpty(),
		),
		Entry(
			"the overrides that a number state lacks",
			func(ctx SpecContext) schematic.SymbolNodeConfig {
				enums := []enumeration.Enumeration{{Name: "Level"}}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return enumSvc.NewWriter(tx).CreateMany(ctx, enums)
				})).To(Succeed())
				enum := enums[0]
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return enumSvc.NewWriter(tx).Delete(ctx, enum.Key)
					})).To(Succeed())
				})
				return schematic.SymbolNodeConfig{
					Spec: customRef(sym),
					States: []schematic.StateBinding{{
						Key:         "level",
						Units:       new("C"),
						Enumeration: &enum.Key,
						Match:       &symbol.Matcher{Variant: symbol.ExactMatcher{}},
						Default:     new(1.0),
					}},
				}
			},
			HaveExactElements(SatisfyAll(
				HaveField("Key", "level"),
				HaveField("Units", HaveValue(Equal("C"))),
				HaveField("Enumeration", BeNil()),
				HaveField("Match", BeNil()),
				HaveField("Default", BeNil()),
			)),
			BeEmpty(),
		),
		Entry(
			"the bounds of an enumeration state",
			func(SpecContext) schematic.SymbolNodeConfig {
				return schematic.SymbolNodeConfig{
					Spec: customRef(sym),
					States: []schematic.StateBinding{{
						Key:    "active",
						Bounds: &spatial.Bounds{Lower: 0, Upper: 1},
					}},
				}
			},
			HaveExactElements(SatisfyAll(
				HaveField("Key", "active"),
				HaveField("Bounds", BeNil()),
			)),
			BeEmpty(),
		),
		Entry(
			"a binding of a command that the symbol lacks",
			func(SpecContext) schematic.SymbolNodeConfig {
				return schematic.SymbolNodeConfig{
					Spec: customRef(sym),
					Commands: []schematic.CommandBinding{
						{Key: "gone", Channel: 2},
						{Key: "reset", Channel: 2},
					},
				}
			},
			BeEmpty(),
			HaveExactElements(HaveField("Key", "reset")),
		),
		Entry(
			"a binding of a command that has a state",
			func(SpecContext) schematic.SymbolNodeConfig {
				return schematic.SymbolNodeConfig{
					Spec:     customRef(sym),
					Commands: []schematic.CommandBinding{{Key: "activate", Channel: 2}},
				}
			},
			BeEmpty(),
			BeEmpty(),
		),
	)
})
