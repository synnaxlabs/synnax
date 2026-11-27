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
