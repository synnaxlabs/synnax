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

	// of returns the visible record of segment i of key once the owner ran every queued
	// request.
	of := func(key uint32, i int) state.Segment {
		GinkgoHelper()
		MustSucceed(s.Snapshot())
		return s.Channel(key).Segments[i]
	}

	It("Should take a tagged sealed segment out of the state, and keep an open one "+
		"that its part does not hold", func() {
		sealedID, openID := of(5, 0).ID, of(5, 1).ID
		Expect(s.Parted(1, []state.Tag{
			{Part: 7, Segment: of(5, 0)},
			{Part: 7, Segment: of(5, 1)},
		}, nil, func() {})).To(Succeed())
		segs := MustSucceed(s.Snapshot()).Channels[0].Segments
		Expect(segs).To(HaveLen(1))
		Expect(segs[0].ID).To(Equal(openID))
		Expect(segs[0].ID).ToNot(Equal(sealedID))
		Expect(segs[0].Part).To(BeZero())
	})

	It("Should leave an open segment", func() {
		open := of(5, 1)
		Expect(open.Sealed).To(BeFalse())
		Expect(
			s.Parted(1, []state.Tag{{Part: 7, Segment: open}}, nil, func() {}),
		).To(Succeed())
		Expect(of(5, 1).Part).To(BeZero())
	})

	It("Should skip a tag whose segment left the state", func() {
		gone := of(5, 0)
		gone.ID = 99
		Expect(s.Parted(1, []state.Tag{
			{Part: 7, Segment: gone},
			{Part: 8, Segment: segment()},
		}, nil, func() {})).To(Succeed())
		Expect(of(5, 0).Part).To(BeZero())
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Segments).To(BeEmpty())
	})

	It("Should leave the part record of a segment that the state took out live when a "+
		"tombstone does not reach it", func() {
		Expect(
			s.Parted(1, []state.Tag{{Part: 7, Segment: of(5, 0)}}, nil, func() {}),
		).To(Succeed())
		Expect(s.Durable(11, []log.Record{
			&log.Tombstone{Key: 5, Start: 35, End: 60},
		})).To(Succeed())
		snap := MustSucceed(s.Snapshot())
		Expect(snap.Kills).To(BeEmpty())
		Expect(snap.Channels[0].Segments).To(HaveLen(1))
		Expect(snap.Channels[0].Segments[0].Pointers).To(BeEmpty())
	})

	It("Should take out a sealed segment whose record its part holds", func() {
		Expect(
			s.Parted(1, []state.Tag{{Part: 7, Segment: segment()}}, nil, func() {}),
		).To(Succeed())
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Segments).To(BeEmpty())
	})

	It("Should leave a segment whose record differs from its part's", func() {
		other := segment()
		other.TailCRC++
		Expect(
			s.Parted(1, []state.Tag{{Part: 7, Segment: other}}, nil, func() {}),
		).To(Succeed())
		Expect(segment().Part).To(BeZero())
	})
})
