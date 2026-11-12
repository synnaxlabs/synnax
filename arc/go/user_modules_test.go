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
	"fmt"
	"slices"
	"strings"

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
		trig2   uint32 = 104
		fnOut2  uint32 = 105
	)
	resolver := channelSymbols(map[string]channelDef{
		"trig":     {types.U8(), int(trig)},
		"fn_out":   {types.I64(), int(fnOut)},
		"seq_out":  {types.U8(), int(seqOut)},
		"text_out": {types.String(), int(textOut)},
		"trig2":    {types.U8(), int(trig2)},
		"fn_out2":  {types.I64(), int(fnOut2)},
	})
	digests := []channels.Digest{
		{Key: trig, DataType: telem.Uint8T},
		{Key: fnOut, DataType: telem.Int64T},
		{Key: seqOut, DataType: telem.Uint8T},
		{Key: textOut, DataType: telem.StringT},
		{Key: trig2, DataType: telem.Uint8T},
		{Key: fnOut2, DataType: telem.Int64T},
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
	// pulses ingests one trigger sample per value, settles the program after each,
	// and returns what it wrote after each sample.
	pulses := func(
		ctx SpecContext,
		h *runtimeHarness,
		values ...uint8,
	) []telem.Frame[uint32] {
		GinkgoHelper()
		elapsed := telem.TimeSpan(0)
		frames := make([]telem.Frame[uint32], 0, len(values))
		for _, v := range values {
			h.Ingest(trig, telem.NewSeriesV[uint8](v))
			for range 3 {
				elapsed += telem.Millisecond
				h.Tick(ctx, elapsed)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			frames = append(frames, out)
		}
		return frames
	}

	// send ingests v on key, settles the program, and returns what it wrote.
	send := func(
		ctx SpecContext,
		h *runtimeHarness,
		key uint32,
		v uint8,
	) telem.Frame[uint32] {
		GinkgoHelper()
		h.Ingest(key, telem.NewSeriesV[uint8](v))
		for i := range 3 {
			h.Tick(ctx, telem.TimeSpan(i+1)*telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		return out
	}
	// all returns every sample written to key across frames.
	all := func(frames []telem.Frame[uint32], key uint32) []int64 {
		var r []int64
		for _, fr := range frames {
			for _, s := range fr.Get(key).Series {
				r = append(r, s.Unmarshal[int64]()...)
			}
		}
		return r
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

	It("Should alias an exported channel alias inside a function", func(
		ctx SpecContext,
	) {
		h := newModuleRuntimeHarness(ctx, `
import m

func f() u8 {
    x := m.b
    return x
}

trig -> f() -> seq_out
`, map[string]string{"m": "export b := trig"}, resolver, digests...)
		defer h.Close(ctx)
		vals := last(run(ctx, h, 1), seqOut).Unmarshal[uint8]()
		Expect(vals[len(vals)-1]).To(Equal(uint8(1)))
	})

	It("Should bind a variable to an exported function that reads a channel", func(
		ctx SpecContext,
	) {
		h := newModuleRuntimeHarness(ctx, `
import m

x := m.f()
x + 1 -> fn_out
`, map[string]string{"m": "export func f() i64 { return i64(trig) * 2 }"},
			resolver, digests...)
		defer h.Close(ctx)
		vals := last(run(ctx, h, 1), fnOut).Unmarshal[int64]()
		Expect(vals[len(vals)-1]).To(Equal(int64(3)))
	})

	Describe("Triggers", func() {
		It("Should fire an expression on each sample of an exported alias", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

i64(m.b) * 2 -> fn_out
`, map[string]string{"m": "export b := trig"}, resolver, digests...)
			defer h.Close(ctx)
			frames := pulses(ctx, h, 1, 3)
			Expect(
				last(frames[0], fnOut).Unmarshal[int64](),
			).To(ContainElement(int64(2)))
			Expect(
				last(frames[1], fnOut).Unmarshal[int64](),
			).To(ContainElement(int64(6)))
		})

		It(
			"Should fire an expression on each value of an exported expression variable",
			func(
				ctx SpecContext,
			) {
				h := newModuleRuntimeHarness(ctx, `
import m

m.doubled + 1 -> fn_out
`, map[string]string{"m": "export doubled := i64(trig) * 2"}, resolver, digests...)
				defer h.Close(ctx)
				frames := pulses(ctx, h, 1, 2)
				Expect(
					last(frames[0], fnOut).Unmarshal[int64](),
				).To(ContainElement(int64(3)))
				Expect(
					last(frames[1], fnOut).Unmarshal[int64](),
				).To(ContainElement(int64(5)))
			},
		)

		It("Should fire a flow from a variable bound to an exported alias", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

x := m.b
x -> seq_out
`, map[string]string{"m": "export b := trig"}, resolver, digests...)
			defer h.Close(ctx)
			frames := pulses(ctx, h, 1, 4)
			Expect(
				last(frames[0], seqOut).Unmarshal[uint8](),
			).To(ContainElement(uint8(1)))
			Expect(
				last(frames[1], seqOut).Unmarshal[uint8](),
			).To(ContainElement(uint8(4)))
		})

		It("Should fire a flow from a variable bound to an exported function", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

x := m.f()
x -> fn_out
`, map[string]string{"m": "export func f() i64 { return i64(trig) * 2 }"},
				resolver, digests...)
			defer h.Close(ctx)
			frames := pulses(ctx, h, 1, 3)
			Expect(
				last(frames[0], fnOut).Unmarshal[int64](),
			).To(ContainElement(int64(2)))
			Expect(
				last(frames[1], fnOut).Unmarshal[int64](),
			).To(ContainElement(int64(6)))
		})

		It("Should transition on an exported expression variable", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

sequence main {
    stage wait {
        m.doubled > 2 => next
    }
    stage done {
        u8(1) -> seq_out
    }
}

trig == 1 => main
`, map[string]string{"m": "export doubled := i64(trig) * 2"}, resolver, digests...)
			defer h.Close(ctx)
			frames := pulses(ctx, h, 1, 2)
			Expect(frames[0].Get(seqOut).Series).To(BeEmpty())
			Expect(
				last(frames[1], seqOut).Unmarshal[uint8](),
			).To(ContainElement(uint8(1)))
		})

		It("Should transition a module sequence on a dependency's variable", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import a

trig == 1 => a.s
`, map[string]string{
				"a": `
import b

export sequence s {
    stage wait {
        b.doubled > 2 => next
    }
    stage done {
        u8(1) -> seq_out
    }
}`,
				"b": "export doubled := i64(trig) * 2",
			}, resolver, digests...)
			defer h.Close(ctx)
			frames := pulses(ctx, h, 1, 2)
			Expect(frames[0].Get(seqOut).Series).To(BeEmpty())
			Expect(
				last(frames[1], seqOut).Unmarshal[uint8](),
			).To(ContainElement(uint8(1)))
		})

		Describe("Calls that read channels", func() {
			const (
				a    uint32 = 110
				b    uint32 = 111
				c    uint32 = 112
				done uint32 = 113
				sum  uint32 = 114
			)
			const turbo = `
limit f32 := 15

export func stopped() bool {
    return a < limit and b < limit and c < limit
}

export func total() f32 {
    return a + b + c
}`
			const cooldown = `
sequence s {
    stage cool {
        %s => next
    }
    stage evacuate {
        u8(1) -> done
        trig == 2 => cool
    }
}

trig == 1 => s
`
			newHarness := func(
				ctx SpecContext,
				source string,
				modules map[string]string,
			) *runtimeHarness {
				GinkgoHelper()
				return newModuleRuntimeHarness(
					ctx,
					source,
					modules,
					slices.Concat(resolver, channelSymbols(map[string]channelDef{
						"a":    {types.F32(), int(a)},
						"b":    {types.F32(), int(b)},
						"c":    {types.F32(), int(c)},
						"done": {types.U8(), int(done)},
						"sum":  {types.F32(), int(sum)},
					})),
					slices.Concat(digests, []channels.Digest{
						{Key: done, DataType: telem.Uint8T},
						{Key: sum, DataType: telem.Float32T},
					})...,
				)
			}
			// set ingests v on key, settles the program, and returns what it wrote.
			set := func(
				ctx SpecContext,
				h *runtimeHarness,
				key uint32,
				v float32,
			) telem.Frame[uint32] {
				GinkgoHelper()
				h.Ingest(key, telem.NewSeriesV(v))
				for t := range 3 {
					h.Tick(ctx, telem.TimeSpan(t+1)*telem.Millisecond)
					h.channelState.ClearReads()
				}
				out, _ := h.Flush()
				return out
			}
			// SY-5483 will make a call wait for a value. Until then, an empty channel
			// reads as 0, so every channel gets a value before the sequence starts.
			start := func(ctx SpecContext, h *runtimeHarness) {
				GinkgoHelper()
				for _, key := range []uint32{a, b, c} {
					set(ctx, h, key, 20)
				}
				Expect(pulses(ctx, h, 1)[0].Get(done).Series).To(BeEmpty())
			}
			// drop lowers each channel below the limit in order and returns what the
			// last drop wrote. The earlier drops must write nothing.
			drop := func(
				ctx SpecContext,
				h *runtimeHarness,
				order ...uint32,
			) telem.Frame[uint32] {
				GinkgoHelper()
				for _, key := range order[:len(order)-1] {
					Expect(set(ctx, h, key, 10).Get(done).Series).To(BeEmpty())
				}
				return set(ctx, h, order[len(order)-1], 10)
			}
			modules := map[string]string{"turbo": turbo}
			module := "import turbo\n" + fmt.Sprintf(cooldown, "turbo.stopped()")

			DescribeTable(
				"Should transition when the last channel a function reads updates",
				func(ctx SpecContext, order ...uint32) {
					h := newHarness(ctx, module, modules)
					defer h.Close(ctx)
					start(ctx, h)
					Expect(last(drop(ctx, h, order...), done).Unmarshal[uint8]()).
						To(Equal([]uint8{1}))
				},
				Entry("a updates last", b, c, a),
				Entry("b updates last", a, c, b),
				Entry("c updates last", a, b, c),
			)

			It("Should not transition on a channel that rose again", func(
				ctx SpecContext,
			) {
				h := newHarness(ctx, module, modules)
				defer h.Close(ctx)
				start(ctx, h)
				Expect(set(ctx, h, a, 10).Get(done).Series).To(BeEmpty())
				Expect(set(ctx, h, a, 20).Get(done).Series).To(BeEmpty())
				Expect(drop(ctx, h, b, c).Get(done).Series).To(BeEmpty())
				Expect(last(set(ctx, h, a, 10), done).Unmarshal[uint8]()).
					To(Equal([]uint8{1}))
			})

			DescribeTable(
				"Should transition on a function defined",
				func(ctx SpecContext, source string, modules map[string]string) {
					h := newHarness(ctx, source, modules)
					defer h.Close(ctx)
					start(ctx, h)
					Expect(last(drop(ctx, h, a, b, c), done).Unmarshal[uint8]()).
						To(Equal([]uint8{1}))
				},
				Entry(
					"in the automation",
					strings.ReplaceAll(turbo, "export ", "")+
						fmt.Sprintf(cooldown, "stopped()"),
					nil,
				),
				Entry(
					"in a module that imports a module",
					"import lib.turbo\n"+fmt.Sprintf(cooldown, "turbo.stopped()"),
					map[string]string{
						"lib.limits": "export limit f32 := 15",
						"lib.turbo": `
import lib.limits

export func stopped() bool {
    return a < limits.limit and b < limits.limit and c < limits.limit
}`,
					},
				),
			)

			It("Should transition again after the stage is re-entered", func(
				ctx SpecContext,
			) {
				h := newHarness(ctx, module, modules)
				defer h.Close(ctx)
				start(ctx, h)
				Expect(last(drop(ctx, h, a, b, c), done).Unmarshal[uint8]()).
					To(Equal([]uint8{1}))
				for _, key := range []uint32{a, b, c} {
					set(ctx, h, key, 20)
				}
				Expect(pulses(ctx, h, 2)[0].Get(done).Series).To(BeEmpty())
				Expect(last(drop(ctx, h, b, c, a), done).Unmarshal[uint8]()).
					To(Equal([]uint8{1}))
			})

			It("Should write a function's result when any channel it reads updates",
				func(ctx SpecContext) {
					h := newHarness(
						ctx,
						"import turbo\n\nturbo.total() -> sum",
						modules,
					)
					defer h.Close(ctx)
					for _, key := range []uint32{a, b, c} {
						set(ctx, h, key, 1)
					}
					var sums []float32
					for _, key := range []uint32{a, b, c} {
						vals := last(set(ctx, h, key, 2), sum).Unmarshal[float32]()
						sums = append(sums, vals[len(vals)-1])
					}
					Expect(sums).To(Equal([]float32{4, 5, 6}))
				},
			)
		})
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

	Describe("Stateful variables", func() {
		const counter = `
export func count() i64 {
    n i64 $= 0
    n = n + 1
    return n
}`

		It("Should keep a stateful variable across calls to an exported function", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

trig -> m.count() -> fn_out
`, map[string]string{"m": counter}, resolver, digests...)
			defer h.Close(ctx)
			Expect(all(pulses(ctx, h, 1, 1, 1, 1), fnOut)).
				To(Equal([]int64{1, 2, 3, 4}))
		})

		It("Should keep separate state for each call site", func(ctx SpecContext) {
			h := newModuleRuntimeHarness(ctx, `
import m

trig -> m.count() -> fn_out
trig2 -> m.count() -> fn_out2
`, map[string]string{"m": counter}, resolver, digests...)
			defer h.Close(ctx)
			frames := pulses(ctx, h, 1, 1, 1)
			h.Ingest(trig2, telem.NewSeriesV[uint8](1))
			frames = append(frames, pulses(ctx, h, 1)...)
			Expect(all(frames, fnOut)).To(Equal([]int64{1, 2, 3, 4}))
			Expect(all(frames, fnOut2)).To(Equal([]int64{1}))
		})

		It("Should keep state in a function called from a module stage", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

trig == 1 => m.s
`, map[string]string{"m": counter + `

export sequence s { stage a { trig -> count() -> fn_out } }`},
				resolver, digests...)
			defer h.Close(ctx)
			// The first pulse only starts the sequence.
			Expect(all(pulses(ctx, h, 1, 1, 1, 1), fnOut)).
				To(Equal([]int64{1, 2, 3}))
		})

		It("Should keep state in a function of a module that a module imports", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import lib.a

trig -> a.f() -> fn_out
`, map[string]string{
				"lib.a": `
import lib.b

export func f() {t u8} i64 {
    calls i64 $= 0
    calls = calls + 1
    return calls * 100 + b.count()
}`,
				"lib.b": counter,
			}, resolver, digests...)
			defer h.Close(ctx)
			Expect(all(pulses(ctx, h, 1, 1, 1, 1), fnOut)).
				To(Equal([]int64{101, 202, 303, 404}))
		})

		It("Should keep separate state for each module that imports a module", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import lib.a
import lib.c

trig -> a.f() -> fn_out
trig -> c.f() -> fn_out2
`, map[string]string{
				"lib.a": "import lib.b\n\nexport func f() {t u8} i64 { return b.count() }",
				"lib.c": "import lib.b\n\nexport func f() {t u8} i64 { return b.count() }",
				"lib.b": counter,
			}, resolver, digests...)
			defer h.Close(ctx)
			frames := pulses(ctx, h, 1, 1, 1)
			Expect(all(frames, fnOut)).To(Equal([]int64{1, 2, 3}))
			Expect(all(frames, fnOut2)).To(Equal([]int64{1, 2, 3}))
		})

		It("Should keep state in an exported function bound to a variable", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

x := m.sum()
x -> fn_out
`, map[string]string{"m": `
export func sum() i64 {
    n i64 $= 0
    n = n + i64(trig)
    return n
}`}, resolver, digests...)
			defer h.Close(ctx)
			Expect(all(pulses(ctx, h, 1, 2, 3), fnOut)).To(Equal([]int64{1, 3, 6}))
		})

		It("Should reset a function's state when its module stage is re-entered", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

trig2 == 1 => m.s
`, map[string]string{"m": counter + `

export sequence s {
    stage a {
        trig -> count() -> fn_out
        trig2 == 2 => next
    }
    stage b {
        trig2 == 3 => a
    }
}`}, resolver, digests...)
			defer h.Close(ctx)
			send(ctx, h, trig2, 1)
			Expect(all([]telem.Frame[uint32]{
				send(ctx, h, trig, 1),
				send(ctx, h, trig, 1),
			}, fnOut)).To(Equal([]int64{1, 2}))
			send(ctx, h, trig2, 2)
			send(ctx, h, trig2, 3)
			Expect(all([]telem.Frame[uint32]{send(ctx, h, trig, 1)}, fnOut)).
				To(Equal([]int64{1}))
		})

		It("Should keep a module sequence's state across stage re-entry", func(
			ctx SpecContext,
		) {
			h := newModuleRuntimeHarness(ctx, `
import m

trig2 == 1 => m.s
`, map[string]string{"m": `
export sequence s {
    x i64 $= 0
    stage a {
        x = x + 1
        x -> fn_out
        trig2 == 2 => next
    }
    stage b {
        trig2 == 3 => a
    }
}`}, resolver, digests...)
			defer h.Close(ctx)
			first := last(send(ctx, h, trig2, 1), fnOut).Unmarshal[int64]()
			Expect(first[len(first)-1]).To(Equal(int64(1)))
			Expect(send(ctx, h, trig2, 2).Get(fnOut).Series).To(BeEmpty())
			again := send(ctx, h, trig2, 3)
			Expect(all([]telem.Frame[uint32]{again}, fnOut)).To(Equal([]int64{2}))
		})
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
