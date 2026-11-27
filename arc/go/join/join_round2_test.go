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

// lockstep reads ts in history one round at a time: each round pushes one chunk of
// each input that has more, then calls Next at the earliest newest stamp of the inputs
// that have more. It returns the first failed push, or nil.
func lockstep(j *join.Join, ts [][]telem.TimeStamp, chunk int, out *join.Output) error {
	pos := make([]int, len(ts))
	for {
		pushed := false
		for i := range ts {
			if pos[i] == len(ts[i]) {
				continue
			}
			hi := min(len(ts[i]), pos[i]+chunk)
			t := telem.NewSeries(ts[i][pos[i]:hi])
			t.Alignment = telem.Alignment(pos[i])
			data := telem.NewSeries(make([]float64, hi-pos[i]))
			if err := j.Push(i, data, t); err != nil {
				return err
			}
			pos[i], pushed = hi, true
		}
		if !pushed {
			break
		}
		until := telem.TimeStampMax
		for i := range ts {
			if pos[i] < len(ts[i]) {
				until = min(until, ts[i][pos[i]-1])
			}
		}
		j.Next(until, out)
	}
	j.Next(telem.TimeStampMax, out)
	return nil
}

// rate returns n stamps step apart from 0.
func rate(n int, step telem.TimeStamp) []telem.TimeStamp {
	ts := make([]telem.TimeStamp, n)
	for k := range ts {
		ts[k] = telem.TimeStamp(k) * step
	}
	return ts
}

var _ = Describe("Join check, round 2", func() {
	It("Should refuse a load of the primary after its first live frame "+
		"(Join.Load: only before the input's first live frame)", func() {
		var now telem.TimeStamp
		j := MustSucceed(join.New(join.Config{
			Inputs: floatInputs(2),
			Live:   true,
			Now:    at(&now),
		}))
		now = secs(10)
		Expect(j.Push(0, values(1), stamps(0, 10))).To(Succeed())
		Expect(j.Load(0, values(2), stamps(1, 11))).
			To(MatchError(ContainSubstring("input 0 loads after its first live frame")))
	})

	It("Should never reach the cap in a paced history read when the primary ends "+
		"before another input (Push: a read that pushes at most 65,536 samples at "+
		"once and calls Next at the earliest newest stamp never reaches the cap)",
		func() {
			primary := rate(11, telem.TimeStamp(telem.Second))
			input := rate(100_001, ms)
			j := newJoin(2, 0, 0)
			var out join.Output
			Expect(pace(j, [][]telem.TimeStamp{primary, input}, 4096, &out)).
				To(Succeed())
			Expect(out.Dropped).To(BeZero())
			got, _ := samples(out)
			Expect(got).To(HaveLen(len(primary)))
		})

	It("Should never reach the cap in a history read that pushes one chunk of each "+
		"input per round and calls Next at the earliest newest stamp (Push: such a "+
		"read never reaches the cap)", func() {
		// Both inputs span 262 s: the primary at 1 kHz, the other at 500 Hz.
		primary := rate(262_144, ms)
		input := rate(131_072, 2*ms)
		j := newJoin(2, 0, 0)
		var out join.Output
		Expect(lockstep(j, [][]telem.TimeStamp{primary, input}, 1<<16, &out)).
			To(Succeed())
		Expect(out.Dropped).To(BeZero())
	})

	It("Should drop a primary stamp far past an input's hold limit when the "+
		"stamps lie more than 2^63 ns apart (hold limit: 3 times the median "+
		"interval)", func() {
		// The input's intervals are 1 s, so its limit is 3 s. The primary stamp
		// lies about 9.4e18 ns after the input's last sample.
		const base = telem.TimeStamp(-4_700_000_000_000_000_000)
		sec := telem.TimeStamp(telem.Second)
		j := newJoin(2, 0, base)
		Expect(j.Push(1, values(1, 2, 3), rawStamps(0, base, base+sec, base+2*sec))).
			To(Succeed())
		Expect(j.Push(0, values(9), rawStamps(0, 4_700_000_000_000_000_000))).
			To(Succeed())
		var out join.Output
		j.Next(telem.TimeStampMax, &out)
		Expect(out.Dropped).To(Equal(1))
	})
})
