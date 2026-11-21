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
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Part tags", func() {
	var s *state.State
	BeforeEach(func() {
		s = MustSucceed(state.Open(config(&appended{}), sample()))
		Expect(s.Replay(10, []log.Record{
			&log.Seal{Segment: 3, Length: 12, TailCRC: 3},
		})).To(Succeed())
		run(s)
	})
	// segment returns the visible record of segment 3 of key 6 once the owner ran every
	// queued request.
	segment := func() state.Segment {
		GinkgoHelper()
		MustSucceed(s.Snapshot())
		return s.Channel(6).Segments[0]
	}

	It("Should tag a sealed segment whose record its part holds", func() {
		Expect(s.Parted([]state.Tag{{Part: 7, Segment: segment()}})).To(Succeed())
		Expect(segment().Part).To(Equal(uint64(7)))
	})

	It("Should leave a segment whose record differs from its part's", func() {
		other := segment()
		other.TailCRC++
		Expect(s.Parted([]state.Tag{{Part: 7, Segment: other}})).To(Succeed())
		Expect(segment().Part).To(BeZero())
	})

	It("Should clear the tag of a segment that a tombstone cuts, so a reader takes "+
		"its own pointers", func() {
		Expect(s.Parted([]state.Tag{{Part: 7, Segment: segment()}})).To(Succeed())
		Expect(segment().Part).To(Equal(uint64(7)))
		Expect(s.Durable(11, []log.Record{&log.Tombstone{
			Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
		}})).To(Succeed())
		seg := segment()
		Expect(seg.Part).To(BeZero())
		Expect(seg.Pointers).To(Equal([]state.Pointer{
			pointer(1, 0, 10, 10, 0, 4, 1),
			pointer(1, 2, 30, 30, 8, 4, 1),
		}))
	})
})
