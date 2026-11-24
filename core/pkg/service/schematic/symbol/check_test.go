// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func fixed(v float64) symbol.CommandValue {
	return symbol.CommandValue{Variant: symbol.FixedCommandValue{Value: v}}
}

func equals(state string, values ...float64) *symbol.Condition {
	return &symbol.Condition{
		Variant: symbol.EqualsCondition{State: state, Values: values},
	}
}

func entry(state string) *symbol.Paint {
	return &symbol.Paint{Variant: symbol.EntryPaint{State: state}}
}

func contrast(state string) *symbol.Paint {
	return &symbol.Paint{Variant: symbol.ContrastPaint{State: state}}
}

func motion(m symbol.MotionVariant) *symbol.Motion { return &symbol.Motion{Variant: m} }

func next(keys ...string) symbol.Interaction {
	return symbol.Interaction{Variant: symbol.NextInteraction{
		BaseInteraction: symbol.BaseInteraction{Key: "click", Part: new("body")},
		Commands:        keys,
	}}
}

func fire(key string, part *string, command string) symbol.Interaction {
	return symbol.Interaction{Variant: symbol.FireInteraction{
		BaseInteraction: symbol.BaseInteraction{Key: key, Part: part},
		Command:         command,
	}}
}

// valveSpec returns a spec that keeps every rule of the symbol model.
func valveSpec() symbol.Spec {
	return symbol.Spec{
		Drawing: svg(`<svg><rect id="body"/></svg>`),
		Variant: "static",
		Scale:   1,
		Parts: []symbol.Part{{
			Key:  "body",
			Name: "Body",
			Selectors: []string{
				"#body",
				"#-a",
				".a.b",
				`[data-region-id="r-1"]`,
				`#\31 23`,
			},
		}},
		Handles: []symbol.Handle{},
		States: []symbol.State{
			{Variant: symbol.EnumerationState{
				Key:  "active",
				Name: "Active",
				Match: symbol.Matcher{Variant: symbol.WithinMatcher{
					Bounds: spatial.Bounds{Lower: 0.9, Upper: 1.1},
				}},
			}},
			{Variant: symbol.NumberState{
				Key:    "level",
				Name:   "Level",
				Bounds: spatial.ClosedBounds{Upper: 100},
			}},
		},
		Commands: []symbol.Command{
			{Key: "off", Name: "Off", State: new("active"), Value: fixed(0)},
			{Key: "on", Name: "On", State: new("active"), Value: fixed(1)},
			{
				Key:   "set",
				Name:  "Set",
				State: new("level"),
				Value: symbol.CommandValue{Variant: symbol.AskCommandValue{}},
			},
		},
		Looks: []symbol.Look{
			{
				Key:    "fill",
				Part:   new("body"),
				When:   equals("active", 1),
				Fill:   entry("active"),
				Stroke: contrast("active"),
			},
			{
				Key:  "level",
				Part: new("body"),
				Motion: motion(symbol.LevelMotion{
					State: "level",
					Side:  spatial.OuterLocationBottom,
				}),
			},
		},
		Interactions: []symbol.Interaction{next("off", "on")},
	}
}

