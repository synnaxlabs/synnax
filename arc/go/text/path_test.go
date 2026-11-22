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
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	xdiagnostics "github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Bare channel paths", func() {
	channel := func(name string, t types.Type, id int) symbol.Symbol {
		return symbol.Symbol{Name: name, Kind: symbol.KindChannel, Type: t, ID: id}
	}
	channels := []symbol.Symbol{
		channel("stand.sensors.pt_01", types.Chan(types.F64()), 10),
		channel("stand.valve_cmd", types.Chan(types.F64()), 20),
		channel("stand.trig", types.Chan(types.U8()), 30),
		channel("math.inf", types.Chan(types.F64()), 40),
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

	It("Should read and write bare paths in a flow", func(ctx SpecContext) {
		inter, diagnostics := analyze(ctx, "stand.sensors.pt_01 -> stand.valve_cmd")
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(reads(inter, 10)).To(BeTrue())
		Expect(writes(inter, 20)).To(BeTrue())
	})

	It("Should read a bare path in a flow expression", func(ctx SpecContext) {
		inter, diagnostics := analyze(
			ctx,
			"stand.sensors.pt_01 * 2.0 -> stand.valve_cmd",
		)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(reads(inter, 10)).To(BeTrue())
		Expect(writes(inter, 20)).To(BeTrue())
	})

	It("Should read and assign bare paths in a function and compile", func(
		ctx SpecContext,
	) {
		inter, diagnostics := analyze(ctx, `
func f() {
	stand.valve_cmd = stand.sensors.pt_01 * 2.0
}
stand.trig -> f()
`)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(reads(inter, 10)).To(BeTrue())
		Expect(writes(inter, 20)).To(BeTrue())
		MustSucceed(text.Compile(ctx, inter))
	})

	It("Should set the authority of a bare path", func(ctx SpecContext) {
		inter, diagnostics := analyze(
			ctx,
			"authority (stand.valve_cmd 200)\nstand.sensors.pt_01 -> stand.valve_cmd",
		)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(inter.Authorities.Channels).To(Equal(map[uint32]uint8{20: 200}))
	})

	It("Should report that a bare path with no channel does not exist", func(
		ctx SpecContext,
	) {
		_, diagnostics := analyze(ctx, "stand.pt_02 -> stand.valve_cmd")
		Expect(*diagnostics).To(ContainElement(HaveField(
			"Message",
			"stand.pt_02 does not exist",
		)), diagnostics.String())
	})

	DescribeTable("Should fail a module member that a channel path matches",
		func(ctx SpecContext, source string) {
			_, diagnostics := analyze(ctx, source)
			Expect(*diagnostics).To(ContainElement(HaveField(
				"Code",
				codes.PathClash.Code,
			)), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"module member and channel share a name: math.inf (module math)",
			))
		},
		Entry(
			"in a flow expression",
			"import math\n\nmath.inf * 2.0 -> stand.valve_cmd",
		),
		Entry("at a flow node", "import math\n\nmath.inf -> stand.valve_cmd"),
		Entry("at an authority entry", "import math\n\nauthority (math.inf 200)"),
	)

	It("Should read a module member that no channel path matches", func(
		ctx SpecContext,
	) {
		_, diagnostics := analyze(
			ctx,
			"import math as m\n\nstand.sensors.pt_01 * m.inf -> stand.valve_cmd",
		)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
	})
})
