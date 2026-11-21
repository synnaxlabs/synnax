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
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	xdiagnostics "github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Quoted channel names", func() {
	channel := func(name string, t types.Type, id int) symbol.Symbol {
		return symbol.Symbol{Name: name, Kind: symbol.KindChannel, Type: t, ID: id}
	}
	channels := []symbol.Symbol{
		channel("pt-01", types.Chan(types.F64()), 10),
		channel("valve-cmd", types.Chan(types.F64()), 20),
		channel("trig", types.Chan(types.U8()), 30),
		channel("x", types.Chan(types.F64()), 40),
		channel("\u00e9", types.Chan(types.F64()), 50),
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
	writes := func(inter ir.IR, key uint32) bool {
		for _, n := range inter.Nodes {
			if _, ok := n.Channels.Write[key]; ok {
				return true
			}
		}
		return false
	}

	It("Should read and write quoted channels in a flow", func(ctx SpecContext) {
		inter, diagnostics := analyze(ctx, `'pt-01' -> 'valve-cmd'`)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(reads(inter, 10)).To(BeTrue())
		Expect(writes(inter, 20)).To(BeTrue())
	})

	It("Should read a quoted channel in a flow expression", func(ctx SpecContext) {
		inter, diagnostics := analyze(ctx, `'pt-01' * 2.0 -> 'valve-cmd'`)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(reads(inter, 10)).To(BeTrue())
		Expect(writes(inter, 20)).To(BeTrue())
	})

	It("Should read and assign quoted channels in a function", func(ctx SpecContext) {
		inter, diagnostics := analyze(ctx, `
func f() {
	'valve-cmd' = 'pt-01' * 2.0
}
'trig' -> f()
`)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(reads(inter, 10)).To(BeTrue())
		Expect(writes(inter, 20)).To(BeTrue())
	})

	It("Should bind the channel beside a local of the same name", func(
		ctx SpecContext,
	) {
		inter, diagnostics := analyze(ctx, `
func f() {
	x := 1.0
	'x' = x + 'pt-01'
}
'trig' -> f()
`)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(writes(inter, 40)).To(BeTrue())
	})

	It("Should compile a quoted write and read beside a local of the same name", func(
		ctx SpecContext,
	) {
		compile := func(local string) []byte {
			GinkgoHelper()
			inter, diagnostics := analyze(ctx, "func f() {\n\t"+local+
				" := 1.0\n\t'x' = "+local+" + 'x'\n}\n'trig' -> f()")
			Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
			Expect(reads(inter, 40)).To(BeTrue())
			Expect(writes(inter, 40)).To(BeTrue())
			return MustSucceed(text.Compile(ctx, inter)).WASM
		}
		Expect(compile("x")).To(Equal(compile("y")))
	})

	It("Should set the authority of a quoted channel", func(ctx SpecContext) {
		inter, diagnostics := analyze(
			ctx,
			"authority ('valve-cmd' 200)\n'pt-01' -> 'valve-cmd'",
		)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(inter.Authorities.Channels).To(Equal(map[uint32]uint8{20: 200}))
	})

	It("Should match a channel name in NFC", func(ctx SpecContext) {
		inter, diagnostics := analyze(ctx, "'é' -> 'valve-cmd'")
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(reads(inter, 50)).To(BeTrue())
	})

	It("Should fail when no channel has the name", func(ctx SpecContext) {
		_, diagnostics := analyze(ctx, `'pt-02' -> 'valve-cmd'`)
		Expect(diagnostics.String()).To(ContainSubstring("undefined symbol: pt-02"))
	})

	It("Should not bind a function to a quoted name", func(ctx SpecContext) {
		_, diagnostics := analyze(ctx, `
func f() {
	'valve-cmd' = 1.0
}
'trig' -> 'f'
`)
		Expect(diagnostics.String()).To(ContainSubstring("undefined symbol: f"))
	})
})
