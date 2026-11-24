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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rebuild", func() {
	var s *state.State
	BeforeEach(func() {
		cfg := config(&appended{})
		cfg.Repairing = true
		s = MustSucceed(state.Open(cfg, sample()))
	})
	limits := state.Limits{Segments: 30, Alignments: 15}
	// commit returns a Commit of samples samples of alignment number from first to
	// last, with one slot of segment at length.
	commit := func(
		number, samples uint32,
		first, last int64,
		segment uint64,
		length uint32,
	) log.Commit {
		return log.Commit{
			Samples:   samples,
			Alignment: uint64(telem.NewAlignment(number, 0)),
			First:     first,
			Last:      last,
			Slots:     []log.Slot{{Segment: segment, Length: length, TailCRC: 9}},
		}
	}

	It("Should raise the limits, create each placed segment, and append each range "+
		"with a fresh alignment by the rules of a Commit", func() {
		errs := MustSucceed(s.Rebuild(state.Rebuild{
			Sequence: 12,
			Limits:   limits,
			Creates: []log.Create{
				{Segment: 25, Key: 7, Clock: 7, Version: 1, DataType: 7},
			},
			Commits: []log.Commit{
				commit(12, 2, 60, 70, 4, 32),
				commit(13, 3, 100, 120, 25, 24),
			},
		}))
		Expect(errs).To(Equal([]error{nil, nil}))
		open := sample().Channels[0].Segments
		open[1].Length, open[1].TailCRC = 32, 9
		open[1].Pointers = append(open[1].Pointers, pointer(15, 0, 60, 70, 16, 16, 2))
		Expect(s.ReplaySegments(5)).To(Equal(open))
		Expect(s.ReplaySegments(7)).To(Equal([]state.Segment{{
			ID:       25,
			Clock:    7,
			Version:  1,
			DataType: 7,
			Length:   24,
			TailCRC:  9,
			Pointers: []state.Pointer{pointer(16, 0, 100, 120, 0, 24, 3)},
		}}))
		Expect(s.Replay(13, []log.Record{&log.Intent{Key: 9}})).To(Succeed())
		run(s)
		Expect(MustSucceed(s.Snapshot()).Limits).
			To(Equal(state.Limits{Segments: 30, Alignments: 17}))
	})

	It("Should return the error of a range that breaks a rule, and apply the rest",
		func() {
			errs := MustSucceed(s.Rebuild(state.Rebuild{
				Sequence: 9,
				Limits:   limits,
				Commits: []log.Commit{
					commit(12, 2, 20, 25, 3, 20),
					commit(13, 2, 60, 70, 4, 32),
				},
			}))
			Expect(errs).To(HaveLen(2))
			Expect(errs[0]).To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(ContainSubstring("overlaps its pointer")),
			))
			Expect(errs[1]).ToNot(HaveOccurred())
			Expect(s.ReplaySegments(6)).To(Equal(sample().Channels[1].Segments))
		})

	It("Should keep the limits of a rebuild above those of a later Reserve", func() {
		MustSucceed(s.Rebuild(state.Rebuild{Sequence: 9, Limits: limits}))
		Expect(s.Replay(10, []log.Record{
			&log.Reserve{Segments: 25, Alignments: 12},
		})).To(Succeed())
		run(s)
		Expect(MustSucceed(s.Snapshot()).Limits).To(Equal(limits))
	})

	It("Should keep limits above those of the rebuild", func() {
		MustSucceed(s.Rebuild(state.Rebuild{
			Sequence: 9,
			Limits:   state.Limits{Segments: 5, Alignments: 3},
		}))
		run(s)
		Expect(MustSucceed(s.Snapshot()).Limits).
			To(Equal(state.Limits{Segments: 20, Alignments: 10}))
	})

	It("Should move each segment to the end of its gap, from which a later Commit "+
		"grows", func() {
		MustSucceed(s.Rebuild(state.Rebuild{
			Sequence: 9,
			Limits:   limits,
			Ends:     []log.Slot{{Segment: 4, Length: 40, TailCRC: 11}},
		}))
		c := commit(12, 2, 60, 70, 4, 56)
		Expect(s.Replay(10, []log.Record{&c})).To(Succeed())
		segs := s.ReplaySegments(5)
		Expect(segs[1].Length).To(Equal(uint32(56)))
		Expect(segs[1].Pointers[1]).To(Equal(pointer(12, 0, 60, 70, 40, 16, 2)))
	})

	DescribeTable("Should refuse a rebuild that breaks a rule",
		func(r state.Rebuild, msg string) {
			Expect(s.Rebuild(r)).Error().To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("an entry before the state's", state.Rebuild{Sequence: 8, Limits: limits},
			"rebuild at entry 8, before entry 9"),
		Entry("an alignment number limit past the domain limit", state.Rebuild{
			Sequence: 9,
			Limits:   state.Limits{Alignments: uint64(telem.LeadingDomain)},
			Commits:  []log.Commit{commit(1, 2, 60, 70, 4, 32)},
		}, "rebuild of alignment number limit 2147483649, above 2147483648"),
		Entry("a create of a segment that exists", state.Rebuild{
			Sequence: 9,
			Limits:   limits,
			Creates: []log.Create{
				{Segment: 4, Key: 5, Clock: 5, Version: 1, DataType: 7},
			},
		}, "create of segment 4, which exists"),
		Entry("an end below the length", state.Rebuild{
			Sequence: 9,
			Limits:   limits,
			Ends:     []log.Slot{{Segment: 4, Length: 8}},
		}, "end of segment 4 at length 8, below 16"),
		Entry("an end of a sealed segment", state.Rebuild{
			Sequence: 9,
			Limits:   limits,
			Ends:     []log.Slot{{Segment: 2, Length: 30}},
		}, "end of segment 2, which is sealed"),
	)
})

