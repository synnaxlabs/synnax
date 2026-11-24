// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package join_test

import (
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// stepped returns s with an alignment multiple of m.
func stepped(s telem.Series, m uint32) telem.Series {
	s.AlignmentMultiple = m
	return s
}

var _ = Describe("Join check", func() {
	DescribeTable(
		"Should refuse each push that Push documents as invalid",
		func(i int, data, t telem.Series, msg string) {
			j := newJoin(2, 0, 0)
			Expect(j.Push(i, data, t)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("an input that does not exist", 2, values(1), stamps(0, 1),
			"input 2 does not exist"),
		Entry("stamps that are not timestamps", 1, values(1), values(1),
			"stamps of input 1 have data type float64, not timestamp"),
		Entry("a primary whose alignment steps by more than 1", 0, values(1),
			stepped(stamps(0, 1), 2), "primary input 0 steps its alignment by 2"),
	)

	It("Should leave an input as it was when a Push fails part way", func() {
		j := newJoin(2, 0, 0)
		Expect(j.Push(1, values(10, 20, 30), stamps(0, 1, 2, 1.5))).
			To(MatchError(ContainSubstring("does not follow")))
		Expect(j.Push(1, values(40), stamps(3, 3))).To(Succeed())
		Expect(j.Push(0, values(1, 2), stamps(0, 1, 3))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStampMax, &out)
		got, _ := samples(out)
		Expect(got).To(Equal([]sample{{3, 1, []float64{2, 40}}}))
		Expect(out.Dropped).To(Equal(1))
	})

	It("Should refuse a primary stamp before one that Next passed", func() {
		j := newJoin(1, 0, 0)
		Expect(j.Push(0, values(1, 2), stamps(0, 1, 2))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStamp(2*telem.Second), &out)
		Expect(j.Push(0, values(3), stamps(2, 1))).
			To(MatchError(ContainSubstring("does not follow")))
	})

	It("Should start a new run when the alignments of the primary jump", func() {
		j := newJoin(2, 0, 0)
		Expect(j.Push(1, values(10, 20, 30), stamps(0, 1, 2, 3))).To(Succeed())
		Expect(j.Push(0, values(1, 2), stamps(0, 1, 2))).To(Succeed())
		Expect(j.Push(0, values(3), stamps(100, 3))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStampMax, &out)
		got, runs := samples(out)
		Expect(got).To(Equal([]sample{
			{1, 0, []float64{1, 10}},
			{2, 1, []float64{2, 20}},
			{3, 100, []float64{3, 30}},
		}))
		Expect(runs).To(Equal([]int{0, 0, 1}))
	})

	It("Should drop the oldest intervals from the ring", func() {
		// 16 intervals of 1 s, then 9 of 10 s: the median of the last 16 is 10 s, so
		// the limit is 30 s and a stamp 24 s after the last sample holds it. The
		// median of the first 16, or of all 25, is 1 s.
		ss := make([]float64, 0, 26)
		for i := range 17 {
			ss = append(ss, float64(i))
		}
		for i := range 9 {
			ss = append(ss, float64(26+10*i))
		}
		j := newJoin(2, 0, 0)
		Expect(j.Push(1, values(make([]float64, len(ss))...), stamps(0, ss...))).
			To(Succeed())
		Expect(j.Push(0, values(1), stamps(0, 130))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStampMax, &out)
		got, _ := samples(out)
		Expect(got).To(Equal([]sample{{130, 0, []float64{1, 0}}}))
	})
	It("Should hold a live input that stops for 3 of its frame gaps, and drop each "+
		"stamp of the frames after that", func() {
		// Two 1 kHz inputs in frames of 40 samples. Each primary frame arrives at the
		// end of its 40 ms, and the other's frame 20 ms after it. The other input's
		// last frame arrives at 180 ms, and its frames come 40 ms apart, so it holds
		// for up to 120 ms after that: the primary frames that arrive at 200 ms to 280
		// ms hold, and those at 320 ms and 360 ms drop. Frame 0 has no other input, and
		// at frame 1 it has no frame gap yet.
		const frame, frames, stop = 40, 9, 4
		var now telem.TimeStamp
		j := MustSucceed(join.New(join.Config{
			DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
			Live:      true,
			Now:       func() telem.TimeStamp { return now },
		}))
		var out join.Output
		dropped := make([]int, 0, frames)
		for f := range frames {
			ps, vs := make([]float64, frame), make([]float64, frame)
			for k := range frame {
				ps[k] = float64(f*frame+k) / 1000
				vs[k] = float64(f*frame + k)
			}
			now = telem.TimeStamp(40*(f+1)) * telem.MillisecondTS
			Expect(j.Push(0, values(vs...), stamps(telem.Alignment(f*frame), ps...))).
				To(Succeed())
			before := out.Dropped
			j.Next(telem.TimeStamp(ps[frame-1]*float64(telem.Second)), &out)
			dropped = append(dropped, out.Dropped-before)
			if f < stop {
				now += 20 * telem.MillisecondTS
				others := make([]float64, frame)
				for k := range frame {
					others[k] = ps[k] + 0.0005
				}
				Expect(j.Push(1, values(vs...), stamps(0, others...))).To(Succeed())
			}
		}
		Expect(dropped).To(Equal([]int{frame, frame, 0, 0, 0, 0, 0, frame, frame}))
	})

	It("Should fail a history push that puts an input past 65,536 samples after its "+
		"held one, and name the input", func() {
		const most = 1 << 16
		ss := make([]float64, most+1)
		for i := range ss {
			ss[i] = float64(i) / 1000
		}
		history := newJoin(2, 0, 0)
		full := values(make([]float64, most)...)
		Expect(history.Push(1, full, stamps(0, ss[:most]...))).To(Succeed())
		Expect(history.Push(1, values(0), stamps(0, ss[most]))).
			To(MatchError(ContainSubstring("input 1")))
		live := MustSucceed(join.New(join.Config{
			DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
			Live:      true,
		}))
		Expect(live.Push(1, values(make([]float64, most+1)...), stamps(0, ss...))).
			To(Succeed())
	})
	DescribeTable("Should keep the hold limit while 3 in 4 of the last intervals lie "+
		"within a factor of 2 of their median, and drop it below that",
		func(spans []float64, held bool) {
			ss := make([]float64, 0, len(spans)+1)
			ss = append(ss, 0)
			for _, span := range spans {
				ss = append(ss, ss[len(ss)-1]+span)
			}
			j := newJoin(2, 0, 0)
			Expect(j.Push(1, values(make([]float64, len(ss))...), stamps(0, ss...))).
				To(Succeed())
			last := ss[len(ss)-1]
			Expect(j.Push(0, values(1), stamps(0, last+100))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, _ := samples(out)
			Expect(got).To(HaveLen(map[bool]int{true: 1, false: 0}[held]))
		},
		Entry("12 of 16 within: steady, so 100 s past drops",
			[]float64{1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 1, 1, 1}, false),
		Entry("11 of 16 within: no limit, so 100 s past holds",
			[]float64{1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 3, 1, 1}, true),
		Entry("3 of 4 within: steady", []float64{1, 1, 1, 5}, false),
		Entry("2 of 4 within: no limit", []float64{1, 1, 5, 5}, true),
	)

	It(
		"Should drop stamps long after an input stops that had one gap before it "+
			"stopped",
		func() {
			// A 1 Hz input with one 3 s gap among its last 16 intervals stops at 18 s.
			ss := []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 13, 14, 15, 16, 17, 18}
			j := newJoin(2, 0, 0)
			Expect(j.Push(1, values(make([]float64, len(ss))...), stamps(0, ss...))).
				To(Succeed())
			Expect(j.Push(0, values(1, 2), stamps(0, 19, 1000))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			Expect(out.Dropped).To(Equal(1))
		},
	)

	It("Should hold an input of Poisson stamps with no limit", func() {
		for seed := range uint64(8) {
			r := rand.New(rand.NewPCG(seed, 1))
			ss := make([]float64, 0, 33)
			ss = append(ss, 0)
			for range 32 {
				ss = append(ss, ss[len(ss)-1]+r.ExpFloat64())
			}
			j := newJoin(2, 0, 0)
			Expect(j.Push(1, values(make([]float64, len(ss))...), stamps(0, ss...))).
				To(Succeed())
			Expect(j.Push(0, values(1), stamps(0, ss[len(ss)-1]+1000))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			Expect(out.Dropped).To(BeZero(), "seed %d", seed)
		}
	})
})
