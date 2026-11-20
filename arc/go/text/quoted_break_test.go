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
	"github.com/synnaxlabs/arc/formatter"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	xdiagnostics "github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Quoted channel names (break)", func() {
	channel := func(name string, t types.Type, id int) symbol.Symbol {
		return symbol.Symbol{Name: name, Kind: symbol.KindChannel, Type: t, ID: id}
	}
	channels := []symbol.Symbol{
		channel("2024", types.Chan(types.F64()), 10),
		channel("out", types.Chan(types.F64()), 20),
		channel("trig", types.Chan(types.U8()), 30),
		channel("x", types.Chan(types.F64()), 2024),
		channel("pt:01", types.Chan(types.F64()), 50),
		channel("pt_01", types.Chan(types.F64()), 60),
		channel("log", types.Chan(types.String()), 80),
		channel("a}b", types.Chan(types.F64()), 90),
	}
	analyze := func(ctx SpecContext, source string) (ir.IR, *xdiagnostics.Diagnostics) {
		GinkgoHelper()
		return text.Analyze(
			ctx,
			MustSucceed(text.Parse(text.Text{Raw: source})),
			NewRoot(nil, channels...),
		)
	}
	reads := func(inter ir.IR, key uint32) bool {
		for _, n := range inter.Nodes {
			if _, ok := n.Channels.Read[key]; ok {
				return true
			}
		}
		return false
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

	Describe("A: a digit-only quoted name", func() {
		It("A0 binds the channel of that name at the head of a flow", func(
			ctx SpecContext,
		) {
			inter, diagnostics := analyze(ctx, `'2024' -> 'out'`)
			Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
			Expect(reads(inter, 10)).To(BeTrue())
		})

		It("A1 binds the channel of that name in a flow expression", func(
			ctx SpecContext,
		) {
			inter, diagnostics := analyze(ctx, `'2024' * 2.0 -> 'out'`)
			Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
			Expect(
				reads(inter, 2024),
			).To(BeFalse(), "reads channel x, whose ID is 2024")
			Expect(reads(inter, 10)).To(BeTrue())
		})
	})

	Describe("B: an empty quoted name", func() {
		It("B1 fails analysis in a declaration", func() {
			_, diagnostics := text.Parse(
				text.Text{Raw: "func f() {\n\ty := ''\n}\n'trig' -> f()"},
			)
			Expect(diagnostics.String()).To(HavePrefix(
				"2:6 error [E1000]: syntax error: token recognition error at: ''''",
			))
		})

		It("B2 fails analysis as an assigned value", func() {
			_, diagnostics := text.Parse(
				text.Text{Raw: "func f() {\n\t'out' = ''\n}\n'trig' -> f()"},
			)
			Expect(diagnostics.String()).To(HavePrefix(
				"2:9 error [E1000]: syntax error: token recognition error at: ''''",
			))
		})
	})

	Describe("C: a quoted name in a format string placeholder", func() {
		It("C1 may hold a colon", func(ctx SpecContext) {
			_, diagnostics := analyze(ctx, `'trig' -> f"{'pt:01'}" -> 'log'`)
			Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		})

		It("C2 may hold a closing brace", func(ctx SpecContext) {
			_, diagnostics := analyze(ctx, `'trig' -> f"{'a}b'}" -> 'log'`)
			Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		})
	})

	Describe("D: a quoted channel as a channel input of a function instance", func() {
		It("D1 binds by name", func(ctx SpecContext) {
			inter, diagnostics := analyze(ctx,
				"func r(c chan f64) f64 {\n\treturn c\n}\nr(c='pt_01') -> 'out'")
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(reads(inter, 60)).To(BeTrue())
		})

		It("D2 binds by position", func(ctx SpecContext) {
			_, diagnostics := analyze(
				ctx,
				"func r(c chan f64) f64 {\n\treturn c\n}\n'trig' -> r('pt_01') -> 'out'",
			)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		})
	})

	Describe("E: the trigger of a quoted channel beside a symbol of its name", func() {
		It("E1 reads the channel beside a flow variable", func(ctx SpecContext) {
			inter, diagnostics := analyze(ctx, "x := 3.0\n'x' * 2.0 -> 'out'")
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			MustSucceed(text.Compile(ctx, inter))
			Expect(onKeys(inter)).To(ConsistOf(uint32(2024)))
		})

		It("E2 reads the channel beside a function", func(ctx SpecContext) {
			inter, diagnostics := analyze(ctx,
				"func x() f64 {\n\treturn 1.0\n}\n'x' * 2.0 -> 'out'")
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(onKeys(inter)).To(ConsistOf(uint32(2024)))
		})

		It("E3 reads the channel in a call argument beside a flow variable", func(
			ctx SpecContext,
		) {
			inter, diagnostics := analyze(ctx,
				"x := 3.0\nfunc g(v f64) f64 {\n\treturn v\n}\ng('x' * 2.0) -> 'out'")
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(onKeys(inter)).To(ContainElement(uint32(2024)))
		})

		It("E4 reads the channel in a stage condition beside a flow variable", func(
			ctx SpecContext,
		) {
			inter, diagnostics := analyze(ctx, `
x := 1.0
sequence main {
	stage a {
		'x' > 2.0 => next
	}
	stage b {
		1.0 -> 'out'
	}
}
'pt_01' => main
`)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(onKeys(inter)).To(ContainElement(uint32(2024)))
		})
	})

	It("F1 the formatter keeps the meaning of a quoted name before a paren", func(
		ctx SpecContext,
	) {
		src := "'pt_01' -> 'out' (2.0) -> 'out'"
		_, before := analyze(ctx, src)
		Expect(before.Ok()).To(BeTrue(), before.String())
		formatted := formatter.Format(src)
		_, after := analyze(ctx, formatted)
		Expect(after.Ok()).To(BeTrue(), formatted+"\n"+after.String())
	})

	It("G1 an undefined quoted name suggests only channels", func(ctx SpecContext) {
		_, diagnostics := analyze(ctx,
			"func outx() {\n\t'out' = 1.0\n}\n'pt_01' -> 'outy'")
		Expect(diagnostics.String()).To(ContainSubstring("undefined symbol: outy"))
		Expect(diagnostics.String()).ToNot(ContainSubstring("outx"))
	})
})
