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
	It("Should hold a live input that stops through the next primary frame, and drop "+
		"each stamp of the frames after it", func() {
		// Two 1 kHz inputs in frames of 40 samples, the other's frame after the
		// primary's. The other input stops after frame 3.
		const frame, frames, stop = 40, 7, 4
		j := MustSucceed(join.New(join.Config{
			DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
			Live:      true,
		}))
		var out join.Output
		dropped := make([]int, 0, frames)
		for f := range frames {
			ps, vs := make([]float64, frame), make([]float64, frame)
			for k := range frame {
				ps[k] = float64(f*frame+k) / 1000
				vs[k] = float64(f*frame + k)
			}
			Expect(j.Push(0, values(vs...), stamps(telem.Alignment(f*frame), ps...))).
				To(Succeed())
			before := out.Dropped
			j.Next(telem.TimeStamp(ps[frame-1]*float64(telem.Second)), &out)
			dropped = append(dropped, out.Dropped-before)
			if f < stop {
				others := make([]float64, frame)
				for k := range frame {
					others[k] = ps[k] + 0.0005
				}
				Expect(j.Push(1, values(vs...), stamps(0, others...))).To(Succeed())
			}
		}
		Expect(dropped).To(Equal([]int{frame, 0, 0, 0, 0, frame, frame}))
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
	DescribeTable("Should keep the hold limit up to a largest interval of 2 times the "+
		"median, and drop it past that",
		func(largest float64, held bool) {
			// 15 intervals of 1 s, then one of largest: the median is 1 s.
			ss := make([]float64, 0, 17)
			for i := range 16 {
				ss = append(ss, float64(i))
			}
			ss = append(ss, 15+largest)
			j := newJoin(2, 0, 0)
			Expect(j.Push(1, values(make([]float64, len(ss))...), stamps(0, ss...))).
				To(Succeed())
			last := ss[len(ss)-1]
			Expect(j.Push(0, values(1), stamps(0, last+4))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, _ := samples(out)
			Expect(got).To(HaveLen(map[bool]int{true: 1, false: 0}[held]))
		},
		Entry("at 2 times: a limit of 3 s, so 4 s past drops", 2.0, false),
		Entry("past 2 times: no limit, so 4 s past holds", 2.5, true),
	)
})
