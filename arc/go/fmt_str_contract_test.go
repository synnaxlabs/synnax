// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

// The text each format string renders to, in a function and in a flow.
var _ = Describe("Format string output", func() {
	run := func(ctx SpecContext, source string) string {
		GinkgoHelper()
		h := newRuntimeHarness(ctx, source, channelSymbols(map[string]channelDef{
			"trig": {types.U8(), 100},
			"log":  {types.String(), 101},
		}),
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.StringT},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV[uint8](1))
		for range 3 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		series := out.Get(101).Series
		Expect(series).ToNot(BeEmpty())
		values := series[len(series)-1].Unmarshal[string]()
		return values[len(values)-1]
	}
	inFunction := func(literal string) string {
		return "func f() {\n    n i64 := 42\n    log = " + literal + "\n}\ntrig -> f()"
	}
	inFlow := func(literal string) string {
		return "trig -> " + literal + " -> log"
	}

	DescribeTable("Should render",
		func(ctx SpecContext, literal, expected string) {
			Expect(run(ctx, inFunction(literal))).To(Equal(expected), "in a function")
			Expect(run(ctx, inFlow(literal))).To(Equal(expected), "in a flow")
		},
		Entry("plain text", `f"abc"`, "abc"),
		Entry("doubled open brace", `f"{{"`, "{"),
		Entry("doubled close brace", `f"}}"`, "}"),
		Entry("doubled braces around text", `f"{{x}}"`, "{x}"),
		Entry("a bare close brace", `f"}"`, "}"),
		Entry("a placeholder", `f"{42}"`, "42"),
		Entry("doubled braces and a placeholder", `f"{{{42}}}"`, "{42}"),
		Entry("text around a placeholder", `f"a {42} b"`, "a 42 b"),
		Entry("a tab escape", `f"\t{42}"`, "\t42"),
		Entry("a newline escape", `f"a\nb"`, "a\nb"),
		Entry("an escaped quote", `f"\"{42}\""`, `"42"`),
		Entry("an escaped backslash", `f"\\{42}"`, `\42`),
		Entry("a backslash before a placeholder", `f"\{42}"`, `\42`),
		Entry("a path before a placeholder", `f"C:\data\{42}"`, `C:\data\42`),
		Entry("a unicode escape", `f"\u00B0{42}"`, "°42"),
		// Escapes apply before the split. On main, only a flow rendered this as {42}.
		Entry("an escaped open brace", `f"\u007B42}"`, "42"),
		Entry("non-ASCII text", `f"温度 {42}°"`, "温度 42°"),
		Entry("raw text", `rf"a\nb"`, `a\nb`),
		Entry("a raw backslash before a placeholder", `rf"C:\{42}"`, `C:\42`),
		Entry("a spec", `f"{3.14159:.2f}"`, "3.14"),
	)

	DescribeTable("Should render a stateful default",
		func(ctx SpecContext, literal, expected string) {
			Expect(
				run(
					ctx,
					"func f() {\n    s str $= "+literal+"\n    log = s\n}\ntrig -> f()",
				),
			).
				To(Equal(expected))
		},
		Entry("plain text", `f"abc"`, "abc"),
		Entry("doubled braces", `f"{{x}}"`, "{x}"),
		Entry("a backslash before a brace", `f"\{{"`, `\{`),
		Entry("non-ASCII text", `f"温度°"`, "温度°"),
	)

	DescribeTable("Should reject",
		func(ctx SpecContext, literal, message string) {
			for _, source := range []string{inFunction(literal), inFlow(literal)} {
				root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
					Name: "trig",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   100,
				}, &symbol.Symbol{
					Name: "log",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.String()),
					ID:   101,
				}))
				Expect(arc.CompileText(ctx, arc.Text{Raw: source}, root)).Error().
					To(MatchError(ContainSubstring(message)), source)
			}
		},
		Entry("an unmatched open brace", `f"{"`, "unmatched '{'"),
		Entry("an escaped open brace alone", `f"\u007B"`, "unmatched '{'"),
		Entry("an empty placeholder", `f"{}"`, "must contain an expression"),
	)
})
