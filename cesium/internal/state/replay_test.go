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
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// sample is a durable state: clock 5 with a sealed and an open segment, and key 6 on
// clock 5.
func sample() *state.Snapshot {
	return &state.Snapshot{
		Sequence: 9,
		Limits:   state.Limits{Segments: 20, Alignments: 10},
		Channels: []*state.Channel{
			{Key: 5, Segments: []state.Segment{
				{
					ID: 2, Clock: 5, Version: 1, DataType: 7, Sealed: true,
					Length: 24, TailCRC: 1,
					Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 24, 3)},
				},
				{
					ID: 4, Clock: 5, Version: 1, DataType: 7, Length: 16, TailCRC: 2,
					Pointers: []state.Pointer{pointer(2, 0, 40, 50, 0, 16, 2)},
				},
			}},
			{Key: 6, Segments: []state.Segment{{
				ID: 3, Clock: 5, Version: 1, DataType: 4, Length: 12, TailCRC: 3,
				Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 12, 3)},
			}}},
		},
		Intents: []uint32{8},
	}
}

var _ = Describe("Config", func() {
	DescribeTable(
		"Should refuse a config with a field that is not set",
		func(change func(*state.Config), field string) {
			cfg := config(&appended{})
			change(&cfg)
			Expect(cfg.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(field)),
			))
			Expect(
				state.Open(cfg, head(1, 1)),
			).Error().
				To(MatchError(validate.ErrValidation))
		},
		Entry("a version it does not read", func(c *state.Config) { c.Version = 2 },
			"versions"),
		Entry("append", func(c *state.Config) { c.Append = nil }, "append"),
		Entry("executor", func(c *state.Config) { c.Executor = nil }, "executor"),
	)
})

var _ = Describe("Load", func() {
	It("Should open on a head and give back the same durable state", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		run(s)
		Expect(MustSucceed(s.Snapshot())).To(Equal(sample()))
		ch := s.Channel(5)
		Expect(ch.Segments).To(Equal(sample().Channels[0].Segments))
		Expect(s.Channel(7)).To(BeNil())
	})

	DescribeTable(
		"Should refuse a head whose values break a rule",
		func(change func(*state.Snapshot), wraps error, msg string) {
			h := sample()
			change(h)
			Expect(state.Open(config(&appended{}), h)).Error().To(SatisfyAll(
				MatchError(wraps),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("an alignment number limit above stored data", func(h *state.Snapshot) {
			h.Limits.Alignments = 1<<31 + 1
		}, state.ErrRefused, "alignment number limit 2147483649"),
		Entry("segment 0", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].ID = 0
		}, state.ErrRefused, "head names segment 0"),
		Entry("a segment at the limit", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].ID = 20
		}, state.ErrRefused, "segment 20, not from 1 to below the limit 20"),
		Entry("a segment in two keys", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].ID = 2
		}, state.ErrRefused, "head names segment 2 twice"),
		Entry("a version this binary does not read", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].Version = 2
		}, log.ErrUnknown, "segment 3 has format version 2"),
		Entry("a pointer of no samples", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].Pointers[0].Samples = 0
		}, state.ErrRefused, "a pointer of 0 samples"),
		Entry("fewer bytes than samples", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].Pointers[0].Length = 2
		}, state.ErrRefused, "a pointer of 3 samples in 2 bytes"),
		Entry("a first stamp after the last", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].Pointers[0].First = 31
		}, state.ErrRefused, "from 31 to 30"),
		Entry("one sample with two stamps", func(h *state.Snapshot) {
			p := &h.Channels[1].Segments[0].Pointers[0]
			p.Samples, p.Length = 1, 4
		}, state.ErrRefused, "a pointer of 1 samples in 4 bytes from 10 to 30"),
		Entry("pointers that share stream bytes", func(h *state.Snapshot) {
			seg := &h.Channels[0].Segments[1]
			seg.Pointers = append(seg.Pointers, pointer(3, 0, 60, 70, 8, 8, 2))
		}, state.ErrRefused, "a pointer at 8, before the end 16"),
		Entry("alignment number 0", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].Pointers[0] = pointer(0, 0, 10, 30, 0, 12, 3)
		}, state.ErrRefused, "with alignment 0 of 3 samples"),
		Entry("an alignment number at the limit", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].Pointers[0] = pointer(10, 0, 10, 30, 0, 12, 3)
		}, state.ErrRefused, "with alignment 42949672960"),
		Entry("a sample index past its stretch", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].Pointers[0] = pointer(
				1,
				1<<32-2,
				10,
				30,
				0,
				12,
				3,
			)
		}, state.ErrRefused, "of 3 samples"),
		Entry("a pointer past the length", func(h *state.Snapshot) {
			h.Channels[1].Segments[0].Length = 11
		}, state.ErrRefused, "segment 3 has a pointer past its length 11"),
		Entry("pointers of one key that overlap", func(h *state.Snapshot) {
			h.Channels[0].Segments[1].Pointers[0] = pointer(2, 0, 30, 50, 0, 16, 2)
		}, state.ErrRefused, "key 5 has pointers that overlap at 30"),
	)
})

