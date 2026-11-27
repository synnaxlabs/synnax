// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/state"
)

var _ = Describe("Channel", func() {
	It("Should give the sealed and open segments in rising order of identity", func() {
		ch := state.NewChannel(
			5,
			state.Segment{ID: 1, Sealed: true},
			state.Segment{ID: 2},
			state.Segment{ID: 3, Sealed: true},
			state.Segment{ID: 4, Sealed: true},
			state.Segment{ID: 6},
		)
		Expect(ch.Key).To(Equal(uint32(5)))
		Expect(ch.Len()).To(Equal(5))
		Expect(segments(ch)).To(Equal([]state.Segment{
			{ID: 1, Sealed: true},
			{ID: 2},
			{ID: 3, Sealed: true},
			{ID: 4, Sealed: true},
			{ID: 6},
		}))
	})

	It("Should stop at the first segment that the caller declines", func() {
		ch := state.NewChannel(5, state.Segment{ID: 1}, state.Segment{ID: 2})
		var ids []uint64
		for seg := range ch.All() {
			ids = append(ids, seg.ID)
			if seg.ID == 1 {
				break
			}
		}
		Expect(ids).To(Equal([]uint64{1}))
	})

	It("Should hold no segment when it has none", func() {
		ch := state.NewChannel(5)
		Expect(ch.Len()).To(BeZero())
		Expect(segments(ch)).To(BeEmpty())
	})
})
