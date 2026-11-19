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
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/sequence"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
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
	{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.Bool())},
	{Name: "abort_btn", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
	{Name: "pressure", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
	{Name: "valve_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
}

// analyzeAndExpectSuccess parses the source, analyzes it, and expects success.
func analyzeAndExpectSuccess(bCtx SpecContext, source string) {
	GinkgoHelper()
	ast := MustSucceed(parser.Parse(source))
	ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
	analyzer.AnalyzeProgram(ctx)
	Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
}

// analyzeAndExpectError parses the source, analyzes it, expects failure, and returns
// the first error message.
func analyzeAndExpectError(bCtx SpecContext, source string) string {
	GinkgoHelper()
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
		DescribeTable(
			"Valid",
			analyzeAndExpectSuccess,
			Entry("variable declared in a sequence body", `
				sequence main {
					counter := 0
					stage s1 {
					}
				}
			`),
			Entry("reassigning a literal variable in a stage", `
				sequence main {
					counter := 0
					stage s1 {
						counter = 1
					}
				}
			`),
			Entry("reassigning a literal variable in a sequence body", `
				sequence main {
					counter := 0
					counter = 1
					stage s1 {
					}
				}
			`),
			Entry("re-expressing a channel-read variable in a sequence body", `
				sequence main {
					r := pressure + 1
					r = pressure + 2
					stage s1 {
					}
				}
			`),
			Entry("re-expressing a channel-read variable in a stage", `
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
			Entry("channel read/write declared in a sequence body", `
				sequence main {
					p := pressure
					stage s1 {
					}
				}
			`),
			Entry(
				"rebinding a channel read/write to a same-type channel in a sequence body",
				`
				sequence main {
					p := pressure
					p = valve_cmd
					stage s1 {
					}
				}
			`,
			),
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

		DescribeTable(
			"Invalid",
			func(bCtx SpecContext, source, expectedError string) {
				Expect(
					analyzeAndExpectError(bCtx, source),
				).To(ContainSubstring(expectedError))
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
			Entry("a top-level variable after a sequence that declares it", `
				sequence main {
					limit := 1.0
				}
				limit := 5.0
			`, "name limit conflicts with existing variable"),
			Entry("a top-level variable after a nested stage that declares it", `
				sequence main {
					stage s1 {
						limit := 1.0
					}
				}
				limit := 5.0
			`, "name limit conflicts with existing variable"),
			Entry("a top-level variable after a top-level stage that declares it", `
				stage s1 {
					limit := 1.0
				}
				limit := 5.0
			`, "name limit conflicts with existing variable"),
			Entry("a sequence variable after a stage that declares it", `
				sequence main {
					stage s1 {
						limit := 1.0
					}
					limit := 5.0
				}
			`, "name limit conflicts with existing variable"),
			Entry("a sequence variable after a nested sequence that declares it", `
				sequence main {
					sequence inner {
						limit := 1.0
					}
					limit := 5.0
				}
			`, "name limit conflicts with existing variable"),
			Entry("a stage variable after a nested sequence that declares it", `
				sequence main {
					stage s1 {
						sequence inner {
							limit := 1.0
						}
						limit := 5.0
					}
				}
			`, "name limit conflicts with existing variable"),
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
			Entry("rebinding a channel read/write to a different-type channel", `
				sequence main {
					p := pressure
					p = start_cmd
					stage s1 {
					}
				}
			`, "type mismatch: cannot rebind 'p'"),
			Entry("rebinding a top-level channel read/write in a stage", `
				p := pressure
				sequence main {
					stage s1 {
						p = valve_cmd
					}
				}
			`, "cannot reassign top-level variable 'p'"),
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
			Entry("compound assignment to a scoped variable is unsupported", `
				sequence main {
					level f64 := 0
					level += 1
					stage s1 {
					}
				}
			`, "compound and indexed assignment"),
			Entry("reassigning a bare channel with '='", `
				sequence main {
					stage s1 {
						pressure = 1.0
					}
				}
			`, "use '<value> -> pressure' to write to 'pressure'"),
			Entry("rebinding a channel read/write to a literal", `
				sequence main {
					p := pressure
					p = 5.0
					stage s1 {
					}
				}
			`, "use '<value> -> p' to write to 'p'"),
			Entry("rebinding a channel read/write to a value variable", `
				sequence main {
					p := pressure
					level f64 := 0
					p = level
					stage s1 {
					}
				}
			`, "use '<value> -> p' to write to 'p'"),
			Entry("rebinding a channel read/write to an undefined name", `
				sequence main {
					p := pressure
					p = mystery
					stage s1 {
					}
				}
			`, "use '<value> -> p' to write to 'p'"),
		)
	})

	Describe("Variables And Aliases In Flows", func() {
		DescribeTable("Valid",
			analyzeAndExpectSuccess,
			Entry("channel read/write drives a transition condition", `
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
			Entry("channel read/write is a flow source", `
				sequence main {
					p := pressure
					stage s1 {
						p -> valve_cmd
					}
				}
			`),
			Entry("rebinding a channel read/write to a same-type channel", `
				sequence main {
					p := pressure
					p = valve_cmd
					stage s1 {
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
				Expect(
					analyzeAndExpectError(bCtx, source),
				).To(ContainSubstring(expectedError))
			},
			Entry("type mismatch feeding an expression into a variable sink", `
				sequence main {
					level f64 := 0
					stage s1 {
						"hello" -> level
					}
				}
			`, "does not match"),
			Entry("writing a flow into a reactive variable", `
				sequence main {
					r := pressure + 1
					stage s1 {
						pressure -> r
					}
				}
			`, "read-only"),
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
		DescribeTable(
			"Valid Top-Level Stages",
			analyzeAndExpectSuccess,
			Entry("named top-level stage", `
				stage main {
					true -> start_cmd
				}
			`),
			Entry("anonymous top-level stage", `
				stage {
					true -> start_cmd
				}
			`),
			Entry("top-level stage with anonymous routing case body", `
				stage main {
					start_cmd -> select() -> {
						true: stage { 1 -> abort_btn },
						false: stage { 1 -> abort_btn }
					}
				}
			`),
			Entry(
				"top-level stage containing a nested sequence with anonymous routing",
				`
				stage main {
					sequence inner {
						start_cmd -> select() -> {
							true: stage { 1 -> abort_btn }
						}
					}
				}
			`,
			),
			Entry("top-level stage with anonymous flow target", `
				stage main {
					start_cmd -> stage { 1 -> abort_btn }
				}
			`),
			Entry(
				"top-level stage containing a nested sequence with anonymous flow target",
				`
				stage main {
					sequence inner {
						start_cmd -> stage { 1 -> abort_btn }
					}
				}
			`,
			),
		)
	})

	Describe("Anonymous Routing Case Bodies", func() {
		DescribeTable("Valid Anonymous Routing",
			analyzeAndExpectSuccess,
			Entry("anonymous stage in routing case body (stage context)", `
				sequence main {
					stage hold {
						start_cmd -> select() -> {
							true: stage { 1 -> abort_btn }
						}
					}
				}
			`),
			Entry("anonymous sequence in routing case body (stage context)", `
				sequence main {
					stage hold {
						start_cmd -> select() -> {
							true: sequence {
								1 -> abort_btn
							}
						}
					}
				}
			`),
			Entry("anonymous routing directly in sequence body", `
				sequence main {
					start_cmd -> select() -> {
						true: stage { 1 -> abort_btn }
					}
				}
			`),
			Entry("anonymous routing inside nested sequence", `
				sequence main {
					stage hold {
						sequence inner {
							start_cmd -> select() -> {
								true: stage { 1 -> abort_btn }
							}
						}
					}
				}
			`),
			Entry("empty anonymous stage body", `
				sequence main {
					stage hold {
						start_cmd -> select() -> {
							true: stage { },
							false: stage { 1 -> abort_btn }
						}
					}
				}
			`),
			Entry("empty anonymous sequence body", `
				sequence main {
					stage hold {
						start_cmd -> select() -> {
							true: sequence { },
							false: stage { 1 -> abort_btn }
						}
					}
				}
			`),
		)

		It(
			"Should give a named anonymous routing case body its code",
			func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
				sequence main {
					stage hold {
						start_cmd -> select() -> {
							true: stage my_stage { 1 -> abort_btn }
						}
					}
				}
			`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(*ctx.Diagnostics).ToNot(BeEmpty())
				Expect((*ctx.Diagnostics)[0].Code).
					To(Equal(codes.NamedFlowBody.Code))
			},
		)

		DescribeTable("Should reject named anonymous routing case bodies",
			func(bCtx SpecContext, source, expectedError string) {
				msg := analyzeAndExpectError(bCtx, source)
				Expect(msg).To(ContainSubstring(expectedError))
			},
			Entry("named anonymous stage", `
				sequence main {
					stage hold {
						start_cmd -> select() -> {
							true: stage my_stage { 1 -> abort_btn }
						}
					}
				}
			`, `a stage or sequence in a flow cannot have a name`),
			Entry("named anonymous sequence", `
				sequence main {
					stage hold {
						start_cmd -> select() -> {
							true: sequence my_seq { 1 -> abort_btn }
						}
					}
				}
			`, `a stage or sequence in a flow cannot have a name`),
		)
	})

	Describe("Anonymous Flow Target Bodies", func() {
		DescribeTable("Valid Anonymous Flow Targets",
			analyzeAndExpectSuccess,
			Entry("anonymous stage as flow target (stage context)", `
				sequence main {
					stage hold {
						start_cmd -> stage { 1 -> abort_btn }
					}
				}
			`),
			Entry("anonymous sequence as flow target (stage context)", `
				sequence main {
					stage hold {
						start_cmd -> sequence {
							1 -> abort_btn
						}
					}
				}
			`),
			Entry("anonymous flow target directly in sequence body", `
				sequence main {
					start_cmd -> stage { 1 -> abort_btn }
				}
			`),
			Entry("anonymous stage flow target at module scope", `
				start_cmd -> stage { 1 -> abort_btn }
			`),
			Entry("anonymous sequence flow target at module scope", `
				start_cmd -> sequence {
					1 -> abort_btn
				}
			`),
			Entry("module-scope anonymous stage with multiple parallel writes", `
				start_cmd -> stage {
					1 -> abort_btn
					true -> start_cmd
				}
			`),
			Entry("module-scope anonymous sequence with multiple steps", `
				start_cmd -> sequence {
					1 -> abort_btn
					true -> start_cmd
				}
			`),
			Entry("anonymous flow target inside nested sequence", `
				sequence main {
					stage hold {
						sequence inner {
							start_cmd -> stage { 1 -> abort_btn }
						}
					}
				}
			`),
			Entry("empty anonymous stage flow target", `
				sequence main {
					stage hold {
						start_cmd -> stage { }
					}
				}
			`),
			Entry("empty anonymous sequence flow target", `
				sequence main {
					stage hold {
						start_cmd -> sequence { }
					}
				}
			`),
		)

		DescribeTable("Should reject named anonymous flow targets",
			func(bCtx SpecContext, source, expectedError string) {
				msg := analyzeAndExpectError(bCtx, source)
				Expect(msg).To(ContainSubstring(expectedError))
			},
			Entry("named anonymous stage", `
				sequence main {
					stage hold {
						start_cmd -> stage my_stage { 1 -> abort_btn }
					}
				}
			`, `a stage or sequence in a flow cannot have a name`),
			Entry("named anonymous sequence", `
				sequence main {
					stage hold {
						start_cmd -> sequence my_seq { 1 -> abort_btn }
					}
				}
			`, `a stage or sequence in a flow cannot have a name`),
		)
	})

	Describe("RejectReactiveAssignment", func() {
		It("Should point the assignment at a flow write", func() {
			stmt := MustSucceed(parser.ParseStatement("valve = 1"))
			var diag diagnostics.Diagnostics
			sequence.RejectReactiveAssignment(&diag, stmt.Assignment())
			Expect(diag).To(HaveLen(1))
			Expect(diag[0].Message).
				To(Equal("invalid assignment: use '<value> -> valve' to write to 'valve'"))
		})
	})
})

