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
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("User modules", func() {
	const (
		trig    uint32 = 100
		fnOut   uint32 = 101
		seqOut  uint32 = 102
		textOut uint32 = 103
	)
	resolver := channelSymbols(map[string]channelDef{
		"trig":     {types.U8(), int(trig)},
		"fn_out":   {types.I64(), int(fnOut)},
		"seq_out":  {types.U8(), int(seqOut)},
		"text_out": {types.String(), int(textOut)},
	})
	digests := []channels.Digest{
		{Key: trig, DataType: telem.Uint8T},
		{Key: fnOut, DataType: telem.Int64T},
		{Key: seqOut, DataType: telem.Uint8T},
		{Key: textOut, DataType: telem.StringT},
	}
	last := func(fr telem.Frame[uint32], key uint32) telem.Series {
		GinkgoHelper()
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		return ch.Series[len(ch.Series)-1]
	}
	// run ingests one trigger sample per pulse, settles the program after each,
	// and returns what it wrote.
	run := func(ctx SpecContext, h *runtimeHarness, pulses int) telem.Frame[uint32] {
		GinkgoHelper()
		elapsed := telem.TimeSpan(0)
		for range pulses {
			h.Ingest(trig, telem.NewSeriesV[uint8](1))
			for range 3 {
				elapsed += telem.Millisecond
				h.Tick(ctx, elapsed)
				h.channelState.ClearReads()
			}
		}
		out, _ := h.Flush()
		return out
	}

	It("Should call an exported function and activate an exported sequence", func(
		ctx SpecContext,
	) {
		h := newModuleRuntimeHarness(ctx, `
import m

trig -> m.f() -> fn_out
trig == 1 => m.s
`, map[string]string{"m": `
export func f() {t u8} i64 { return 7 }
export sequence s { stage a { 1 -> seq_out } }
`}, resolver, digests...)
		defer h.Close(ctx)
		out := run(ctx, h, 1)
		Expect(last(out, fnOut).Unmarshal[int64]()).To(Equal([]int64{7}))
		Expect(last(out, seqOut).Unmarshal[uint8]()).To(Equal([]uint8{1}))
	})

	It("Should call through a module that imports a module", func(ctx SpecContext) {
		h := newModuleRuntimeHarness(ctx, `
import lib.a

trig -> a.f() -> fn_out
`, map[string]string{
			"lib.a": "import lib.b\n\nexport func f() {t u8} i64 { return b.h() + 1 }",
			"lib.b": "export func h() i64 { return 41 }",
		}, resolver, digests...)
		defer h.Close(ctx)
		out := run(ctx, h, 1)
		Expect(last(out, fnOut).Unmarshal[int64]()).To(Equal([]int64{42}))
	})

	It("Should read an exported expression variable", func(ctx SpecContext) {
		h := newModuleRuntimeHarness(ctx, `
import m

m.doubled -> fn_out
`, map[string]string{"m": `
export doubled := i64(trig) * 2
`}, resolver, digests...)
		defer h.Close(ctx)
		out := run(ctx, h, 1)
		Expect(last(out, fnOut).Unmarshal[int64]()).To(Equal([]int64{2}))
	})

	It("Should read a dependency's expression variable from a module stage", func(
		ctx SpecContext,
	) {
		h := newModuleRuntimeHarness(ctx, `
import a

trig == 1 => a.s
`, map[string]string{
			"a": "import b\n\nexport stage s { b.doubled -> fn_out }",
			"b": "export doubled := i64(trig) * 2",
		}, resolver, digests...)
		defer h.Close(ctx)
		out := run(ctx, h, 2)
		vals := last(out, fnOut).Unmarshal[int64]()
		Expect(vals[len(vals)-1]).To(Equal(int64(2)))
	})

	It("Should read an exported expression variable inside an expression", func(
		ctx SpecContext,
	) {
		h := newModuleRuntimeHarness(ctx, `
import m

m.doubled + 1 -> fn_out
`, map[string]string{"m": `
export doubled := i64(trig) * 2
`}, resolver, digests...)
		defer h.Close(ctx)
		vals := last(run(ctx, h, 1), fnOut).Unmarshal[int64]()
		Expect(vals[len(vals)-1]).To(Equal(int64(3)))
	})

	It("Should format an exported expression variable", func(ctx SpecContext) {
		h := newModuleRuntimeHarness(ctx, `
import m

trig -> f"v={m.doubled}" -> text_out
`, map[string]string{"m": `
export doubled := i64(trig) * 2
`}, resolver, digests...)
		defer h.Close(ctx)
		texts := last(run(ctx, h, 1), textOut).Unmarshal[string]()
		Expect(texts[len(texts)-1]).To(Equal("v=2"))
	})

	It(
		"Should read a dependency's expression variable inside a module expression",
		func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import a

trig == 1 => a.s
`, map[string]string{
				"a": "import b\n\nexport stage s { b.doubled + 1 -> fn_out }",
				"b": "export doubled := i64(trig) * 2",
			}, resolver, digests...)
			defer h.Close(ctx)
			vals := last(run(ctx, h, 2), fnOut).Unmarshal[int64]()
			Expect(vals[len(vals)-1]).To(Equal(int64(3)))
		},
	)

	It("Should bind a variable to an exported channel alias", func(ctx SpecContext) {
		h := newModuleRuntimeHarness(ctx, `
import m

x := m.b
x -> seq_out
`, map[string]string{"m": "export b := trig"}, resolver, digests...)
		defer h.Close(ctx)
		vals := last(run(ctx, h, 1), seqOut).Unmarshal[uint8]()
		Expect(vals[len(vals)-1]).To(Equal(uint8(1)))
	})

	It("Should run an exported function that formats a channel it reads", func(
		ctx SpecContext,
	) {
		h := newModuleRuntimeHarness(ctx, `
import m

m.reactive() -> text_out
`, map[string]string{"m": `
export func reactive() str { return f"Data: {trig}" }
`}, resolver, digests...)
		defer h.Close(ctx)
		out := run(ctx, h, 1)
		texts := last(out, textOut).Unmarshal[string]()
		Expect(texts[len(texts)-1]).To(Equal("Data: 1"))
	})

	It("Should activate a module sequence from inside a sequence", func(
		ctx SpecContext,
	) {
		h := newModuleRuntimeHarness(ctx, `
import m as mm

sequence main {
    stage go { trig == 1 => mm.s }
}

trig == 1 => main
`, map[string]string{"m": `
export sequence s { stage a { 1 -> seq_out } }
`}, resolver, digests...)
		defer h.Close(ctx)
		// The first pulse starts main; a stage sees only samples after its entry.
		out := run(ctx, h, 2)
		Expect(last(out, seqOut).Unmarshal[uint8]()).To(Equal([]uint8{1}))
	})
})
