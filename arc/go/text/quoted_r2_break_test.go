// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	xdiagnostics "github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Quoted channel names (round 2 break)", func() {
	channel := func(name string, t types.Type, id int) symbol.Symbol {
		return symbol.Symbol{Name: name, Kind: symbol.KindChannel, Type: t, ID: id}
	}
	channels := []symbol.Symbol{
		channel("out", types.Chan(types.F64()), 20),
		channel("outx", types.Chan(types.F64()), 21),
		channel("trig", types.Chan(types.U8()), 30),
		channel("pt_01", types.Chan(types.F64()), 60),
		channel("log", types.Chan(types.String()), 80),
		channel("x", types.Chan(types.F64()), 2024),
		channel("valve-cmd", types.Chan(types.F64()), 150),
	}
	analyze := func(ctx SpecContext, source string) (ir.IR, *xdiagnostics.Diagnostics) {
		GinkgoHelper()
		return text.Analyze(
			ctx,
			MustSucceed(text.Parse(text.Text{Raw: source})),
			NewRoot(nil, channels...),
		)
	}
	onKeys := func(inter ir.IR) []any {
		var keys []any
		for _, n := range inter.Nodes {
			if n.Type != "on" {
				continue
			}
			for _, in := range n.Inputs {
				if in.Name == "channel" {
					keys = append(keys, in.Value)
				}
			}
		}
		return keys
	}

	DescribeTable("R2-T1 compiles v0.62.0 text with a string in a placeholder",
		func(ctx SpecContext, source string) {
			inter, diagnostics := analyze(ctx, source)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			MustSucceed(text.Compile(ctx, inter))
		},
		Entry("an apostrophe", "func f() {\n\tlog = f`{\"it's\"}`\n}\ntrig -> f()"),
	)

	It("R2-T1b keeps the spec after a string that holds one bracket", func(
		ctx SpecContext,
	) {
		inter, diagnostics := analyze(
			ctx,
			"func f() {\n\tlog = f`{\"[\":5s}`\n}\ntrig -> f()",
		)
		Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		var placeholders []parser.FmtSegment
		for _, segments := range inter.FmtStrings {
			for _, s := range segments {
				if s.Expr != nil {
					placeholders = append(placeholders, s)
				}
			}
		}
		Expect(placeholders).To(HaveLen(1))
		Expect(placeholders[0].Text).To(Equal(`"["`))
		Expect(placeholders[0].Spec).To(Equal("5s"))
	})

	DescribeTable("R2-T2 triggers a flow expression on the channel its callee reads",
		func(ctx SpecContext, source string) {
			inter, diagnostics := analyze(ctx, source)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			MustSucceed(text.Compile(ctx, inter))
			Expect(onKeys(inter)).To(ConsistOf(uint32(60)))
		},
		Entry("a direct read (control)",
			"func f() f64 {\n\treturn 'pt_01'\n}\nf() * 2.0 -> 'out'"),
		Entry("through a local alias whose name no channel holds",
			"func f() f64 {\n\tp := 'pt_01'\n\treturn p\n}\nf() * 2.0 -> 'out'"),
		Entry("through a local alias named like another channel",
			"func f() f64 {\n\tx := 'pt_01'\n\treturn x\n}\nf() * 2.0 -> 'out'"),
	)

	It("R2-T3 triggers a stage condition on the channel its callee reads", func(
		ctx SpecContext,
	) {
		inter, diagnostics := analyze(ctx, `
func f() f64 {
	x := 'pt_01'
	return x
}
sequence main {
	stage a {
		f() > 2.0 => next
	}
	stage b {
		1.0 -> 'out'
	}
}
'trig' => main
`)
		Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		Expect(onKeys(inter)).To(ContainElement(uint32(60)))
		Expect(onKeys(inter)).ToNot(ContainElement(uint32(2024)))
	})

	It("R2-T4 suggests a channel that a variable of its name shadows", func(
		ctx SpecContext,
	) {
		hint := func(local string) string {
			GinkgoHelper()
			_, diagnostics := analyze(ctx, "func f() {\n\t"+local+
				" := 1.0\n\t'out' = "+local+" + 'outy'\n}\n'trig' -> f()")
			return diagnostics.String()
		}
		Expect(hint("y")).To(ContainSubstring("did you mean: out, outx?"))
		Expect(hint("outx")).To(ContainSubstring("did you mean: out, outx?"))
	})

	It("R2-T5 hints a flow write that names a quoted channel in its quoted form", func(
		ctx SpecContext,
	) {
		_, diagnostics := analyze(ctx, `
sequence main {
	stage a {
		'valve-cmd' = 1.0
	}
}
'trig' => main
`)
		Expect(diagnostics.String()).To(ContainSubstring("-> 'valve-cmd'"))
	})
})
