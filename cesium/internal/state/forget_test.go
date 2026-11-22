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

var _ = Describe("Delete after a switch", func() {
	DescribeTable(
		"Should cut a record that a switch took out of the state",
		func(sealed ...uint64) {
			s := MustSucceed(state.Open(config(&appended{}), sample()))
			Expect(s.Replay(10, []log.Record{
				&log.Seal{Segment: 3, Length: 12, TailCRC: 3},
			})).To(Succeed())
			run(s)
			snap := MustSucceed(s.Snapshot())
			var (
				tags   []state.Tag
				record state.Segment
			)
			for _, ch := range snap.Channels {
				for _, seg := range ch.Segments {
					for _, id := range sealed {
						if seg.ID == id {
							seg.Part = 7
							tags = append(tags, state.Tag{Part: 7, Segment: seg})
						}
					}
					if seg.ID == 3 {
						record = seg
						record.Part = 7
					}
				}
			}
			Expect(s.Parted(5, tags, []state.Range{
				{Key: 5, Oldest: 7, Newest: 7},
				{Key: 6, Oldest: 7, Newest: 7},
			}, func() {})).To(Succeed())
			MustSucceed(s.Snapshot())
			Expect(s.Channel(6).Segments).To(BeEmpty())
			d := MustSucceed(s.BeginDelete(6, 15, 25, state.Preload{
				Key: 6, Generation: 5, Segments: []state.Segment{record},
			}))
			Expect(s.Channel(6).Segments).To(Equal([]state.Segment{record}))
			Expect(s.Durable(11, []log.Record{&log.Tombstone{
				Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
			}})).To(Succeed())
			Expect(d.Release()).To(Succeed())
			snap = MustSucceed(s.Snapshot())
			Expect(snap.Kills).To(Equal([]state.Kill{
				{Key: 6, Segment: 3, Generation: 1},
			}))
			seg := s.Channel(6).Segments[0]
			Expect(seg.Part).To(BeZero())
			Expect(seg.Pointers).To(Equal([]state.Pointer{
				pointer(1, 0, 10, 10, 0, 4, 1),
				pointer(1, 2, 30, 30, 8, 4, 1),
			}))
		},
		Entry("when the switch builds the segment map again", uint64(2), uint64(3)),
		Entry("when the switch takes each segment out of the map", uint64(3)),
	)
})

var _ = Describe("Put-back after a kept switch", func() {
	It("Should take out the put-back records of a delete that a switch kept, at the "+
		"switch after its release", func() {
		head := sample()
		head.Channels[1] = &state.Channel{
			Key:   6,
			Parts: &state.Parts{Oldest: 7, Newest: 7},
		}
		cfg := config(&appended{})
		cfg.Generation = 5
		s := MustSucceed(state.Open(cfg, head))
		run(s)
		record := state.Segment{
			ID: 3, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 12,
			TailCRC: 3, Part: 7,
			Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 12, 3)},
		}
		parted := func() {
			GinkgoHelper()
			Expect(s.Parted(6, nil, []state.Range{{Key: 6, Oldest: 7, Newest: 7}},
				func() {})).To(Succeed())
			MustSucceed(s.Snapshot())
		}
		d := MustSucceed(s.BeginDelete(6, 15, 25, state.Preload{
			Key: 6, Generation: 5, Segments: []state.Segment{record},
		}))
		parted()
		Expect(s.Channel(6).Segments).To(HaveLen(1))
		Expect(d.Release()).To(Succeed())
		parted()
		Expect(s.Channel(6).Segments).To(BeEmpty())
	})
})

var _ = Describe("Snapshot of parts", func() {
	It("Should hold a key that only parts hold, with its parts", func() {
		head := sample()
		head.Channels[1] = &state.Channel{
			Key:   6,
			Parts: &state.Parts{Oldest: 7, Newest: 7},
		}
		cfg := config(&appended{})
		cfg.Generation = 5
		s := MustSucceed(state.Open(cfg, head))
		run(s)
		snap := MustSucceed(s.Snapshot())
		Expect(snap.Channels).To(HaveLen(2))
		Expect(snap.Channels[1].Key).To(Equal(uint32(6)))
		Expect(snap.Channels[1].Segments).To(BeEmpty())
		Expect(snap.Channels[1].Parts).To(Equal(&state.Parts{
			Oldest: 7, Newest: 7, Generation: 5,
		}))
	})
})
