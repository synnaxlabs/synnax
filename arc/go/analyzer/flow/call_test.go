// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flow_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calls of a sequence or stage with parameters", func() {
	const declaration = `
		sequence main(limit f64, valve chan f64) {
			stage press {
				1.0 -> valve
				pressure > limit => next
			}
			stage vent {
				0.0 -> valve
			}
		}
	`
	channels := []symbol.Symbol{
		{
			Name: "start",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.Bool()),
			ID:   1,
		},
		{
			Name: "pressure",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   2,
		},
		{
			Name: "valve_a",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   3,
		},
		{
			Name: "valve_b",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   4,
		},
	}
	analyze := func(bCtx SpecContext, source string) context.Context[parser.IProgramContext] {
		GinkgoHelper()
		ast := MustSucceed(parser.Parse(declaration + source))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, channels...))
		analyzer.AnalyzeProgram(ctx)
		return ctx
	}

	DescribeTable("Valid",
		func(bCtx SpecContext, source string) {
			ctx := analyze(bCtx, source)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		},
		Entry("arguments by position", `start => main(50.0, valve_a)`),
		Entry("arguments by name", `start => main(valve=valve_a, limit=50.0)`),
		Entry("a continuous arrow", `start -> main(50.0, valve_a)`),
		Entry("a negative literal", `start => main(-1.0, valve_a)`),
		Entry("two calls with different arguments", `
			start => main(50.0, valve_a)
			start => main(80.0, valve_b)
		`),
		Entry("a top-level constant and a top-level alias", `
			high := 80.0
			out := valve_b
			start => main(high, out)
		`),
		Entry("a call from a stage of another sequence", `
			sequence outer {
				stage idle {
					start => main(50.0, valve_a)
				}
			}
		`),
		Entry("the parameters of the caller as arguments", `
			sequence outer(top f64, out chan f64) {
				stage idle {
					start => main(top, out)
				}
			}
			start => outer(50.0, valve_a)
		`),
		Entry("a variable of the calling sequence", `
			sequence outer {
				local := 5.0
				stage idle {
					start => main(local, valve_a)
				}
			}
		`),
		Entry("a top-level variable computed at run time", `
			double := 2.0 * 3.0
			start => main(double, valve_a)
		`),
		Entry("an expression over a variable of the calling sequence", `
			sequence outer {
				local := 5.0
				stage idle {
					start => main(local * 2.0, valve_a)
				}
			}
		`),
		Entry("an expression over a parameter of the caller", `
			sequence outer(top f64, out chan f64) {
				stage idle {
					start => main(top * 2.0, out)
				}
			}
			start => outer(50.0, valve_a)
		`),
		Entry("a call in a routing table entry", `
			func split() {value f64} (high f64, low f64) {
				high = value
			}
			pressure -> split() -> {
				high: true => main(80.0, valve_a),
				low: true => main(20.0, valve_b)
			}
		`),
	)

	DescribeTable(
		"Invalid",
		func(bCtx SpecContext, source, message string) {
			ctx := analyze(bCtx, source)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(message))
		},
		Entry(
			"the bare name",
			`start => main`,
			"invalid transition target: main has parameters",
		),
		Entry("a channel alias of the calling sequence for a channel parameter", `
			sequence outer {
				out := valve_a
				stage idle {
					start => main(50.0, out)
				}
			}
		`, "not a constant: argument for parameter 'valve' of sequence 'main' is not fixed at compile time"),
		Entry(
			"an expression for a channel parameter",
			`start => main(50.0, pressure * 2.0)`,
			"channel expected: argument for parameter 'valve' of sequence 'main' must be a channel",
		),
		Entry(
			"a channel for a value parameter",
			`start => main(pressure, valve_a)`,
			"invalid argument value: argument for parameter 'limit' of sequence 'main' cannot be a channel",
		),
		Entry("a channel alias for a value parameter", `
			out := valve_b
			start => main(out, valve_a)
		`, "invalid argument value: argument for parameter 'limit' of sequence 'main' cannot be a channel"),
		Entry(
			"a value for a channel parameter",
			`start => main(50.0, 1.0)`,
			"channel expected: argument for parameter 'valve' of sequence 'main' must be a channel",
		),
		Entry("a missing argument", `start => main(50.0)`,
			"missing required argument: parameter 'valve' of sequence 'main'"),
		Entry("an extra argument", `start => main(50.0, valve_a, valve_b)`,
			"too many arguments: sequence 'main' takes at most 2"),
		Entry("an unknown parameter name", `start => main(gain=50.0, valve=valve_a)`,
			"unknown parameter: 'gain' for sequence 'main'"),
		Entry("a parameter bound two times", `start => main(limit=1.0, limit=2.0)`,
			"duplicate argument: parameter 'limit' of sequence 'main'"),
		Entry("a value of the wrong type", `start => main("high", valve_a)`,
			"type mismatch: expected f64, got str in argument 'limit' of 'main'"),
		Entry("a channel of the wrong type", `start => main(50.0, start)`,
			"type mismatch: expected f64, got bool in argument 'valve' of 'main'"),
		Entry("a missing argument of a stage", `
			stage hold(out chan f64) {
				0.0 -> out
			}
			start => hold()
		`, "missing required argument: parameter 'out' of stage 'hold'"),
		Entry("parentheses on a sequence with no parameters", `
			sequence plain {
				stage idle {
				}
			}
			start => plain()
		`, "invalid transition target: plain has no parameters"),
		Entry("parentheses on a stage with no parameters", `
			stage plain {
			}
			start => plain()
		`, "invalid transition target: plain has no parameters"),
		Entry("an argument for a sequence with no parameters", `
			sequence plain {
				stage idle {
				}
			}
			start => plain(50.0)
		`, "invalid transition target: plain has no parameters"),
		Entry("an argument for a stage with no parameters", `
			stage plain {
			}
			start => plain(50.0)
		`, "invalid transition target: plain has no parameters"),
		Entry("parentheses on a stage of the calling sequence", `
			sequence outer {
				stage first {
					start => second()
				}
				stage second {
				}
			}
		`, "invalid transition target: second has no parameters"),
		Entry("parentheses on a sequence in a routing table entry", `
			func split() {value f64} (high f64, low f64) {
				high = value
				low = value
			}
			sequence plain {
				stage idle {
				}
			}
			pressure -> split() -> {
				high: true => plain(),
				low: true => main(20.0, valve_b)
			}
		`, "invalid transition target: plain has no parameters"),
		Entry("parentheses on a stage in a routing table entry", `
			func split() {value f64} (high f64, low f64) {
				high = value
				low = value
			}
			stage plain {
			}
			pressure -> split() -> {
				high: true => plain(),
				low: true => main(20.0, valve_b)
			}
		`, "invalid transition target: plain has no parameters"),
		Entry("a call as a step", `
			sequence outer {
				main(50.0, valve_a)
			}
		`, "main is not a function"),
	)

	It("Should take a module constant as a value argument", func(bCtx SpecContext) {
		symbols := stl.NewSymbols()
		for i := range channels {
			symbols = append(symbols, &channels[i])
		}
		ast := MustSucceed(parser.Parse(
			"import math\n" + declaration + "start => main(-math.inf, valve_a)",
		))
		ctx := context.NewRoot(bCtx, ast, symbol.NewRoot(nil, symbols))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
	})

	It("Should say how to start a sequence named with no arguments", func(
		bCtx SpecContext,
	) {
		ctx := analyze(bCtx, `start => main`)
		Expect(*ctx.Diagnostics).To(HaveLen(1))
		Expect((*ctx.Diagnostics)[0].Notes).To(HaveLen(1))
		Expect((*ctx.Diagnostics)[0].Notes[0].Message).
			To(Equal("use main()"))
	})

	DescribeTable("Should say how to start one called with no parameter list",
		func(bCtx SpecContext, source string) {
			ctx := analyze(bCtx, source)
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Notes).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Notes[0].Message).
				To(Equal("use plain with no parentheses"))
		},
		Entry("a sequence", `
			sequence plain {
				stage idle {
				}
			}
			start => plain()
		`),
		Entry("a stage", `
			stage plain {
			}
			start => plain()
		`),
	)

	It("Should list the channel arguments that are fixed at compile time", func(
		bCtx SpecContext,
	) {
		ctx := analyze(bCtx, `
			sequence outer {
				out := valve_a
				stage idle {
					start => main(50.0, out)
				}
			}
		`)
		Expect(*ctx.Diagnostics).To(HaveLen(1))
		Expect((*ctx.Diagnostics)[0].Notes).To(HaveLen(1))
		Expect((*ctx.Diagnostics)[0].Notes[0].Message).To(Equal(
			"use a channel, a top-level alias, or a channel parameter",
		))
	})

	It("Should show the signature in a call error", func(bCtx SpecContext) {
		ctx := analyze(bCtx, `start => main(50.0)`)
		Expect(*ctx.Diagnostics).To(HaveLen(1))
		Expect((*ctx.Diagnostics)[0].Notes).To(HaveLen(1))
		Expect((*ctx.Diagnostics)[0].Notes[0].Message).
			To(Equal("signature: main(limit f64, valve chan f64)"))
	})
})

var _ = Describe("BodyNoun", func() {
	It("Should name a sequence and a top-level stage", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
			sequence main(limit f64) {
				stage press {
				}
			}
			stage hold(limit f64) {
			}
		`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		Expect(flow.BodyNoun(MustSucceed(ctx.Scope.Resolve(bCtx, "main")))).
			To(Equal("sequence"))
		Expect(flow.BodyNoun(MustSucceed(ctx.Scope.Resolve(bCtx, "hold")))).
			To(Equal("stage"))
	})
})
