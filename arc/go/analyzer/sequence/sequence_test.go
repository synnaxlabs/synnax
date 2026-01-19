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
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var resolver = symbol.MapResolver{
	"interval": symbol.Symbol{
		Name: "interval",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config:  types.Params{{Name: "duration", Type: types.TimeSpan()}},
			Outputs: types.Params{{Name: "output", Type: types.U8()}},
		}),
	},
	"wait": symbol.Symbol{
		Name: "wait",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config:  types.Params{{Name: "duration", Type: types.TimeSpan()}},
			Outputs: types.Params{{Name: "output", Type: types.U8()}},
		}),
	},
	"log": symbol.Symbol{
		Name: "log",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{{Name: "message", Type: types.String()}},
		}),
	},
	"control": symbol.Symbol{
		Name: "control",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{{Name: "target", Type: types.F64()}},
		}),
	},
	"start_cmd": symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
	"abort_btn": symbol.Symbol{Name: "abort_btn", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
	"pressure":  symbol.Symbol{Name: "pressure", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
	"valve_cmd": symbol.Symbol{Name: "valve_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
}

// analyzeAndExpectSuccess parses the source, analyzes it, and expects success.
func analyzeAndExpectSuccess(source string) {
	ast := MustSucceed(parser.Parse(source))
	ctx := context.CreateRoot(bCtx, ast, resolver)
	Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
}

// analyzeAndExpectError parses the source, analyzes it, expects failure, and returns the error message.
func analyzeAndExpectError(source string) string {
	ast := MustSucceed(parser.Parse(source))
	ctx := context.CreateRoot(bCtx, ast, resolver)
	Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
	Expect(*ctx.Diagnostics).To(HaveLen(1))
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
			func(source, expectedError string) {
				msg := analyzeAndExpectError(source)
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
			func(source, expectedError string) {
				msg := analyzeAndExpectError(source)
				Expect(msg).To(ContainSubstring(expectedError))
			},
			Entry("stage name conflicts with sequence name", `
				sequence main {
					stage abort {
					}
				}
				sequence abort {
					stage safed {
					}
				}
			`, "conflicts with existing symbol"),
			Entry("duplicate sequence names", `
				sequence main {
					stage step1 {
					}
				}
				sequence main {
					stage step1 {
					}
				}
			`, "conflicts with existing symbol"),
			Entry("duplicate stage names within sequence", `
				sequence main {
					stage step1 {
					}
					stage step1 {
					}
				}
			`, "conflicts with existing symbol"),
		)
	})

	Describe("Top-Level Transitions", func() {
		It("Should validate top-level entry points", func() {
			analyzeAndExpectSuccess(`
				start_cmd => main
				sequence main {
					stage step1 {
					}
				}
			`)
		})

		It("Should error when target sequence doesn't exist", func() {
			msg := analyzeAndExpectError(`
				start_cmd => unknown_sequence
			`)
			Expect(msg).To(Equal("undefined symbol: unknown_sequence"))
		})
	})
})
