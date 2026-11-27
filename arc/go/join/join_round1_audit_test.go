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
)

var _ = Describe("Join audit, round 1", func() {
	var now telem.TimeStamp
	newLive := func(from telem.TimeStamp, arrivalTimed bool) *join.Join {
		return MustSucceed(join.New(join.Config{
			Inputs: []join.Input{
				{DataType: telem.Float64T},
				{DataType: telem.Float64T, ArrivalTimed: arrivalTimed},
			},
			From: from,
			Live: true,
			Now:  at(&now),
		}))
	}

	It("Should hold a loaded sample for each primary stamp within one hold limit "+
		"of its own stamp", func() {
		j := newLive(secs(18), false)
		stored := make([]float64, 17)
		for i := range stored {
			stored[i] = float64(i)
		}
		Expect(j.Load(1, values(stored...), stamps(0, stored...))).To(Succeed())
		Expect(j.Push(0, values(1, 2, 3), stamps(0, 18, 19, 20))).To(Succeed())
		var out join.Output
		j.Next(secs(20), &out)
		// The limit is 3 s: 18 s and 19 s hold the sample at 16 s, and 20 s drops.
		Expect(out.Dropped).To(Equal(1))
	})

	It("Should count no loaded primary samples as a frame when it settles "+
		"(Join.Load: loaded samples count as no frame)", func() {
		j := newLive(secs(100), false)
		now = 0
		Expect(j.Load(0, values(1, 2, 3), stamps(0, 1, 2, 3))).To(Succeed())
		var out join.Output
		for _, s := range []float64{100, 101, 102} {
			now = secs(s)
			Expect(j.Push(0, values(s), stamps(telem.Alignment(s), s))).To(Succeed())
			out.Reset()
			j.Next(secs(s), &out)
		}
		// Frames at 100, 101, and 102 s: 3 times the median gap is 3 s, and only 2 s
		// have passed since the first frame.
		Expect(out.Settled).To(BeFalse())
	})

	It("Should drop an arrival input that stopped when its pushes share one arrival "+
		"(Input.ArrivalTimed: it holds while its last push arrived within 3 times the "+
		"median gap)", func() {
		j := newLive(0, true)
		now = 0
		for range 3 {
			Expect(j.Push(1, values(1), telem.Series{})).To(Succeed())
		}
		var out join.Output
		for s := 1; s <= 100; s++ {
			now = secs(float64(s))
			Expect(j.Push(0, values(1), stamps(telem.Alignment(s), float64(s)))).
				To(Succeed())
			j.Next(secs(float64(s)), &out)
		}
		Expect(out.Dropped).To(BeNumerically(">", 0))
	})
})