var _ = Describe("Stage reassignment types", func() {
	DescribeTable("Should give a reassigned expression the variable's type",
		func(bCtx SpecContext, decl, value string, expected types.Type) {
			ast := MustSucceed(parser.Parse("sequence main {\n" + decl +
				"\nstage a {\nv = " + value + "\n}\n}\ngo => main"))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, symbol.Symbol{
				Name: "go",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.Bool()),
				ID:   1,
			}))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			var outputs []types.Type
			for _, sym := range ctx.Scope.Root().Children() {
				if out, ok := sym.Type.Outputs.Get(ir.DefaultOutputParam); ok &&
					sym.Kind == symbol.KindFunction {
					outputs = append(
						outputs,
						ctx.Constraints.ApplySubstitutions(out.Type),
					)
				}
			}
			Expect(outputs).To(Equal([]types.Type{expected}))
		},
		Entry("an i32 variable", "v i32 := 0", "1 + 2", types.I32()),
		Entry("an f32 variable", "v f32 := 0.0", "1.5 * 2.0", types.F32()),
		Entry("an f32 stateful variable", "v f32 $= 0.0", "1.5 * 2.0", types.F32()),
	)

	It("Should give a reassigned reactive variable's expression its value type", func(
		bCtx SpecContext,
	) {
		ast := MustSucceed(
			parser.Parse("sequence main {\nv := sensor + 1.0\nstage a {\n" +
				"v = 1.0 + 2.0\n}\n}\ngo => main"),
		)
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, symbol.Symbol{
			Name: "go", Kind: symbol.KindChannel, Type: types.Chan(types.Bool()), ID: 1,
		}, symbol.Symbol{
			Name: "sensor",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   2,
		}))
		analyzer.AnalyzeProgram(ctx)
		Expect(*ctx.Diagnostics).To(HaveLen(1))
		Expect(
			(*ctx.Diagnostics)[0].Message,
		).To(ContainSubstring("from a constant value"))
		for _, sym := range ctx.Scope.Root().Children() {
			if out, ok := sym.Type.Outputs.Get(ir.DefaultOutputParam); ok &&
				sym.Kind == symbol.KindFunction {
				Expect(ctx.Constraints.ApplySubstitutions(out.Type).Kind).
					ToNot(Equal(types.KindChan))
			}
		}
	})
})

