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
)

var _ = Describe("Join audit, round 1", func() {
	var now telem.TimeStamp
	sec := func(s float64) telem.TimeStamp { return telem.TimeStamp(s * float64(telem.Second)) }
	clock := func() telem.TimeStamp { return now }
	newLive := func(from telem.TimeStamp, arrival []bool) *join.Join {
		return MustSucceed(join.New(join.Config{
			DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
			From:      from,
			Live:      true,
			Arrival:   arrival,
			Now:       clock,
		}))
	}

	It("Should hold a seed for a whole primary frame whose first stamp lies within "+
		"one hold limit of the seed (Config.Live doc)", func() {
		j := newLive(sec(18), nil)
		seed := make([]float64, 17)
		for i := range seed {
			seed[i] = float64(i)
		}
		Expect(j.Push(1, values(seed...), stamps(0, seed...))).To(Succeed())
		Expect(j.Push(0, values(1, 2, 3), stamps(0, 18, 19, 20))).To(Succeed())
		var out join.Output
		j.Next(sec(20), &out)
		Expect(out.Dropped).To(BeZero())
	})

	It("Should count no primary seed before From as a frame when it settles "+
		"(Config.Live: a push before From seeds the input as no frame)", func() {
		j := newLive(sec(100), nil)
		now = 0
		Expect(j.Push(0, values(1, 2, 3), stamps(0, 1, 2, 3))).To(Succeed())
		var out join.Output
		j.Next(sec(3), &out)
		for _, s := range []float64{100, 101, 102} {
			now = sec(s)
			Expect(j.Push(0, values(s), stamps(telem.Alignment(s), s))).To(Succeed())
			out.Reset()
			j.Next(sec(s), &out)
		}
		// Frames at 100, 101, and 102 s: 3 times the median gap is 3 s, and only 2 s
		// have passed since the first frame.
		Expect(out.Settled).To(BeFalse())
	})

	It("Should drop an arrival input that stopped when its pushes share one arrival "+
		"(Config.Arrival: it holds while its last push arrived within 3 times the "+
		"median gap)", func() {
		j := newLive(0, []bool{false, true})
		now = 0
		for range 3 {
			Expect(j.Push(1, values(1), telem.Series{})).To(Succeed())
		}
		var out join.Output
		for s := 1; s <= 100; s++ {
			now = sec(float64(s))
			Expect(j.Push(0, values(1), stamps(telem.Alignment(s), float64(s)))).
				To(Succeed())
			j.Next(sec(float64(s)), &out)
		}
		Expect(out.Dropped).To(BeNumerically(">", 0))
	})
})
