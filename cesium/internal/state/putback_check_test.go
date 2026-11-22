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
	. "github.com/synnaxlabs/x/testutil"
)

// segmentIDs returns the identities of the segments of ch, in its order.
func segmentIDs(ch *state.Channel) []uint64 {
	ids := make([]uint64, len(ch.Segments))
	for i, seg := range ch.Segments {
		ids[i] = seg.ID
	}
	return ids
}

var _ = Describe("Check: a put-back record among the segments of its key", func() {
	It("Should keep the segments of the key in rising order of identity, as a head "+
		"needs", func() {
		// Key 6 holds segment 3 in the head, and part 7 holds the record of segment 11.
		head := sample()
		head.Channels[1] = &state.Channel{
			Key: 6,
			Segments: []state.Segment{{
				ID: 3, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 12,
				TailCRC:  3,
				Pointers: []state.Pointer{pointer(3, 0, 40, 60, 0, 12, 3)},
			}},
			Parts: &state.Parts{Oldest: 7, Newest: 7},
		}
		cfg := config(&appended{})
		cfg.Generation = 5
		s := MustSucceed(state.Open(cfg, head))
		run(s)
		record := state.Segment{
			ID: 11, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 12,
			TailCRC: 3, Part: 7,
			Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 12, 3)},
		}
		d := MustSucceed(s.BeginDelete(6, 15, 25, state.Preload{
			Key: 6, Generation: 5, Segments: []state.Segment{record},
		}))
		Expect(segmentIDs(s.Channel(6))).To(Equal([]uint64{3, 11}))
		Expect(d.Release()).To(Succeed())
		for _, ch := range MustSucceed(s.Snapshot()).Channels {
			if ch.Key == 6 {
				Expect(segmentIDs(ch)).To(Equal([]uint64{3, 11}))
			}
		}
	})
})
