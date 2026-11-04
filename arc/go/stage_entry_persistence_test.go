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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

// A persistence check armed on stage entry acts only on samples taken inside the
// stage (SY-5456) and measures its hold in sample time (SY-5457). Telemetry reaches
// the runtime one delivery latency late, so the first batch a stage sees was sampled
// before entry.
//
// Timeline, with T the stamp at which the run stage enters:
//
//	T-34 .. T-14 ms  samples below 300 are taken, during the previous stage
//	T                run stage enters, check armed, step=700 written
//	T+2              that batch is delivered and dropped
//	T+22             a batch taken inside the stage is delivered
//
// A func with a flow-fed input runs once per sample with the sample's stamp. An
// inline expression such as `sensor < 300.0` compiles to a function with no inputs
// that reads the channel's latest value, runs once per batch, and stamps the cycle
// clock, so stable.for behind it measures arrival to arrival.
var _ = Describe("Stage entry persistence on batched telemetry", func() {
	const (
		stepIdxKey   = 99
		stepKey      = 100
		sensorIdxKey = 199
		sensorKey    = 200
		seenKey      = 300
		base         = 10 * telem.SecondTS
		startupWait  = 250 * telem.Millisecond
	)

	// program wraps one run-stage check in the sequence shape under test.
	program := func(check string) string {
		imports := "import time\n"
		if strings.Contains(check, "stable.") {
			imports += "import stable\n"
		}
		return imports + `func low_for_50(v f64) bool {
    count i64 $= 0
    if v < 300.0 {
        count = count + 1
    } else {
        count = 0
    }
    return count >= 50
}
func is_low(v f64) bool {
    return v < 300.0
}
sequence main {
    stage startup {
        600 -> step
        time.wait{250ms} => next
    }
    stage run {
        700 -> step
        sensor -> sensor_seen
        ` + check + `
    }
    stage shutdown {
        900 -> step
    }
}
1 => main`
	}

	resolver := channelSymbols(map[string]channelDef{
		"step":        {types.I64(), stepKey},
		"sensor":      {types.F64(), sensorKey},
		"sensor_seen": {types.F64(), seenKey},
	})

	newHarness := func(ctx SpecContext, check string) *runtimeHarness {
		return newRuntimeHarness(
			ctx,
			program(check),
			resolver,
			channels.Digest{Key: stepIdxKey, DataType: telem.TimestampT},
			channels.Digest{Key: stepKey, DataType: telem.Int64T, Index: stepIdxKey},
			channels.Digest{Key: sensorIdxKey, DataType: telem.TimestampT},
			channels.Digest{
				Key:      sensorKey,
				DataType: telem.Float64T,
				Index:    sensorIdxKey,
			},
			channels.Digest{Key: seenKey, DataType: telem.Float64T},
		)
	}

	stepWrites := func(fr telem.Frame[uint32]) (vals []int64, stamps []telem.TimeStamp) {
		for _, s := range fr.Get(stepKey).Series {
			vals = append(vals, s.Unmarshal[int64]()...)
		}
		for _, s := range fr.Get(stepIdxKey).Series {
			stamps = append(stamps, s.Unmarshal[telem.TimeStamp]()...)
		}
		return vals, stamps
	}

	seenCount := func(fr telem.Frame[uint32]) (n int) {
		for _, s := range fr.Get(seenKey).Series {
			n += len(s.Unmarshal[float64]())
		}
		return n
	}

	// cycle runs one scheduler cycle at wall clock now and returns the flushed writes.
	cycle := func(
		ctx SpecContext,
		h *runtimeHarness,
		now *telem.TimeStamp,
		at telem.TimeStamp,
	) telem.Frame[uint32] {
		*now = at
		h.Tick(ctx, telem.TimeSpan(at-base))
		h.channelState.ClearReads()
		out, _ := h.Flush()
		return out
	}

	// enterRun drives the sequence through startup into run and returns the entry
	// stamp T, the stamp on the step=700 write.
	enterRun := func(
		ctx SpecContext,
		h *runtimeHarness,
		now *telem.TimeStamp,
	) telem.TimeStamp {
		GinkgoHelper()
		vals, _ := stepWrites(cycle(ctx, h, now, base))
		Expect(vals).To(Equal([]int64{600}))
		out := cycle(ctx, h, now, base+telem.TimeStamp(startupWait))
		vals, stamps := stepWrites(out)
		Expect(vals).To(Equal([]int64{700}), "run must enter after the wait")
		return stamps[0]
	}

	// lowBatch is 100 samples at 5 kHz, every one below 300, starting at from.
	lowBatch := func(from telem.TimeStamp) (telem.Series, telem.Series) {
		stamps := make([]telem.TimeStamp, 100)
		values := make([]float64, 100)
		for i := range stamps {
			stamps[i] = from + telem.TimeStamp(i)*200*telem.MicrosecondTS
			values[i] = 255 + float64(i)*0.35
		}
		return telem.NewSeriesV(stamps...), telem.NewSeriesV(values...)
	}

	// staleBatch was taken 34 to 14 ms before entry: the batch the stage sees first,
	// one 16 ms delivery latency late.
	staleBatch := func(entry telem.TimeStamp) (telem.Series, telem.Series) {
		return lowBatch(entry - 34*telem.MillisecondTS)
	}

	expectAbortAt := func(out telem.Frame[uint32], entry, at telem.TimeStamp) {
		GinkgoHelper()
		vals, stamps := stepWrites(out)
		Expect(vals).To(Equal([]int64{900}), "shutdown must have entered")
		Expect(stamps[0] - entry).To(Equal(at - entry))
	}

	expectNoAbort := func(out telem.Frame[uint32]) {
		GinkgoHelper()
		Expect(out.Get(stepKey).Series).To(BeEmpty(), "no abort expected")
	}

	Describe("A sample counter fed per sample", func() {
		const check = `sensor -> low_for_50{} => shutdown`

		It(
			"Ignores the stale batch and trips on one taken inside the stage",
			func(ctx SpecContext) {
				h := newHarness(ctx, check)
				defer h.Close(ctx)
				now := base
				h.clock.Source = func() telem.TimeStamp { return now }
				entry := enterRun(ctx, h, &now)

				stamps, values := staleBatch(entry)
				h.IngestIndexed(sensorIdxKey, stamps, sensorKey, values)
				out := cycle(ctx, h, &now, entry+2*telem.MillisecondTS)
				Expect(seenCount(out)).To(Equal(0), "pre-entry samples are dropped")
				expectNoAbort(out)

				stamps, values = lowBatch(entry)
				h.IngestIndexed(sensorIdxKey, stamps, sensorKey, values)
				out = cycle(ctx, h, &now, entry+22*telem.MillisecondTS)
				Expect(seenCount(out)).To(Equal(100))
				expectAbortAt(out, entry, entry+22*telem.MillisecondTS)
			},
		)
	})

	Describe("stable.for fed per sample", func() {
		const check = `sensor -> is_low{} -> stable.for{10ms} => shutdown`

		It(
			"Ignores the stale batch and trips on one taken inside the stage",
			func(ctx SpecContext) {
				h := newHarness(ctx, check)
				defer h.Close(ctx)
				now := base
				h.clock.Source = func() telem.TimeStamp { return now }
				entry := enterRun(ctx, h, &now)

				stamps, values := staleBatch(entry)
				h.IngestIndexed(sensorIdxKey, stamps, sensorKey, values)
				out := cycle(ctx, h, &now, entry+2*telem.MillisecondTS)
				Expect(seenCount(out)).To(Equal(0), "pre-entry samples are dropped")
				expectNoAbort(out)

				// 20 ms of low samples taken inside the stage prove a 10 ms hold.
				stamps, values = lowBatch(entry)
				h.IngestIndexed(sensorIdxKey, stamps, sensorKey, values)
				out = cycle(ctx, h, &now, entry+22*telem.MillisecondTS)
				Expect(seenCount(out)).To(Equal(100))
				expectAbortAt(out, entry, entry+22*telem.MillisecondTS)
			},
		)

		It(
			"Does not trip on 2 ms of low samples delivered 16 ms late",
			func(ctx SpecContext) {
				h := newHarness(ctx, check)
				defer h.Close(ctx)
				now := base
				h.clock.Source = func() telem.TimeStamp { return now }
				entry := enterRun(ctx, h, &now)

				// The data shows 2 ms below 300. The 16 ms on the wall clock since the
				// first sample does not count.
				stamps := telem.NewSeriesV(
					entry,
					entry+1*telem.MillisecondTS,
					entry+2*telem.MillisecondTS,
				)
				values := telem.NewSeriesV[float64](290, 292, 294)
				h.IngestIndexed(sensorIdxKey, stamps, sensorKey, values)
				out := cycle(ctx, h, &now, entry+18*telem.MillisecondTS)
				Expect(seenCount(out)).To(Equal(3))
				expectNoAbort(out)
			},
		)

		It("Does not trip on a batch above the threshold", func(ctx SpecContext) {
			h := newHarness(ctx, check)
			defer h.Close(ctx)
			now := base
			h.clock.Source = func() telem.TimeStamp { return now }
			entry := enterRun(ctx, h, &now)

			stamps := telem.NewSeriesV(
				entry,
				entry+2*telem.MillisecondTS,
				entry+4*telem.MillisecondTS,
				entry+6*telem.MillisecondTS,
			)
			values := telem.NewSeriesV[float64](313, 316, 320, 323)
			h.IngestIndexed(sensorIdxKey, stamps, sensorKey, values)
			out := cycle(ctx, h, &now, entry+22*telem.MillisecondTS)
			Expect(seenCount(out)).To(Equal(4))
			expectNoAbort(out)
		})
	})

	Describe("stable.for behind an inline expression", func() {
		const check = `sensor < 300.0 -> stable.for{10ms} => shutdown`

		It(
			"Sees one cycle-stamped value per batch, so it trips 10 ms of arrivals later",
			func(ctx SpecContext) {
				h := newHarness(ctx, check)
				defer h.Close(ctx)
				now := base
				h.clock.Source = func() telem.TimeStamp { return now }
				entry := enterRun(ctx, h, &now)

				// Two samples below 300, 1 ms apart, stamped at the cycle, T+2.
				stamps := telem.NewSeriesV(entry, entry+1*telem.MillisecondTS)
				values := telem.NewSeriesV[float64](290, 292)
				h.IngestIndexed(sensorIdxKey, stamps, sensorKey, values)
				out := cycle(ctx, h, &now, entry+2*telem.MillisecondTS)
				Expect(seenCount(out)).To(Equal(2))
				expectNoAbort(out)

				// The next batch lands 10 ms of wall clock later, stamped T+12.
				stamps = telem.NewSeriesV(
					entry+8*telem.MillisecondTS,
					entry+10*telem.MillisecondTS,
				)
				values = telem.NewSeriesV[float64](294, 296)
				h.IngestIndexed(sensorIdxKey, stamps, sensorKey, values)
				out = cycle(ctx, h, &now, entry+12*telem.MillisecondTS)
				expectAbortAt(out, entry, entry+12*telem.MillisecondTS)
			},
		)
	})
})
