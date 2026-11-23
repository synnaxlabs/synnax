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

var _ = Describe("Preloads", func() {
	var (
		s *state.State
		// record is the record of segment 3 of key 6, which part 7 of head 5 holds.
		record state.Segment
	)
	BeforeEach(func() {
		s = MustSucceed(state.Open(config(&appended{}), sample()))
		Expect(s.Replay(10, []log.Record{
			&log.Seal{Segment: 3, Length: 12, TailCRC: 3},
		})).To(Succeed())
		run(s)
		MustSucceed(s.Snapshot())
		record = s.Channel(6).Segments[0]
		record.Part = 7
		Expect(s.Parted(
			5,
			[]state.Tag{{Part: 7, Segment: record}},
			[]state.Range{{Key: 6, Oldest: 7, Newest: 7}},
			func() {},
		)).To(Succeed())
		MustSucceed(s.Snapshot())
	})
	// writer starts inside segment 3, whose pointer runs from 10 to 30.
	writer := state.WriterConfig{
		Keys:  []state.WriterKey{{Key: 6, DataType: 4}},
		Clock: 5,
		Start: 20,
	}
	preload := func(generation uint64, segs ...state.Segment) state.Preload {
		return state.Preload{Key: 6, Generation: generation, Segments: segs}
	}

	It("Should refuse a writer that starts in a parted segment, through its "+
		"preloaded record", func() {
		Expect(s.OpenWriter(writer, preload(5, record))).Error().To(SatisfyAll(
			conflict(),
			MatchError(ContainSubstring("inside its stored data from 10 to 30")),
		))
	})

	It("Should take the records of a parted segment from the preload alone", func() {
		w := MustSucceed(s.OpenWriter(writer, preload(5)))
		Expect(w.Close()).To(Succeed())
	})

	It("Should refuse a preload from the parts of an older head as stale", func() {
		Expect(s.OpenWriter(writer, preload(4, record))).Error().To(SatisfyAll(
			MatchError(state.ErrStale),
			MatchError(ContainSubstring("key 6 reads the parts of head 5, not 4")),
		))
		Expect(s.BeginDelete(6, 15, 25, preload(4, record))).Error().
			To(MatchError(state.ErrStale))
	})

	It("Should refuse a step with no preload of a key that parts hold", func() {
		Expect(s.OpenWriter(writer)).Error().To(SatisfyAll(
			refused(),
			MatchError(ContainSubstring("no preload of key 6, which parts hold")),
		))
	})

	It("Should find the ends of a delete in the preloaded records of parted segments",
		func() {
			d := MustSucceed(s.BeginDelete(6, 15, 25, preload(5, record)))
			Expect(d.Empty).To(BeFalse())
			Expect(d.Left.Segment.ID).To(Equal(uint64(3)))
			Expect(d.Right.Segment.ID).To(Equal(uint64(3)))
			Expect(d.Release()).To(Succeed())
			d = MustSucceed(s.BeginDelete(6, 15, 25, preload(5)))
			Expect(d.Empty).To(BeTrue())
			Expect(d.Release()).To(Succeed())
		})
})

var _ = Describe("Put-back", func() {
	It("Should put a parted record back in the order of identity of its key", func() {
		head := sample()
		head.Channels[1].Parts = &state.Parts{Oldest: 7, Newest: 7}
		cfg := config(&appended{})
		cfg.Generation = 5
		s := MustSucceed(state.Open(cfg, head))
		run(s)
		record := state.Segment{
			ID: 9, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 8,
			TailCRC: 24, Part: 7,
			Pointers: []state.Pointer{pointer(3, 0, 40, 50, 0, 8, 2)},
		}
		d := MustSucceed(s.BeginDelete(6, 45, 55, state.Preload{
			Key: 6, Generation: 5, Segments: []state.Segment{record},
		}))
		DeferCleanup(func() { Expect(d.Release()).To(Succeed()) })
		MustSucceed(s.Snapshot())
		ids := make([]uint64, 0, 2)
		for _, seg := range s.Channel(6).Segments {
			ids = append(ids, seg.ID)
		}
		Expect(ids).To(Equal([]uint64{3, 9}))
	})

	It("Should put a parted record that the state lacks back for the cut of a delete",
		func() {
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
			d := MustSucceed(s.BeginDelete(6, 15, 25, state.Preload{
				Key: 6, Generation: 5, Segments: []state.Segment{record},
			}))
			Expect(d.Left.Segment.ID).To(Equal(uint64(3)))
			Expect(s.Channel(6).Segments).To(Equal([]state.Segment{record}))
			Expect(s.Durable(10, []log.Record{&log.Tombstone{
				Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
			}})).To(Succeed())
			Expect(d.Release()).To(Succeed())
			snap := MustSucceed(s.Snapshot())
			Expect(
				snap.Kills,
			).To(Equal([]state.Kill{{Key: 6, Segment: 3, Generation: 5}}))
			seg := s.Channel(6).Segments[0]
			Expect(seg.Part).To(BeZero())
			Expect(seg.Pointers).To(Equal([]state.Pointer{
				pointer(1, 0, 10, 10, 0, 4, 1),
				pointer(1, 2, 30, 30, 8, 4, 1),
			}))
		})
})

