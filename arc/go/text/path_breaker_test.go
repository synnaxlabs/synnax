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
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	xdiagnostics "github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Bare channel paths (breaker)", func() {
	channel := func(name string, t types.Type, id int) symbol.Symbol {
		return symbol.Symbol{Name: name, Kind: symbol.KindChannel, Type: t, ID: id}
	}
	channels := []symbol.Symbol{
		channel("stand.sensors.pt_01", types.Chan(types.F64()), 10),
		channel("stand.valve_cmd", types.Chan(types.F64()), 20),
		channel("stand.trig", types.Chan(types.U8()), 30),
		channel("stand.for", types.Chan(types.F64()), 40),
		channel("stand.log", types.Chan(types.String()), 50),
	}
	analyze := func(ctx SpecContext, source string) (ir.IR, *xdiagnostics.Diagnostics) {
		GinkgoHelper()
		return text.Analyze(
			ctx,
			MustSucceed(text.Parse(text.Text{Raw: source})),
			NewRoot(nil, channels...),
		)
	}

	It("B1 reports a missing three-segment path under an imported module as E4011",
		func(ctx SpecContext) {
			_, diagnostics := analyze(
				ctx,
				"import math\n\nmath.inf.x -> stand.valve_cmd",
			)
			Expect(*diagnostics).To(ContainElement(HaveField(
				"Code",
				codes.ChannelUndefined.Code,
			)), diagnostics.String())
			Expect(diagnostics.String()).
				ToNot(ContainSubstring(`module "math" is not imported`))
		})

	It("B2 reports a missing bare path at an authority entry as E4011", func(
		ctx SpecContext,
	) {
		_, diagnostics := analyze(
			ctx,
			"authority (stand.pt_02 200)\nstand.sensors.pt_01 -> stand.valve_cmd",
		)
		Expect(*diagnostics).To(ContainElement(And(
			HaveField("Code", codes.ChannelUndefined.Code),
			HaveField("Message", "channel does not exist: stand.pt_02"),
		)), diagnostics.String())
	})

	It("B3 rejects an authority entry that names a module constant", func(
		ctx SpecContext,
	) {
		inter, diagnostics := analyze(
			ctx,
			"import math\n\nauthority (math.inf 200)\n"+
				"stand.sensors.pt_01 -> stand.valve_cmd",
		)
		Expect(diagnostics.Ok()).To(BeFalse(), "authorities: %v", inter.Authorities)
	})

	It("B5 compiles a path that names a member of a module it does not import", func(
		ctx SpecContext,
	) {
		source := "func f() {\n    stand.valve_cmd = math.inf\n}\nstand.trig -> f()"
		root := NewRoot(nil, append(
			[]symbol.Symbol{channel("math.inf", types.Chan(types.F64()), 60)},
			channels...,
		)...)
		inter, diagnostics := text.Analyze(
			ctx,
			MustSucceed(text.Parse(text.Text{Raw: source})),
			root,
		)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(inter.Nodes).ToNot(BeEmpty())
		_, err := text.Compile(ctx, inter)
		Expect(err).ToNot(HaveOccurred())
	})

	It("B4 names a missing path with the text that D-286 approved", func(
		ctx SpecContext,
	) {
		_, diagnostics := analyze(ctx, "stand.pt_02 -> stand.valve_cmd")
		Expect(*diagnostics).To(ContainElement(HaveField(
			"Message",
			"stand.pt_02 does not exist",
		)), diagnostics.String())
	})

	It("C1 compiles a bare path whose last segment is the keyword for", func(
		ctx SpecContext,
	) {
		inter, diagnostics := analyze(ctx, "stand.for -> stand.valve_cmd")
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
		Expect(inter.Nodes).ToNot(BeEmpty())
	})

	It("C2 compiles a bare path in a format string placeholder", func(
		ctx SpecContext,
	) {
		_, diagnostics := analyze(
			ctx,
			"func f() {\n    stand.log = f\"{stand.sensors.pt_01}\"\n}\n"+
				"stand.trig -> f()\n",
		)
		Expect(*diagnostics).To(BeEmpty(), diagnostics.String())
	})

	It("C6 compiles a bare path with a dashed segment when dashed names are on", func(
		ctx SpecContext,
	) {
		cfg := parser.Config{AllowDashedNames: true}
		source := "func f() {\n    y := stand.pt-01\n}\nstand.trig -> f()\n"
		root := NewRoot(nil, append(
			[]symbol.Symbol{channel("stand.pt-01", types.Chan(types.F64()), 70)},
			channels...,
		)...)
		parsed := MustSucceed(text.Parse(text.Text{Raw: source}, cfg))
		_, diagnostics := text.Analyze(ctx, parsed, root, cfg)
		Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	})

	It("B6 rejects an assignment to a module constant through a bare path", func(
		ctx SpecContext,
	) {
		_, diagnostics := analyze(
			ctx,
			"import math\n\nfunc f() {\n    math.inf = 1.0\n}\nstand.trig -> f()",
		)
		Expect(diagnostics.Ok()).To(BeFalse())
	})

	It("P1 probe: an authority entry that names a function", func(ctx SpecContext) {
		inter, diagnostics := analyze(
			ctx,
			"func g() f64 {\n    return 1.0\n}\nauthority (g 200)\n"+
				"stand.sensors.pt_01 -> stand.valve_cmd",
		)
		Expect(diagnostics.Ok()).To(BeFalse(), "authorities: %v", inter.Authorities)
	})
})
