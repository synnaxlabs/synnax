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
	"github.com/synnaxlabs/arc/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// stepped returns s with an alignment multiple of m.
func stepped(s telem.Series, m uint32) telem.Series {
	s.AlignmentMultiple = m
	return s
}

var _ = Describe("Join", func() {
	Describe("Push", func() {
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

		It("Should fail a history push that puts an input past 65,536 samples after "+
			"its held one, and name the input", func() {
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
			var now telem.TimeStamp
			live := MustSucceed(join.New(join.Config{
				Inputs: floatInputs(2),
				Live:   true,
				Now:    at(&now),
			}))
			Expect(live.Push(1, values(make([]float64, most+1)...), stamps(0, ss...))).
				To(Succeed())
		})
	})
})