var _ = Describe("ReplayEach", func() {
	It("Should apply each record of an entry that breaks no rule, and return each "+
		"that does with its error", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		seal := &log.Seal{Segment: 4, Length: 8}
		c := &log.Commit{
			Samples:   1,
			Alignment: uint64(telem.NewAlignment(3, 0)),
			First:     60,
			Last:      60,
			Slots:     []log.Slot{{Segment: 99, Length: 8}},
		}
		skipped := MustSucceed(s.ReplayEach(10, []log.Record{
			seal,
			&log.Intent{Key: 9},
			c,
		}))
		Expect(skipped).To(HaveLen(2))
		Expect(skipped[0].Record).To(BeIdenticalTo(seal))
		Expect(skipped[0].Err).To(SatisfyAll(
			MatchError(state.ErrRefused),
			MatchError(ContainSubstring("not its committed 16")),
		))
		Expect(skipped[1].Record).To(BeIdenticalTo(c))
		Expect(skipped[1].Err).To(MatchError(ContainSubstring("which no create made")))
		Expect(s.ReplaySegments(5)).To(Equal(sample().Channels[0].Segments))
		Expect(s.Replay(11, nil)).To(Succeed())
		run(s)
		Expect(MustSucceed(s.Snapshot()).Intents).To(ConsistOf(uint32(8), uint32(9)))
	})

	It("Should refuse an entry that does not follow the entry of the state", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		Expect(s.ReplayEach(11, nil)).Error().To(SatisfyAll(
			MatchError(state.ErrRefused),
			MatchError(ContainSubstring("entry 11 follows entry 9")),
		))
	})
})

