// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/diagnostics"
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
			Config: types.Params{
				{Name: "duration", Type: types.TimeSpan()},
			},
			Outputs: types.Params{
				{Name: "output", Type: types.U8()},
			},
		}),
	},
	"wait": symbol.Symbol{
		Name: "wait",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{
				{Name: "duration", Type: types.TimeSpan()},
			},
			Outputs: types.Params{
				{Name: "output", Type: types.U8()},
			},
		}),
	},
	"log": symbol.Symbol{
		Name: "log",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{
				{Name: "message", Type: types.String()},
			},
		}),
	},
	"control": symbol.Symbol{
		Name: "control",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{
				{Name: "target", Type: types.F64()},
			},
		}),
	},
	"start_cmd": symbol.Symbol{
		Name: "start_cmd",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.U8()),
	},
	"abort_btn": symbol.Symbol{
		Name: "abort_btn",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.U8()),
	},
	"pressure": symbol.Symbol{
		Name: "pressure",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	"valve_cmd": symbol.Symbol{
		Name: "valve_cmd",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
}

var _ = Describe("Sequence Analyzer", func() {
	Describe("Simple Sequences", func() {
		It("Should analyze a simple sequence with one stage", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage start {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should analyze a sequence with multiple stages", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
					}
					stage step2 {
					}
					stage step3 {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should analyze multiple sequences", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage start {
					}
				}
				sequence abort {
					stage safed {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Transitions", func() {
		It("Should validate next transitions", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
						1 => next
					}
					stage step2 {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should error when next is used in the last stage", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
						1 => next
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("cannot be used in the last stage"))
		})

		It("Should validate stage name transitions", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
						1 => step2
					}
					stage step2 {
						1 => step1
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should error on unknown stage name", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
						1 => unknown_stage
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("unknown transition target"))
		})

		It("Should validate cross-sequence transitions", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
						1 => abort
					}
				}
				sequence abort {
					stage safed {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should error on unknown sequence name", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
						1 => unknown_sequence
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("unknown transition target"))
		})
	})

	Describe("Name Collisions", func() {
		It("Should error when stage name conflicts with sequence name", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage abort {
					}
				}
				sequence abort {
					stage safed {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("conflicts with sequence name"))
		})

		It("Should error on duplicate sequence names", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
					}
				}
				sequence main {
					stage step1 {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("conflicts with existing symbol"))
		})

		It("Should error on duplicate stage names within a sequence", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
					}
					stage step1 {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("conflicts with existing symbol"))
		})
	})

	Describe("Top-Level Transitions", func() {
		It("Should validate top-level entry points", func() {
			ast := MustSucceed(parser.Parse(`
				start_cmd => main
				sequence main {
					stage step1 {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should error when target is not a sequence", func() {
			ast := MustSucceed(parser.Parse(`
				start_cmd => control
				sequence main {
					stage step1 {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("not a sequence"))
		})

		It("Should error when target sequence doesn't exist", func() {
			ast := MustSucceed(parser.Parse(`
				start_cmd => unknown_sequence
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("unknown sequence"))
		})
	})

	Describe("Safety Warnings", func() {
		It("Should warn when non-abort transition appears before abort", func() {
			ast := MustSucceed(parser.Parse(`
				sequence main {
					stage step1 {
						1 => next,
						1 => abort
					}
					stage step2 {
					}
				}
				sequence abort {
					stage safed {
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			// Should succeed but with a warning
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			// Check for warning (not error)
			hasWarning := false
			for _, d := range *ctx.Diagnostics {
				if d.Severity == diagnostics.Warning {
					hasWarning = true
					break
				}
			}
			Expect(hasWarning).To(BeTrue(), "expected warning about abort ordering")
		})
	})
})
