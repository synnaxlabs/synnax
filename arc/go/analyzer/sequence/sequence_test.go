// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sequence_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var resolver = []symbol.Symbol{
	{
		Name: "interval",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "duration", Type: types.TimeSpan()}},
			Outputs: types.Params{{Name: "output", Type: types.U8()}},
		}),
	},
	{
		Name: "wait",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "duration", Type: types.TimeSpan()}},
			Outputs: types.Params{{Name: "output", Type: types.U8()}},
		}),
	},
	{
		Name: "log",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "message", Type: types.String()}},
		}),
	},
	{
		Name: "control",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "target", Type: types.F64()}},
		}),
	},
	{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
	{Name: "abort_btn", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
	{Name: "pressure", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
	{Name: "valve_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
}

// analyzeAndExpectSuccess parses the source, analyzes it, and expects success.
func analyzeAndExpectSuccess(bCtx SpecContext, source string) {
	ast := MustSucceed(parser.Parse(source))
	ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
	analyzer.AnalyzeProgram(ctx)
	Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
}

// analyzeAndExpectError parses the source, analyzes it, expects failure, and returns the first error message.
func analyzeAndExpectError(bCtx SpecContext, source string) string {
	ast := MustSucceed(parser.Parse(source))
	ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
	analyzer.AnalyzeProgram(ctx)
	Expect(ctx.Diagnostics.Ok()).To(BeFalse())
	Expect(*ctx.Diagnostics).ToNot(BeEmpty())
	return (*ctx.Diagnostics)[0].Message
}

var _ = Describe("Sequence Analyzer", func() {
	DescribeTable("Valid Sequences",
		analyzeAndExpectSuccess,
		Entry("single stage sequence", `
			sequence main {
				stage start {
				}
			}
		`),
		Entry("stage name same as sibling sequence name (different scopes)", `
			sequence main {
				stage abort {
				}
			}
			sequence abort {
				stage safed {
				}
			}
		`),
		Entry("multiple stages in sequence", `
			sequence main {
				stage step1 {
				}
				stage step2 {
				}
				stage step3 {
				}
			}
		`),
		Entry("multiple sequences", `
			sequence main {
				stage start {
				}
			}
			sequence abort {
				stage safed {
				}
			}
		`),
	)

	Describe("Transitions", func() {
		DescribeTable("Valid Transitions",
			analyzeAndExpectSuccess,
			Entry("next transition", `
				sequence main {
					stage step1 {
						1 => next
					}
					stage step2 {
					}
				}
			`),
			Entry("named stage transitions", `
				sequence main {
					stage step1 {
						1 => step2
					}
					stage step2 {
						1 => step1
					}
				}
			`),
			Entry("cross-sequence transitions", `
				sequence main {
					stage step1 {
						1 => abort
					}
				}
				sequence abort {
					stage safed {
					}
				}
			`),
		)

		DescribeTable("Invalid Transitions",
			func(bCtx SpecContext, source, expectedError string) {
				msg := analyzeAndExpectError(bCtx, source)
				Expect(msg).To(Equal(expectedError))
			},
			Entry("unknown stage name", `
				sequence main {
					stage step1 {
						1 => unknown_stage
					}
				}
			`, "undefined symbol: unknown_stage"),
			Entry("unknown sequence name", `
				sequence main {
					stage step1 {
						1 => unknown_sequence
					}
				}
			`, "undefined symbol: unknown_sequence"),
		)
	})

	Describe("Name Collisions", func() {
		DescribeTable("Should detect name conflicts",
			func(bCtx SpecContext, source, expectedError string) {
				msg := analyzeAndExpectError(bCtx, source)
				Expect(msg).To(ContainSubstring(expectedError))
			},
			Entry("duplicate sequence names", `
				sequence main {
					stage step1 {
					}
				}
				sequence main {
					stage step1 {
					}
				}
			`, "conflicts with existing sequence"),
			Entry("duplicate stage names within sequence", `
				sequence main {
					stage step1 {
					}
					stage step1 {
					}
				}
			`, "conflicts with existing stage"),
		)
	})

	Describe("Scoped Variables", func() {
		DescribeTable("Valid",
			analyzeAndExpectSuccess,
			Entry("variable declared in a sequence body", `
				sequence main {
					counter := 0
					stage s1 {
					}
				}
			`),
			Entry("reassigning a constant variable in a stage", `
				sequence main {
					counter := 0
					stage s1 {
						counter = 1
					}
				}
			`),
			Entry("reassigning a constant variable in a sequence body", `
				sequence main {
					counter := 0
					counter = 1
					stage s1 {
					}
				}
			`),
			Entry("re-expressing a reactive variable in a sequence body", `
				sequence main {
					r := pressure + 1
					r = pressure + 2
					stage s1 {
					}
				}
			`),
			Entry("re-expressing a reactive variable in a stage", `
				sequence main {
					r := pressure + 1
					stage s1 {
						r = pressure + 2
					}
				}
			`),
			Entry("nested stage reads a variable from the enclosing sequence", `
				sequence main {
					counter := 0
					stage s1 {
						doubled := counter
					}
				}
			`),
			Entry("channel alias declared in a sequence body", `
				sequence main {
					p := pressure
					stage s1 {
					}
				}
			`),
			Entry("rebinding a channel alias to a same-type channel in a sequence body", `
				sequence main {
					p := pressure
					p = valve_cmd
					stage s1 {
					}
				}
			`),
			Entry("nested sequence reads a variable from the enclosing sequence", `
				sequence main {
					counter := 0
					sequence inner {
						doubled := counter
						stage s1 {
						}
					}
				}
			`),
			Entry("a sequence using a top-level variable declared before it", `
				greeting := "hi"
				sequence main {
					message := greeting
					stage s1 {
					}
				}
			`),
		)

		DescribeTable("Invalid",
			func(bCtx SpecContext, source, expectedError string) {
				Expect(analyzeAndExpectError(bCtx, source)).To(ContainSubstring(expectedError))
			},
			Entry("shadowing a variable inherited from the enclosing sequence", `
				sequence main {
					counter := 0
					stage s1 {
						counter := 1
					}
				}
			`, "conflicts with existing variable"),
			Entry("referencing a variable declared in a sibling stage", `
				sequence main {
					stage s1 {
						x := 1
					}
					stage s2 {
						y := x
					}
				}
			`, "undefined symbol: x"),
			Entry("using a variable before it is declared in the same scope", `
				sequence main {
					stage s1 {
						a := b
						b := 0
					}
				}
			`, "undefined symbol: b"),
			Entry("a stage using a sequence variable declared after the stage", `
				sequence main {
					stage s1 {
						a := counter
					}
					counter := 0
				}
			`, "undefined symbol: counter"),
			Entry("shadowing an inherited variable in a nested sequence", `
				sequence main {
					counter := 0
					sequence inner {
						counter := 1
						stage s1 {
						}
					}
				}
			`, "conflicts with existing variable"),
			Entry("referencing a variable declared in a sibling sequence", `
				sequence a {
					x := 1
					stage s1 {
					}
				}
				sequence b {
					y := x
					stage s1 {
					}
				}
			`, "undefined symbol: x"),
			Entry("rebinding a channel alias to a different-type channel", `
				sequence main {
					p := pressure
					p = start_cmd
					stage s1 {
					}
				}
			`, "cannot rebind alias p"),
			Entry("reassigning a variable that was never declared", `
				sequence main {
					stage s1 {
						missing = 1
					}
				}
			`, "undefined symbol: missing"),
			Entry("using a variable before it is declared in a sequence body", `
				sequence main {
					a := b
					b := 0
					stage s1 {
					}
				}
			`, "undefined symbol: b"),
			Entry("a nested sequence using an enclosing variable declared after it", `
				sequence main {
					sequence inner {
						a := counter
						stage s1 {
						}
					}
					counter := 0
				}
			`, "undefined symbol: counter"),
			Entry("a sequence using a top-level variable declared after it", `
				sequence main {
					message := greeting
					stage s1 {
					}
				}
				greeting := "hi"
			`, "undefined symbol: greeting"),
		)
	})

	Describe("Variables And Aliases In Flows", func() {
		DescribeTable("Valid",
			analyzeAndExpectSuccess,
			Entry("channel alias drives a transition condition", `
				sequence main {
					p := pressure
					stage s1 {
						p > 0.0 => s2
					}
					stage s2 {
					}
				}
			`),
			Entry("value variable is the sink of a flow", `
				sequence main {
					level f64 := 0
					stage s1 {
						pressure -> level
					}
				}
			`),
			Entry("value variable is a standalone flow source", `
				sequence main {
					level f64 := 0
					stage s1 {
						level -> valve_cmd
					}
				}
			`),
			Entry("channel alias is a flow source", `
				sequence main {
					p := pressure
					stage s1 {
						p -> valve_cmd
					}
				}
			`),
			Entry("value variable read in a transition condition", `
				sequence main {
					threshold f64 := 10
					stage s1 {
						pressure > threshold => s2
					}
					stage s2 {
					}
				}
			`),
		)

		DescribeTable("Invalid",
			func(bCtx SpecContext, source, expectedError string) {
				Expect(analyzeAndExpectError(bCtx, source)).To(ContainSubstring(expectedError))
			},
			Entry("type mismatch feeding an expression into a variable sink", `
				sequence main {
					level f64 := 0
					stage s1 {
						"hello" -> level
					}
				}
			`, "does not match"),
		)
	})

	Describe("Top-Level Transitions", func() {
		It("Should validate top-level entry points", func(bCtx SpecContext) {
			analyzeAndExpectSuccess(bCtx, `
				start_cmd => main
				sequence main {
					stage step1 {
					}
				}
			`)
		})

		It("Should error when target sequence doesn't exist", func(bCtx SpecContext) {
			msg := analyzeAndExpectError(bCtx, `
				start_cmd => unknown_sequence
			`)
			Expect(msg).To(Equal("undefined symbol: unknown_sequence"))
		})
	})

	Describe("Top-Level Stages", func() {
		DescribeTable("Valid Top-Level Stages",
			analyzeAndExpectSuccess,
			Entry("named top-level stage", `
				stage main {
					1 -> start_cmd
				}
			`),
			Entry("anonymous top-level stage", `
				stage {
					1 -> start_cmd
				}
			`),
			Entry("top-level stage with inline routing case body", `
				stage main {
					start_cmd -> select{} -> {
						true: stage { 1 -> abort_btn },
						false: stage { 1 -> abort_btn }
					}
				}
			`),
			Entry("top-level stage containing a nested sequence with inline routing", `
				stage main {
					sequence inner {
						start_cmd -> select{} -> {
							true: stage { 1 -> abort_btn }
						}
					}
				}
			`),
			Entry("top-level stage with inline flow target", `
				stage main {
					start_cmd -> stage { 1 -> abort_btn }
				}
			`),
			Entry("top-level stage containing a nested sequence with inline flow target", `
				stage main {
					sequence inner {
						start_cmd -> stage { 1 -> abort_btn }
					}
				}
			`),
		)
	})

	Describe("Inline Routing Case Bodies", func() {
		DescribeTable("Valid Inline Routing",
			analyzeAndExpectSuccess,
			Entry("inline stage in routing case body (stage context)", `
				sequence main {
					stage hold {
						start_cmd -> select{} -> {
							true: stage { 1 -> abort_btn }
						}
					}
				}
			`),
			Entry("inline sequence in routing case body (stage context)", `
				sequence main {
					stage hold {
						start_cmd -> select{} -> {
							true: sequence {
								1 -> abort_btn
							}
						}
					}
				}
			`),
			Entry("inline routing directly in sequence body", `
				sequence main {
					start_cmd -> select{} -> {
						true: stage { 1 -> abort_btn }
					}
				}
			`),
			Entry("inline routing inside nested sequence", `
				sequence main {
					stage hold {
						sequence inner {
							start_cmd -> select{} -> {
								true: stage { 1 -> abort_btn }
							}
						}
					}
				}
			`),
			Entry("empty inline stage body", `
				sequence main {
					stage hold {
						start_cmd -> select{} -> {
							true: stage { },
							false: stage { 1 -> abort_btn }
						}
					}
				}
			`),
			Entry("empty inline sequence body", `
				sequence main {
					stage hold {
						start_cmd -> select{} -> {
							true: sequence { },
							false: stage { 1 -> abort_btn }
						}
					}
				}
			`),
		)

		DescribeTable("Should reject named inline routing case bodies",
			func(bCtx SpecContext, source, expectedError string) {
				msg := analyzeAndExpectError(bCtx, source)
				Expect(msg).To(ContainSubstring(expectedError))
			},
			Entry("named inline stage", `
				sequence main {
					stage hold {
						start_cmd -> select{} -> {
							true: stage my_stage { 1 -> abort_btn }
						}
					}
				}
			`, `inline routing case body stages must be anonymous; remove name "my_stage"`),
			Entry("named inline sequence", `
				sequence main {
					stage hold {
						start_cmd -> select{} -> {
							true: sequence my_seq { 1 -> abort_btn }
						}
					}
				}
			`, `inline routing case body sequences must be anonymous; remove name "my_seq"`),
		)
	})

	Describe("Inline Flow Target Bodies", func() {
		DescribeTable("Valid Inline Flow Targets",
			analyzeAndExpectSuccess,
			Entry("inline stage as flow target (stage context)", `
				sequence main {
					stage hold {
						start_cmd -> stage { 1 -> abort_btn }
					}
				}
			`),
			Entry("inline sequence as flow target (stage context)", `
				sequence main {
					stage hold {
						start_cmd -> sequence {
							1 -> abort_btn
						}
					}
				}
			`),
			Entry("inline flow target directly in sequence body", `
				sequence main {
					start_cmd -> stage { 1 -> abort_btn }
				}
			`),
			Entry("inline stage flow target at module scope", `
				start_cmd -> stage { 1 -> abort_btn }
			`),
			Entry("inline sequence flow target at module scope", `
				start_cmd -> sequence {
					1 -> abort_btn
				}
			`),
			Entry("module-scope inline stage with multiple parallel writes", `
				start_cmd -> stage {
					1 -> abort_btn
					1 -> start_cmd
				}
			`),
			Entry("module-scope inline sequence with multiple steps", `
				start_cmd -> sequence {
					1 -> abort_btn
					1 -> start_cmd
				}
			`),
			Entry("inline flow target inside nested sequence", `
				sequence main {
					stage hold {
						sequence inner {
							start_cmd -> stage { 1 -> abort_btn }
						}
					}
				}
			`),
			Entry("empty inline stage flow target", `
				sequence main {
					stage hold {
						start_cmd -> stage { }
					}
				}
			`),
			Entry("empty inline sequence flow target", `
				sequence main {
					stage hold {
						start_cmd -> sequence { }
					}
				}
			`),
		)

		DescribeTable("Should reject named inline flow targets",
			func(bCtx SpecContext, source, expectedError string) {
				msg := analyzeAndExpectError(bCtx, source)
				Expect(msg).To(ContainSubstring(expectedError))
			},
			Entry("named inline stage", `
				sequence main {
					stage hold {
						start_cmd -> stage my_stage { 1 -> abort_btn }
					}
				}
			`, `inline routing case body stages must be anonymous; remove name "my_stage"`),
			Entry("named inline sequence", `
				sequence main {
					stage hold {
						start_cmd -> sequence my_seq { 1 -> abort_btn }
					}
				}
			`, `inline routing case body sequences must be anonymous; remove name "my_seq"`),
		)
	})
})
