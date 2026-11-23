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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Symbol node bindings", func() {
	var heater symbol.Symbol

	BeforeEach(func(ctx SpecContext) {
		active := "active"
		heater = defaults.Fill(symbol.Symbol{
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
						Bounds:    spatial.Bounds{Lower: 0, Upper: 100},
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
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Symbol.NewWriter(tx).Create(ctx, &heater)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Symbol.NewWriter(tx).Delete(ctx, heater.Key)
			})).To(Succeed())
		})
	})

	custom := func() symbol.Ref {
		return symbol.Ref{Variant: symbol.CustomRef{Key: heater.Key}}
	}
	builtin := func(name string) symbol.Ref {
		return symbol.Ref{Variant: symbol.BuiltinRef{Name: name}}
	}
	// fits returns a node of the custom symbol that binds each of its states and its
	// command with no state.
	fits := func() schematic.SymbolNodeConfig {
		return schematic.SymbolNodeConfig{
			Spec: custom(),
			States: []schematic.StateBinding{
				{Key: "active", Source: &schematic.Source{
					Variant: schematic.ChannelSource{Key: new(channel.Key(1))},
				}},
				{Key: "level", Bounds: &spatial.Bounds{Lower: 10, Upper: 20}},
			},
			Commands: []schematic.CommandBinding{{Key: "reset", Channel: 2}},
		}
	}
	diagram := func(cfg schematic.SymbolNodeConfig) schematic.Schematic {
		return schematic.Schematic{
			Project: &proj.Key,
			Name:    "symbols",
			Nodes: []schematic.Node{configured(
				node("n", 0, 0),
				defaults.Fill(schematic.NodeConfig{Variant: cfg}),
			)},
		}
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

	It(
		"Should save a node whose bindings fit its custom symbol",
		func(ctx SpecContext) {
			s := diagram(fits())
			Expect(create(ctx, &s)).To(Succeed())
		},
	)

	It("Should save a node whose bindings fit its built-in symbol", func(
		ctx SpecContext,
	) {
		s := diagram(schematic.SymbolNodeConfig{
			Spec: builtin("valve"),
			States: []schematic.StateBinding{{
				Key:     "active",
				Match:   &symbol.Matcher{Variant: symbol.ExactMatcher{}},
				Default: new(1.0),
			}},
		})
		Expect(create(ctx, &s)).To(Succeed())
	})

	It("Should save the bindings of a missing symbol unchecked", func(ctx SpecContext) {
		s := diagram(schematic.SymbolNodeConfig{
			Spec:     symbol.Ref{Variant: symbol.MissingRef{Key: "gone"}},
			States:   []schematic.StateBinding{{Key: "anything"}},
			Commands: []schematic.CommandBinding{{Key: "anything", Channel: 2}},
		})
		Expect(create(ctx, &s)).To(Succeed())
	})

	DescribeTable(
		"Should refuse a node that its symbol does not allow",
		func(ctx SpecContext, edit func(*schematic.SymbolNodeConfig), msg string) {
			cfg := fits()
			edit(&cfg)
			s := diagram(cfg)
			Expect(create(ctx, &s)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry(
			"a built-in name that the Core lacks",
			func(c *schematic.SymbolNodeConfig) {
				c.Spec = builtin("warp_drive")
			},
			`nodes.0.config.spec.name: "warp_drive" names no built-in symbol`,
		),
		Entry("a state that the built-in lacks", func(c *schematic.SymbolNodeConfig) {
			c.Spec = builtin("valve")
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
			s := diagram(cfg)
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
		s := diagram(fits())
		Expect(create(ctx, &s)).To(Succeed())
		cfg := fits()
		cfg.States[0].Key = "speed"
		Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{
			schematic.NewSetNodeAction(schematic.SetNodePayload{Node: configured(
				node("n", 0, 0),
				defaults.Fill(schematic.NodeConfig{Variant: cfg}),
			)}),
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				`nodes.0.config.states.0.key: "speed" names no state of the symbol`,
			)),
		))
	})

	It("Should refuse the delete of a custom symbol that a node draws", func(
		ctx SpecContext,
	) {
		s := diagram(fits())
		Expect(create(ctx, &s)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Symbol.NewWriter(tx).Delete(ctx, heater.Key)
		})).To(MatchError(gorp.RestrictError{
			Target:    gorp.NewTarget("schematic_symbol", heater.Key),
			Holder:    gorp.NewTarget("schematic", s.Key),
			Reference: "nodes.config.spec.key",
		}))
	})
})