var _ = Describe("Replay", func() {
	var s *state.State
	BeforeEach(func() { s = MustSucceed(state.Open(config(&appended{}), sample())) })

	replay := func(records ...log.Record) error { return s.Replay(10, records) }

	It(
		"Should replay creates, commits, seals, drops, and intents into the state",
		func() {
			Expect(s.Replay(10, []log.Record{
				create(7, 9, 5),
				commit(2, 3, 0, 60, 70, slot(4, 24), slot(7, 8)),
				commit(1, 3, 2, 80, 80, slot(4, 28), slot(7, 12)),
				commit(1, 5, 0, 90, 90, slot(4, 32), slot(7, 16)),
				&log.Seal{Segment: 4, Length: 32, TailCRC: 96},
				&log.Intent{Key: 6},
				&log.Intent{Key: 8, Cleared: true},
			})).To(Succeed())
			Expect(s.Replay(11, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
			run(s)
			snap := MustSucceed(s.Snapshot())
			Expect(snap.Sequence).To(Equal(uint64(11)))
			Expect(snap.Intents).To(BeNil())
			Expect(snap.Channels).To(HaveLen(2))
			Expect(snap.Channels[0].Segments[1]).To(Equal(state.Segment{
				ID: 4, Clock: 5, Version: 1, DataType: 7, Sealed: true, Length: 32,
				TailCRC: 96,
				Pointers: []state.Pointer{
					pointer(2, 0, 40, 50, 0, 16, 2),
					pointer(3, 0, 60, 80, 16, 12, 3),
					pointer(5, 0, 90, 90, 28, 4, 1),
				},
			}))
			Expect(
				snap.Channels[1],
			).To(Equal(&state.Channel{Key: 9, Segments: []state.Segment{{
				ID: 7, Clock: 5, Version: 1, DataType: 4, Length: 16, TailCRC: 48,
				Pointers: []state.Pointer{
					pointer(3, 0, 60, 80, 0, 12, 3),
					pointer(5, 0, 90, 90, 12, 4, 1),
				},
			}}}))
			Expect(s.Channel(9).Segments).To(Equal(snap.Channels[1].Segments))
			Expect(s.Channel(6)).To(BeNil())
		},
	)

	It("Should seal every open segment at a restart", func() {
		Expect(replay(&log.Restart{})).To(Succeed())
		run(s)
		snap := MustSucceed(s.Snapshot())
		for _, ch := range snap.Channels {
			for _, seg := range ch.Segments {
				Expect(seg.Sealed).To(BeTrue())
			}
		}
	})

	It("Should raise the limits with a reserve", func() {
		Expect(replay(&log.Reserve{Segments: 40, Alignments: 100})).To(Succeed())
		run(s)
		Expect(MustSucceed(s.Snapshot()).Limits).
			To(Equal(state.Limits{Segments: 40, Alignments: 100}))
	})

	It("Should refuse an entry that does not follow the last one", func() {
		Expect(s.Replay(11, []log.Record{&log.Restart{}})).
			To(MatchError(ContainSubstring("entry 11 follows entry 9")))
	})

	DescribeTable(
		"Should refuse a record that breaks a rule",
		func(r log.Record, wraps error, msg string) {
			Expect(replay(r)).To(SatisfyAll(
				MatchError(wraps),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("a reserve above stored data", &log.Reserve{
			Segments:   20,
			Alignments: 1<<31 + 1,
		}, state.ErrRefused, "alignment number limit 2147483649"),
		Entry("a reserve below the limits", &log.Reserve{Segments: 19, Alignments: 10},
			state.ErrRefused, "reserve of limits 19 and 10, below 20 and 10"),
		Entry("a create of segment 0", create(0, 9, 5), state.ErrRefused,
			"create of segment 0"),
		Entry("a create at the limit", create(20, 9, 5), state.ErrRefused,
			"create of segment 20, not from 1 to below the limit 20"),
		Entry("a create of a segment that exists", create(3, 9, 5), state.ErrRefused,
			"create of segment 3, which exists"),
		Entry("a create of a version this binary does not read",
			&log.Create{Segment: 7, Key: 9, Clock: 5, Version: 2}, log.ErrUnknown,
			"segment 7 has format version 2"),
		Entry("a commit with no slot", commit(1, 3, 0, 60, 60), state.ErrRefused,
			"commit with no slot"),
		Entry("a commit of no samples", commit(0, 3, 0, 60, 60, slot(4, 20)),
			state.ErrRefused, "commit of 0 samples"),
		Entry("a commit that ends before it starts", commit(2, 3, 0, 60, 59,
			slot(4, 20)), state.ErrRefused, "commit of 2 samples from 60 to 59"),
		Entry("one sample with two stamps", commit(1, 3, 0, 60, 61, slot(4, 20)),
			state.ErrRefused, "commit of 1 samples from 60 to 61"),
		Entry("alignment number 0", commit(1, 0, 0, 60, 60, slot(4, 20)),
			state.ErrRefused, "alignment number 0"),
		Entry("an alignment number at the limit", commit(1, 10, 0, 60, 60, slot(4, 20)),
			state.ErrRefused, "alignment number 10, not from 1 to below the limit 10"),
		Entry("a sample index past its stretch", commit(2, 3, 1<<32-1, 60, 61,
			slot(4, 20)), state.ErrRefused, "commit of 2 samples from sample index"),
		Entry("slots out of order", commit(1, 3, 0, 60, 60, slot(4, 20), slot(3, 16)),
			state.ErrRefused, "slot 1 names segment 3, not above segment 4"),
		Entry("a segment no create made", commit(1, 3, 0, 60, 60, slot(11, 4)),
			state.ErrRefused, "names segment 11, which no create made"),
		Entry("a sealed segment", commit(1, 3, 0, 60, 60, slot(2, 28)),
			state.ErrRefused, "names segment 2, which is sealed"),
		Entry("a length that does not grow", commit(1, 3, 0, 60, 60, slot(4, 16)),
			state.ErrRefused, "gives segment 4 length 16, not above 16"),
		Entry("a commit over stored data", commit(1, 3, 0, 45, 45, slot(4, 20)),
			state.ErrRefused, "commit of key 5 from 45 to 45 overlaps its pointer"),
		Entry("a seal of a segment no create made", &log.Seal{Segment: 11},
			state.ErrRefused, "seal of segment 11, which no create made"),
		Entry("a seal of a sealed segment", &log.Seal{
			Segment: 2, Length: 24,
			TailCRC: 1,
		}, state.ErrRefused, "seal of segment 2, which is sealed"),
		Entry("a seal at another length", &log.Seal{Segment: 4, Length: 12, TailCRC: 2},
			state.ErrRefused, "seal of segment 4 at length 12"),
		Entry("a seal with another tail", &log.Seal{Segment: 4, Length: 16, TailCRC: 9},
			state.ErrRefused, "seal of segment 4 at length 16 and tail 0x9"),
		Entry("a drop of a clock that a kept segment names", &log.Drop{Key: 5},
			state.ErrRefused, "drop of key 5, the clock of segment 3 of key 6"),
		Entry("a clear of a key with no intent", &log.Intent{Key: 9, Cleared: true},
			state.ErrRefused, "intent of key 9 cleared, but it has none"),
		Entry("a replace", &log.Replace{Old: 2, New: 5}, log.ErrUnknown, "Replace"),
		Entry("a check", &log.Check{Segment: 2}, log.ErrUnknown, "Check"),
	)

	It("Should refuse a commit that names segments of two clocks", func() {
		Expect(replay(create(7, 9, 6))).To(Succeed())
		Expect(s.Replay(11, []log.Record{
			commit(1, 3, 0, 60, 60, slot(4, 20), slot(7, 4)),
		})).To(MatchError(ContainSubstring("segments of clocks 5 and 6")))
	})

	It("Should refuse a commit that names two segments of one key", func() {
		Expect(replay(create(7, 6, 5))).To(Succeed())
		Expect(s.Replay(11, []log.Record{
			commit(1, 3, 0, 60, 60, slot(3, 16), slot(7, 4)),
		})).To(MatchError(ContainSubstring("names two segments of key 6")))
	})

	It("Should refuse a segment version that the config does not read", func() {
		cfg := config(&appended{})
		cfg.Versions = set.New[uint16](2)
		cfg.Version = 2
		Expect(state.Open(cfg, sample())).Error().To(MatchError(log.ErrUnknown))
	})
})