var _ = Describe("Parameters", func() {
	DescribeTable("Valid",
		analyzeAndExpectSuccess,
		Entry("a sequence with a value and a channel parameter", `
			sequence main(limit f64, valve chan f64) {
				stage press {
					1.0 -> valve
					pressure > limit => next
				}
				stage vent {
					0.0 -> valve
				}
			}
		`),
		Entry("a top-level stage with a channel parameter", `
			stage hold(valve chan f64) {
				0.0 -> valve
			}
		`),
		Entry("a parameter with a default", `
			sequence main(valve chan f64, limit f64 = 50.0) {
				stage press {
					pressure > limit => next
				}
				stage vent {
					0.0 -> valve
				}
			}
		`),
		Entry("a channel parameter as a flow source", `
			sequence main(sensor chan f64) {
				stage copy {
					sensor -> valve_cmd
				}
			}
		`),
		Entry("a variable computed from a value parameter", `
			sequence main(limit f64) {
				high := limit * 2.0
				stage press {
					pressure > high => next
				}
				stage vent {
				}
			}
		`),
		Entry("a variable that copies a value parameter", `
			sequence main(limit f64) {
				copy := limit
				stage press {
					pressure > copy => next
				}
				stage vent {
				}
			}
		`),
		Entry("a value parameter as a function input", `
			sequence main(note str) {
				stage hold {
					start_cmd -> log(message=note)
				}
			}
		`),
		Entry("a parameter as the source of a function trigger", `
			func double() {v f64} f64 {
				return v * 2.0
			}
			sequence main(limit f64, sensor chan f64) {
				stage copy {
					limit -> double() -> valve_cmd
					sensor -> double() -> valve_cmd
				}
			}
		`),
		Entry("a parameter with the name of a channel", `
			sequence main(pressure f64) {
				stage hold {
					pressure -> valve_cmd
				}
			}
		`),
	)

	DescribeTable("Invalid",
		func(bCtx SpecContext, source, expectedError string) {
			Expect(
				analyzeAndExpectError(bCtx, source),
			).To(ContainSubstring(expectedError))
		},
		Entry("a parameter list on a stage in a sequence", `
			sequence main {
				stage press(limit f64) {
				}
			}
		`, "invalid stage or sequence declaration: only a top-level stage or sequence can have parameters"),
		Entry("a parameter list on a sequence in a sequence", `
			sequence main {
				sequence inner(limit f64) {
					0.0 -> valve_cmd
				}
			}
		`, "invalid stage or sequence declaration: only a top-level stage or sequence can have parameters"),
		Entry("a parameter list on an anonymous stage", `
			start_cmd => stage(limit f64) {
				0.0 -> valve_cmd
			}
		`, "invalid stage or sequence declaration: only a top-level stage or sequence can have parameters"),
		Entry("a parameter list on a sequence with no name", `
			sequence(limit f64) {
				0.0 -> valve_cmd
			}
		`, "invalid stage or sequence declaration: a sequence with parameters needs a name"),
		Entry("a parameter list on a stage with no name", `
			stage(limit f64) {
				0.0 -> valve_cmd
			}
		`, "invalid stage or sequence declaration: a stage with parameters needs a name"),
		Entry("an empty parameter list on a sequence", `
			sequence main() {
				stage press {
				}
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("an empty parameter list on a stage", `
			stage hold() {
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("an empty parameter list on a stage in a sequence", `
			sequence main {
				stage press() {
				}
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("an empty parameter list on a sequence in a sequence", `
			sequence main {
				sequence inner() {
					0.0 -> valve_cmd
				}
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("an empty parameter list on an anonymous stage", `
			start_cmd => stage() {
				0.0 -> valve_cmd
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("an empty parameter list on a sequence with no name", `
			sequence() {
				0.0 -> valve_cmd
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("spaces in the parentheses of a sequence", `
			sequence main(    ) {
				stage press {
				}
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("spaces in the parentheses of a stage", `
			stage hold(    ) {
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("a new line in the parentheses of a sequence", `
			sequence main(
			) {
				stage press {
				}
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("a new line in the parentheses of a stage", `
			stage hold(
			) {
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("a space before the empty parentheses of a sequence", `
			sequence main () {
				stage press {
				}
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("a space before the empty parentheses of a stage", `
			stage hold () {
			}
		`, "invalid stage or sequence declaration: a parameter list cannot be empty"),
		Entry("a default of the wrong type", `
			sequence main(limit f64 = "high") {
				stage press {
				}
			}
		`, "invalid default value for input parameter limit"),
		Entry("a default on a channel parameter of a sequence", `
			sequence main(valve chan f64 = 0) {
				stage press {
				}
			}
		`, "invalid stage or sequence declaration: channel parameter 'valve' "+
			"cannot have a default"),
		Entry("a default on a channel parameter of a stage", `
			stage hold(valve chan f64 = 0) {
				0.0 -> valve
			}
		`, "invalid stage or sequence declaration: channel parameter 'valve' "+
			"cannot have a default"),
		Entry("a bool default on a channel parameter of a sequence", `
			sequence main(valve chan bool = false) {
				stage press {
				}
			}
		`, "invalid stage or sequence declaration: channel parameter 'valve' "+
			"cannot have a default"),
		Entry("a bool default on a channel parameter of a stage", `
			stage hold(valve chan bool = false) {
				true -> valve
			}
		`, "invalid stage or sequence declaration: channel parameter 'valve' "+
			"cannot have a default"),
		Entry("a default on a channel parameter after a value parameter", `
			sequence main(limit f64 = 1.0, valve chan u8 = 1) {
				stage press {
				}
			}
		`, "invalid stage or sequence declaration: channel parameter 'valve' "+
			"cannot have a default"),
		Entry("a required parameter after an optional one", `
			sequence main(limit f64 = 1.0, valve chan f64) {
				stage press {
				}
			}
		`, "required input parameter valve cannot follow optional input parameters"),
		Entry("an assignment to a value parameter", `
			sequence main(limit f64) {
				stage press {
					limit = 1.0
				}
			}
		`, "invalid assignment: cannot reassign parameter 'limit'"),
		Entry("a flow into a value parameter", `
			sequence main(limit f64) {
				stage press {
					1.0 -> limit
				}
			}
		`, "invalid assignment: cannot reassign parameter 'limit'"),
		Entry("an assignment to a channel parameter", `
			sequence main(valve chan f64) {
				stage press {
					valve = 1.0
				}
			}
		`, "use '<value> -> valve' to write to 'valve'"),
		Entry("a flow of the wrong type into a channel parameter", `
			sequence main(valve chan f64) {
				stage press {
					start_cmd -> valve
				}
			}
		`, "does not match channel valve value type f64"),
		Entry("an alias of a channel parameter", `
			sequence main(valve chan f64) {
				out := valve
				stage press {
					1.0 -> out
				}
			}
		`, "invalid assignment: a channel parameter cannot have an alias, use 'valve' directly"),
		Entry("two parameters with one name", `
			sequence main(limit f64, limit f64) {
				stage press {
				}
			}
		`, "name limit conflicts with existing input parameter"),
		Entry("a parameter with the name of a top-level variable above it", `
			limit := 1.0
			sequence main(limit f64) {
				stage press {
				}
			}
		`, "name limit conflicts with existing variable"),
		Entry("a parameter with the name of a function", `
			func limit() f64 {
				return 1.0
			}
			sequence main(limit f64) {
				stage press {
				}
			}
		`, "name limit conflicts with existing function"),
		Entry("a parameter with the name of another sequence", `
			sequence main(abort f64) {
				stage press {
				}
			}
			sequence abort {
				stage safed {
				}
			}
		`, "name abort conflicts with existing sequence"),
		Entry("a variable with the name of a parameter", `
			sequence main(limit f64) {
				limit := 1.0
				stage press {
				}
			}
		`, "name limit conflicts with existing input parameter"),
		Entry("a parameter with the name of a stage in its sequence", `
			sequence main(press f64) {
				stage press {
				}
			}
		`, "name press conflicts with existing stage"),
		Entry("the bare name of its own sequence", `
			sequence main(limit f64) {
				stage press {
					start_cmd => main
				}
			}
		`, "invalid transition target: main has parameters"),
		Entry("a member of its own sequence by qualified name", `
			sequence main(limit f64) {
				stage press {
					start_cmd => main.vent
				}
				stage vent {
				}
			}
		`, "undefined symbol: main.vent"),
		Entry("a parameter read by qualified name from outside", `
			sequence main(limit f64) {
				stage press {
				}
			}
			main.limit -> valve_cmd
		`, "undefined symbol: main.limit"),
	)

	It("Should give the sequence type its inputs", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
			sequence main(valve chan f64, limit f64 = 50.0) {
				stage press {
				}
			}
		`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
		sequence.CollectDeclarations(ctx)
		main := MustSucceed(ctx.Scope.Resolve(bCtx, "main"))
		Expect(main.Type.Inputs).To(Equal(types.Params{
			{Name: "valve", Type: types.Chan(types.F64())},
			{Name: "limit", Type: types.F64(), Value: 50.0},
		}))
	})
})
