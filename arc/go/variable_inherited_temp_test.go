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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

// Temp scratch suite for the value-variable node model. Kept for fast iteration.
var _ = Describe("Variables (temp)", func() {
	lastI64 := func(fr telem.Frame[uint32], key uint32) int64 {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[int64](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	trigger := func(h *runtimeHarness, ctx SpecContext, key uint32) {
		h.Ingest(key, telem.NewSeriesV[uint8](1))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
	}

	It("DEBUG increments tick by tick", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    count := 0
			    stage s1 {
			        count + 1 -> count
			        count -> out
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)

		h.Ingest(100, telem.NewSeriesV[uint8](1))
		for i := range 5 {
			fmt.Printf("--- tick %d\n", i+1)
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
			out, _ := h.Flush()
			for _, s := range out.Get(101).Series {
				fmt.Printf("out: %v\n", telem.UnmarshalSeries[int64](s))
			}
		}
	})

	// Declarations are bindings, not steps: they must not block step advance.
	It("advances past sequence-scoped declarations", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    my_var := "my_var"
			    1 -> out
			    other := "other"
			    2 -> out
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(2)))
	})

	// Reassignment in one stage must be visible to a later read in another.
	It("reassigns a sequence-scoped var from a stage", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
			"go2":   {types.U8(), 102},
			"go1":   {types.U8(), 103},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    my_var := 1
			    stage s1 {
			        my_var -> out
			        go2 => next
			    }
			    stage s2 {
			        my_var = 2
			        go1 => s1
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
			channels.Digest{Key: 102, DataType: telem.Uint8T},
			channels.Digest{Key: 103, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(1)))
		trigger(h, ctx, 102)
		trigger(h, ctx, 103)
		out, _ = h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(2)))
	})

	// An expression that reads AND writes the same variable (increment loop).
	It("increments a var from an expression", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
			"go2":   {types.U8(), 102},
			"go1":   {types.U8(), 103},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    count := 0
			    stage s1 {
			        count + 1 -> count
			        count -> out
			        go2 => next
			    }
			    stage s2 {
			        count = count + 1
			        go1 => s1
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
			channels.Digest{Key: 102, DataType: telem.Uint8T},
			channels.Digest{Key: 103, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		collect := func(fr telem.Frame[uint32]) []int64 {
			var got []int64
			for _, s := range fr.Get(101).Series {
				got = append(got, telem.UnmarshalSeries[int64](s)...)
			}
			return got
		}
		// Increments fire once per scope entry (own writes do not re-trigger);
		// reads fire only on unconsumed values.
		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(collect(out)).To(Equal([]int64{0, 1}))
		trigger(h, ctx, 102)
		trigger(h, ctx, 103)
		out, _ = h.Flush()
		Expect(collect(out)).To(Equal([]int64{2, 3}))
	})

	// Re-entry without a reassignment while away: the read must stay silent on
	// entry (already-seen value) and fire once with the new increment.
	It("writes once per re-entry of an increment loop", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
			"go2":   {types.U8(), 102},
			"go1":   {types.U8(), 103},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    count := 0
			    stage s1 {
			        count + 1 -> count
			        count -> out
			        go2 => next
			    }
			    stage s2 {
			        go1 => s1
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
			channels.Digest{Key: 102, DataType: telem.Uint8T},
			channels.Digest{Key: 103, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		collect := func(fr telem.Frame[uint32]) []int64 {
			var got []int64
			for _, s := range fr.Get(101).Series {
				got = append(got, telem.UnmarshalSeries[int64](s)...)
			}
			return got
		}
		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(collect(out)).To(Equal([]int64{0, 1}))
		trigger(h, ctx, 102)
		trigger(h, ctx, 103)
		out, _ = h.Flush()
		Expect(collect(out)).To(Equal([]int64{2}))
	})

	// A := variable declared in the stage re-seeds on each stage re-entry.
	It("re-seeds a stage-scoped var on re-entry", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
			"go2":   {types.U8(), 102},
			"go1":   {types.U8(), 103},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    stage s1 {
			        count := 0
			        count + 1 -> count
			        count -> out
			        go2 => next
			    }
			    stage s2 {
			        go1 => s1
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
			channels.Digest{Key: 102, DataType: telem.Uint8T},
			channels.Digest{Key: 103, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		collect := func(fr telem.Frame[uint32]) []int64 {
			var got []int64
			for _, s := range fr.Get(101).Series {
				got = append(got, telem.UnmarshalSeries[int64](s)...)
			}
			return got
		}
		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(collect(out)).To(Equal([]int64{0, 1}))
		trigger(h, ctx, 102)
		trigger(h, ctx, 103)
		out, _ = h.Flush()
		Expect(collect(out)).To(Equal([]int64{0, 1}))
	})

	// A channel alias rebind: writes follow the current binding across re-entry.
	It("routes alias writes to the rebound channel", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out_a": {types.I64(), 101},
			"out_b": {types.I64(), 102},
			"go2":   {types.U8(), 103},
			"go1":   {types.U8(), 104},
		})
		h := newRuntimeHarness(ctx, `
			l := out_a
			sequence main {
			    stage s1 {
			        1 -> l
			        go2 => next
			    }
			    stage s2 {
			        l = out_b
			        go1 => s1
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
			channels.Digest{Key: 102, DataType: telem.Int64T},
			channels.Digest{Key: 103, DataType: telem.Uint8T},
			channels.Digest{Key: 104, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(1)))
		Expect(out.Get(102).Series).To(BeEmpty())
		trigger(h, ctx, 103)
		trigger(h, ctx, 104)
		out, _ = h.Flush()
		Expect(lastI64(out, 102)).To(Equal(int64(1)))
	})

	It("routes alias reads to the rebound channel", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"in_a":  {types.I64(), 101},
			"in_b":  {types.I64(), 102},
			"out":   {types.I64(), 103},
			"go2":   {types.U8(), 104},
			"go1":   {types.U8(), 105},
		})
		h := newRuntimeHarness(ctx, `
			r := in_a
			sequence main {
			    stage s1 {
			        r -> out
			        go2 => next
			    }
			    stage s2 {
			        r = in_b
			        go1 => s1
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
			channels.Digest{Key: 102, DataType: telem.Int64T},
			channels.Digest{Key: 103, DataType: telem.Int64T},
			channels.Digest{Key: 104, DataType: telem.Uint8T},
			channels.Digest{Key: 105, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		h.Ingest(101, telem.NewSeriesV[int64](10))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		Expect(lastI64(out, 103)).To(Equal(int64(10)))

		trigger(h, ctx, 104)
		trigger(h, ctx, 105)
		h.Ingest(102, telem.NewSeriesV[int64](20))
		h.Ingest(101, telem.NewSeriesV[int64](99))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ = h.Flush()
		Expect(lastI64(out, 103)).To(Equal(int64(20)))
	})

	It("routes alias expression reads to the rebound channel", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"in_a":  {types.I64(), 101},
			"in_b":  {types.I64(), 102},
			"out":   {types.I64(), 103},
			"go2":   {types.U8(), 104},
			"go1":   {types.U8(), 105},
		})
		h := newRuntimeHarness(ctx, `
			r := in_a
			sequence main {
			    stage s1 {
			        r * 2 -> out
			        go2 => next
			    }
			    stage s2 {
			        r = in_b
			        go1 => s1
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
			channels.Digest{Key: 102, DataType: telem.Int64T},
			channels.Digest{Key: 103, DataType: telem.Int64T},
			channels.Digest{Key: 104, DataType: telem.Uint8T},
			channels.Digest{Key: 105, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		h.Ingest(101, telem.NewSeriesV[int64](10))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		Expect(lastI64(out, 103)).To(Equal(int64(20)))

		trigger(h, ctx, 104)
		trigger(h, ctx, 105)
		h.Ingest(102, telem.NewSeriesV[int64](30))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ = h.Flush()
		Expect(lastI64(out, 103)).To(Equal(int64(60)))
	})

	It("routes a top-level alias expression read across a stage rebind", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"in_a":  {types.I64(), 101},
			"in_b":  {types.I64(), 102},
			"out":   {types.I64(), 103},
			"go2":   {types.U8(), 104},
		})
		h := newRuntimeHarness(ctx, `
			r := in_a
			r * 2 -> out
			sequence main {
			    stage s1 {
			        go2 => next
			    }
			    stage s2 {
			        r = in_b
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
			channels.Digest{Key: 102, DataType: telem.Int64T},
			channels.Digest{Key: 103, DataType: telem.Int64T},
			channels.Digest{Key: 104, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		h.Ingest(101, telem.NewSeriesV[int64](10))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		Expect(lastI64(out, 103)).To(Equal(int64(20)))

		trigger(h, ctx, 104)
		h.Ingest(102, telem.NewSeriesV[int64](30))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ = h.Flush()
		Expect(lastI64(out, 103)).To(Equal(int64(60)))
	})

	It("DEBUG reactive var IR channels", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"cpu": {types.F64(), 201},
			"log": {types.String(), 202},
		})
		stlSyms := stl.NewSymbols()
		ambient := make([]*symbol.Symbol, 0, len(stlSyms)+len(resolver))
		ambient = append(ambient, stlSyms...)
		for i := range resolver {
			ambient = append(ambient, &resolver[i])
		}
		p, err := arc.CompileText(ctx, arc.Text{Raw: `
			stage main {
			    my_var := "Init: " + str(cpu)
			    my_var -> log
			    sequence {
			        "change now" -> log
			        my_var = "First: " + str(cpu)
			        my_var = "Second: " + str(cpu)
			    }
			}`}, symbol.NewRoot(nil, ambient))
		if err != nil {
			fmt.Printf("compile error: %v\n", err)
			return
		}
		for _, n := range p.IR.Nodes {
			fmt.Printf("node %s type=%s read=%v write=%v\n", n.Key, n.Type, n.Channels.Read, n.Channels.Write)
			for _, in := range n.Inputs {
				if in.Value != nil {
					fmt.Printf("  input %s value=%v\n", in.Name, in.Value)
				}
			}
		}
	})

	// A format string interpolating a reassigned variable.
	It("interpolates a reassigned var in a format string", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.String(), 101},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    stage s1 {
			        count := 0
			        count + 1 -> count
			        f"{count}" -> out
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.StringT},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		out, _ := h.Flush()
		ch := out.Get(101)
		Expect(ch.Series).ToNot(BeEmpty())
		last := ch.Series[len(ch.Series)-1]
		Expect(string(last.At(-1))).To(Equal("1"))
	})

	// A variable declared in the enclosing sequence, read in a child stage.
	It("reads a sequence-scoped var from a child stage", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    count := 0
			    stage s1 {
			        count + 1 -> out
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(1)))
	})

	// A flow writing a variable declared in the same stage.
	It("writes a stage-scoped var declared in the same stage", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    stage s1 {
			        count := 0
			        count + 1 -> count
			        count -> out
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(1)))
	})

	// A variable declared in the same stage that reads it.
	It("reads a stage-scoped var declared in the same stage", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start": {types.U8(), 100},
			"out":   {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx, `
			sequence main {
			    stage s1 {
			        count := 0
			        count + 1 -> out
			    }
			}
			start => main`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)

		trigger(h, ctx, 100)
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(1)))
	})
})
