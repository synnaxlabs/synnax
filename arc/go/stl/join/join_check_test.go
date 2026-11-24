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
)

var _ = Describe("Join check", func() {
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

	It("Should pair an input on the primary's clock by alignment, when its frame "+
		"comes after the primary's", func() {
		j := newJoin(2, 0, 0)
		var out join.Output
		Expect(j.Push(1, values(10), stamps(0, 1))).To(Succeed())
		Expect(j.Push(0, values(1), stamps(0, 1))).To(Succeed())
		j.Next(telem.TimeStamp(telem.Second), &out)
		Expect(j.Push(0, values(2), stamps(1, 2))).To(Succeed())
		j.Next(telem.TimeStamp(2*telem.Second), &out)
		Expect(j.Push(1, values(20), stamps(1, 2))).To(Succeed())
		j.Next(telem.TimeStamp(2*telem.Second), &out)
		got, _ := samples(out)
		Expect(got).To(Equal([]sample{
			{1, 0, []float64{1, 10}},
			{2, 1, []float64{2, 20}},
		}))
	})
})