var _ = Describe("Checks", func() {
	create := func(ctx SpecContext, data symbol.Spec) error {
		GinkgoHelper()
		sym := symbol.Symbol{Name: "valve", Data: data}
		tx := DeferClose(db.OpenTx())
		return svc.NewWriter(tx).Create(ctx, &sym)
	}

	It("Should accept a symbol that keeps every rule", func(ctx SpecContext) {
		Expect(create(ctx, valveSpec())).To(Succeed())
	})

	It("Should accept one click on the whole drawing and one on each part", func(
		ctx SpecContext,
	) {
		data := valveSpec()
		data.Parts = append(data.Parts, symbol.Part{
			Key:       "lamp",
			Name:      "Lamp",
			Selectors: []string{"#lamp"},
		})
		data.Interactions = append(
			data.Interactions,
			fire("vent", nil, "on"),
			fire("test", new("lamp"), "off"),
		)
		Expect(create(ctx, data)).To(Succeed())
	})

	It("Should keep a negative spin and flow rate", func(ctx SpecContext) {
		data := valveSpec()
		data.Looks[1].Motion = motion(
			symbol.SpinMotion{Rate: -0.5, State: new("level")},
		)
		data.Looks = append(data.Looks, symbol.Look{
			Key:    "flow",
			Part:   new("body"),
			Motion: motion(symbol.FlowMotion{Rate: -2}),
		})
		sym := symbol.Symbol{Name: "pump", Data: data}
		tx := DeferClose(db.OpenTx())
		Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
		var res symbol.Symbol
		Expect(svc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&res).
			Exec(ctx, tx)).To(Succeed())
		Expect(res.Data.Looks[1].Motion.Variant).To(Equal(symbol.SpinMotion{
			Rate:  -0.5,
			State: new("level"),
		}))
		Expect(res.Data.Looks[2].Motion.Variant).To(Equal(symbol.FlowMotion{Rate: -2}))
	})

	DescribeTable("Should refuse a symbol that breaks a rule",
		func(ctx SpecContext, breakRule func(*symbol.Spec), message string) {
			data := valveSpec()
			breakRule(&data)
			Expect(create(ctx, data)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry("a repeated part key", func(s *symbol.Spec) {
			s.Parts = append(s.Parts, s.Parts[0])
		}, `data.parts.1.key: another part has the key "body"`),
		Entry("a repeated state key", func(s *symbol.Spec) {
			s.States = append(s.States, s.States[0])
		}, `data.states.2.key: another state has the key "active"`),
		Entry("a repeated command key", func(s *symbol.Spec) {
			s.Commands[1].Key = "off"
		}, `data.commands.1.key: another command has the key "off"`),
		Entry("a repeated look key", func(s *symbol.Spec) {
			s.Looks[1].Key = "fill"
		}, `data.looks.1.key: another look has the key "fill"`),
		Entry("a repeated handle key", func(s *symbol.Spec) {
			h := symbol.Handle{Key: "in", Orientation: spatial.OuterLocationLeft}
			s.Handles = []symbol.Handle{h, h}
		}, `data.handles.1.key: another handle has the key "in"`),
		Entry("a repeated interaction key", func(s *symbol.Spec) {
			s.Interactions = append(s.Interactions, next("off", "on"))
		}, `data.interactions.1.key: another interaction has the key "click"`),
		Entry("a selector by position", func(s *symbol.Spec) {
			s.Parts[0].Selectors = []string{"rect:nth-of-type(1)"}
		}, `data.parts.0.selectors.0: "rect:nth-of-type(1)" is not an #id`),
		Entry("a selector with a combinator", func(s *symbol.Spec) {
			s.Parts[0].Selectors = []string{"g > #body"}
		}, `data.parts.0.selectors.0: "g > #body" is not an #id`),
		Entry("an id that starts with a digit", func(s *symbol.Spec) {
			s.Parts[0].Selectors = []string{"#1"}
		}, `data.parts.0.selectors.0: "#1" is not an #id`),
		Entry("a class that starts with a digit", func(s *symbol.Spec) {
			s.Parts[0].Selectors = []string{".a.1a"}
		}, `data.parts.0.selectors.0: ".a.1a" is not an #id`),
		Entry("an id that starts with a hyphen and a digit", func(s *symbol.Spec) {
			s.Parts[0].Selectors = []string{"#-1"}
		}, `data.parts.0.selectors.0: "#-1" is not an #id`),
		Entry("within bounds whose lower is not below the upper", func(s *symbol.Spec) {
			s.States[0] = symbol.State{Variant: symbol.EnumerationState{
				Key: "active",
				Match: symbol.Matcher{Variant: symbol.WithinMatcher{
					Bounds: spatial.Bounds{Lower: 1, Upper: 1},
				}},
			}}
		}, "data.states.0.match.bounds: lower 1 must be below upper 1"),
		Entry("number bounds whose lower is not below the upper", func(s *symbol.Spec) {
			s.States[1] = symbol.State{Variant: symbol.NumberState{
				Key:    "level",
				Bounds: spatial.ClosedBounds{Lower: 5, Upper: 0},
			}}
		}, "must be less than upper"),
		Entry("an ask command with no state", func(s *symbol.Spec) {
			s.Commands[2].State = nil
		}, "data.commands.2.state: an ask command needs a state"),
		Entry("a command on no state", func(s *symbol.Spec) {
			s.Commands[2].State = new("pressure")
		}, `data.commands.2.state: "pressure" names no state`),
		Entry("a look on no part", func(s *symbol.Spec) {
			s.Looks[0].Part = new("lid")
		}, `data.looks.0.part: "lid" names no part`),
		Entry("an equals condition on a number state", func(s *symbol.Spec) {
			s.Looks[0].When = equals("level", 1)
		}, `data.looks.0.when.state: "level" names a number state`),
		Entry("an entry paint on a number state", func(s *symbol.Spec) {
			s.Looks[0].Fill = entry("level")
		}, `data.looks.0.fill.state: "level" names a number state`),
		Entry("a contrast paint on a number state", func(s *symbol.Spec) {
			s.Looks[0].Stroke = contrast("level")
		}, `data.looks.0.stroke.state: "level" names a number state`),
		Entry("a motion on an enumeration state", func(s *symbol.Spec) {
			s.Looks[1].Motion = motion(symbol.LevelMotion{
				State: "active",
				Side:  spatial.OuterLocationBottom,
			})
		}, `data.looks.1.motion.state: "active" names an enumeration state`),
		Entry("a spin rate that is not a number", func(s *symbol.Spec) {
			s.Looks[1].Motion = motion(symbol.SpinMotion{Rate: math.NaN()})
		}, "data.looks.1.motion.rate: must be finite"),
		Entry("an infinite flow rate", func(s *symbol.Spec) {
			s.Looks[1].Motion = motion(symbol.FlowMotion{Rate: math.Inf(-1)})
		}, "data.looks.1.motion.rate: must be finite"),
		Entry("an interaction on no part", func(s *symbol.Spec) {
			s.Interactions[0] = fire("click", new("lid"), "on")
		}, `data.interactions.0.part: "lid" names no part`),
		Entry("a fire of no command", func(s *symbol.Spec) {
			s.Interactions[0] = fire("click", nil, "vent")
		}, `data.interactions.0.command: "vent" names no command`),
		Entry("a second interaction on one part", func(s *symbol.Spec) {
			s.Interactions = append(s.Interactions, fire("vent", new("body"), "on"))
		}, `data.interactions.1.part: another interaction is on part "body"`),
		Entry("a second interaction on the whole drawing", func(s *symbol.Spec) {
			s.Interactions = []symbol.Interaction{
				fire("open", nil, "on"),
				fire("close", nil, "off"),
			}
		}, "data.interactions.1.part: another interaction is on the whole drawing"),
		Entry("a next with one command", func(s *symbol.Spec) {
			s.Interactions[0] = next("on")
		}, "data.interactions.0.commands: next needs two commands, got 1"),
		Entry("a next with an ask command", func(s *symbol.Spec) {
			s.Commands[2].State = new("active")
			s.Interactions[0] = next("off", "set")
		}, `data.interactions.0.commands: command "set" must be fixed and have a state`),
		Entry("a next across two states", func(s *symbol.Spec) {
			s.Commands[2].Value = fixed(5)
			s.Interactions[0] = next("off", "set")
		}, `data.interactions.0.commands: commands "off" and "set" must have one state`),
		Entry("a next whose values match to one value", func(s *symbol.Spec) {
			s.Commands[1].Value = fixed(0.5)
		}, `commands "off" and "on" must write different values to state "active"`),
	)

	It("Should refuse an edit that breaks a rule", func(ctx SpecContext) {
		sym := symbol.Symbol{Name: "valve", Data: valveSpec()}
		tx := DeferClose(db.OpenTx())
		Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
		data := valveSpec()
		data.Looks[0].Part = new("lid")
		Expect(svc.NewWriter(tx).Apply(ctx, sym.Key, symbol.NewSetAction(
			symbol.SetPayload{Path: []string{"data"}, Value: data},
		))).To(MatchError(ContainSubstring(`data.looks.0.part: "lid" names no part`)))
	})
})

var _ = Describe("Migrated selectors", func() {
	DescribeTable(
		"Should pass the checks of a v4 name selector that they refuse",
		func(ctx SpecContext, markup, sel string) {
			data := MustSucceed(v5.MigrateSpec(ctx, v4.Spec{
				SVG:     markup,
				Variant: "static",
				Scale:   1,
				States: []v4.State{{Key: "off", Name: "Off", Regions: []v4.Region{
					{Key: "r", Name: "r", Selectors: []string{sel}},
				}}},
			}))
			sym := symbol.Symbol{Name: "Migrated", Data: data}
			tx := DeferClose(db.OpenTx())
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
		},
		Entry("an id that starts with a digit", `<svg><rect id="1a"/></svg>`, "#1a"),
		Entry("an id with a comma", `<svg><rect id="a,b"/></svg>`, "#a,b"),
		Entry(
			"a class that starts with a digit",
			`<svg><rect class="1a"/></svg>`,
			".1a",
		),
		Entry("a name that selects no shape", `<svg><rect/></svg>`, "#a,b"),
	)
})

var _ = Describe("CheckBounds", func() {
	It("Should accept bounds whose lower is below their upper", func() {
		Expect(symbol.CheckBounds(
			[]string{"bounds"},
			spatial.Bounds{Lower: 0.9, Upper: 1.1},
		)).To(Succeed())
	})

	DescribeTable(
		"Should refuse bounds whose lower is not below their upper",
		func(b spatial.Bounds, msg string) {
			Expect(symbol.CheckBounds([]string{"match", "bounds"}, b)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("equal", spatial.Bounds{Lower: 1, Upper: 1},
			"match.bounds: lower 1 must be below upper 1"),
		Entry("inverted", spatial.Bounds{Lower: 2, Upper: 1},
			"match.bounds: lower 2 must be below upper 1"),
	)
})