var _ = Describe("Remove", func() {
	var s *state.State
	open := func(h *state.Snapshot) {
		cfg := config(&appended{})
		cfg.Repairing = true
		s = MustSucceed(state.Open(cfg, h))
	}

	It("Should give the keys of the state", func() {
		open(sample())
		Expect(s.ReplayKeys()).To(Equal([]uint32{5, 6}))
	})

	It("Should take a segment of the head out of the state", func() {
		open(sample())
		Expect(s.Remove(5, 4, false)).To(Succeed())
		Expect(s.ReplaySegments(5)).To(Equal(sample().Channels[0].Segments[:1]))
		run(s)
		want := sample()
		want.Channels[0].Segments = want.Channels[0].Segments[:1]
		Expect(MustSucceed(s.Snapshot())).To(Equal(want))
	})

	It("Should kill the record of a segment that a part holds", func() {
		h := sample()
		h.Channels[0].Parts = &state.Parts{Oldest: 1, Newest: 1}
		open(h)
		Expect(s.Remove(5, 30, true)).To(Succeed())
		parts, ok := s.ReplayParts(5)
		Expect(ok).To(BeTrue())
		Expect(
			parts.Kills,
		).To(Equal([]state.Kill{{Key: 5, Segment: 30, Generation: 1}}))
	})

	DescribeTable("Should refuse a removal of a segment that the state does not hold",
		func(key uint32, segment uint64) {
			open(sample())
			Expect(s.Remove(key, segment, false)).To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(ContainSubstring("which the state does not hold")),
			))
		},
		Entry("of a key that it lacks", uint32(9), uint64(4)),
		Entry("of another key", uint32(6), uint64(4)),
		Entry("of no segment", uint32(5), uint64(30)),
	)
})

var _ = Describe("Gone", func() {
	var s *state.State
	BeforeEach(func() {
		cfg := config(&appended{})
		cfg.Repairing = true
		s = MustSucceed(state.Open(cfg, sample()))
	})
	create := &log.Create{Segment: 19, Key: 5, Clock: 5, Version: 1, DataType: 7}
	// replace returns a Replace of segment 4 by 19 with one pointer from first to last.
	replace := func(first, last int64) *log.Replace {
		return &log.Replace{
			Old:     4,
			New:     19,
			Length:  16,
			TailCRC: 7,
			Pointers: []log.Pointer{{
				Offset:    0,
				Length:    16,
				Samples:   2,
				Alignment: uint64(telem.NewAlignment(2, 0)),
				First:     first,
				Last:      last,
			}},
		}
	}

	It("Should take an open old segment whose file is gone out of the state, and give "+
		"the copy the pointers of the Replace", func() {
		s.Gone(4)
		Expect(s.Replay(10, []log.Record{create, replace(40, 50)})).To(Succeed())
		segs := s.ReplaySegments(5)
		Expect(segs).To(HaveLen(2))
		Expect(segs[0]).To(Equal(sample().Channels[0].Segments[0]))
		Expect(segs[1]).To(Equal(state.Segment{
			ID:       19,
			Clock:    5,
			Version:  1,
			DataType: 7,
			Sealed:   true,
			Length:   16,
			TailCRC:  7,
			Pointers: []state.Pointer{pointer(2, 0, 40, 50, 0, 16, 2)},
		}))
	})

	It("Should refuse the Replace of an open segment whose file is not gone", func() {
		Expect(s.Replay(10, []log.Record{create, replace(40, 50)})).
			To(MatchError(ContainSubstring("replace of segment 4, which is not sealed")))
	})

	It("Should skip a Replace whose pointers overlap another pointer of the key, and "+
		"change nothing", func() {
		s.Gone(4)
		skipped := MustSucceed(s.ReplayEach(10, []log.Record{create, replace(20, 25)}))
		Expect(skipped).To(HaveLen(1))
		Expect(skipped[0].Err).To(MatchError(ContainSubstring(
			"overlaps a pointer of key 5 from 10 to 30",
		)))
		segs := s.ReplaySegments(5)
		Expect(segs[:2]).To(Equal(sample().Channels[0].Segments))
		Expect(segs[2].Sealed).To(BeFalse())
	})
})