var _ = Describe("Replay preloads", func() {
	// parted is a sealed record of key 6 that part 7 holds, and that the state lacks.
	parted := state.Segment{
		ID: 3, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 12,
		TailCRC: 3, Part: 7,
		Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 12, 3)},
	}

	It("Should cut a parted record that a replayed Tombstone reaches, through its "+
		"preload", func() {
		head := sample()
		head.Channels[1] = &state.Channel{
			Key:   6,
			Parts: &state.Parts{Oldest: 7, Newest: 7},
		}
		s := MustSucceed(state.Open(config(&appended{}), head))
		Expect(s.Replay(10, []log.Record{&log.Tombstone{
			Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
		}}, state.Preload{Key: 6, Segments: []state.Segment{parted}})).To(Succeed())
		run(s)
		snap := MustSucceed(s.Snapshot())
		Expect(snap.Kills).To(Equal([]state.Kill{{Key: 6, Segment: 3, Generation: 1}}))
		seg := s.Channel(6).Segments[0]
		Expect(seg.Part).To(BeZero())
		Expect(seg.Pointers).To(Equal([]state.Pointer{
			pointer(1, 0, 10, 10, 0, 4, 1),
			pointer(1, 2, 30, 30, 8, 4, 1),
		}))
	})

	It("Should refuse a replayed commit over a parted record that its preload holds",
		func() {
			s := MustSucceed(state.Open(config(&appended{}), sample()))
			over := state.Segment{
				ID: 9, Clock: 5, Version: 1, DataType: 7, Sealed: true, Length: 8,
				Part: 7, Pointers: []state.Pointer{pointer(1, 0, 30, 35, 0, 8, 2)},
			}
			Expect(s.Replay(10, []log.Record{commit(1, 3, 0, 32, 32, slot(4, 20))},
				state.Preload{Key: 5, Segments: []state.Segment{over}})).To(SatisfyAll(
				refused(),
				MatchError(
					ContainSubstring("from 32 to 32 overlaps its pointer from 30 "+
						"to 35"),
				),
			))
		})
})

var _ = Describe("Clock users", func() {
	// user is a sealed record of key 6 on clock 5 that part 7 holds, and that the
	// state lacks.
	user := state.User{Key: 6, Segment: state.Segment{
		ID: 3, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 12,
		TailCRC: 3, Part: 7,
		Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 12, 3)},
	}}
	var s *state.State
	BeforeEach(func() {
		head := sample()
		head.Channels[1] = &state.Channel{
			Key:   6,
			Parts: &state.Parts{Oldest: 7, Newest: 7},
		}
		s = MustSucceed(state.Open(config(&appended{}), head))
	})
	users := state.Preload{Key: 5, Generation: 1, Users: []state.User{user}}

	It("Should refuse a replayed Drop of a clock that a parted record of a kept key "+
		"names", func() {
		Expect(s.Replay(10, []log.Record{&log.Drop{Key: 5}}, users)).To(SatisfyAll(
			refused(),
			MatchError(
				ContainSubstring("drop of key 5, the clock of segment 3 of key 6"),
			),
		))
	})

	It("Should drop a clock whose parted users belong to a key that a Drop took out",
		func() {
			Expect(s.Replay(10, []log.Record{
				&log.Drop{Key: 6},
				&log.Drop{Key: 5},
			}, users)).To(Succeed())
		})

	It("Should refuse a replayed Tombstone of a clock in the range of a parted user",
		func() {
			Expect(s.Replay(10, []log.Record{
				&log.Tombstone{Key: 5, Start: 15, End: 25},
			}, users)).To(SatisfyAll(
				refused(),
				MatchError(
					ContainSubstring("tombstone of clock 5 from 15 to 25, which "+
						"key 6 uses from 10 to 30"),
				),
			))
		})

	It("Should refuse the Tombstone of a delete of a clock whose preload names a "+
		"parted user", func() {
		run(s)
		d := MustSucceed(s.BeginDelete(5, 15, 25, users))
		Expect(d.Check(&log.Tombstone{Key: 5, Start: 15, End: 25})).To(SatisfyAll(
			refused(),
			MatchError(ContainSubstring("which key 6 uses from 10 to 30")),
		))
		Expect(d.Release()).To(Succeed())
	})

	It("Should refuse the durable Drop of a clock that BeginDrop found a parted user "+
		"of", func() {
		run(s)
		Expect(s.BeginDrop(5, users)).To(Succeed())
		Expect(s.Durable(10, []log.Record{&log.Drop{Key: 5}})).To(Succeed())
		Expect(s.Snapshot()).Error().To(MatchError(
			ContainSubstring("drop of key 5, the clock of segment 3 of key 6"),
		))
	})
})
