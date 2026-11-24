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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Encoding", func() {
	var (
		// seal4 seals clock segment 4 of the sample at its 16 bytes: 4 blocks of 4.
		seal4 = &log.Seal{Segment: 4, Length: 16, TailCRC: 2}
		// encoded replaces segment 4 by segment 9, which holds its stamps in one block.
		encoded = &log.Replace{
			Old:     4,
			New:     9,
			Length:  4,
			TailCRC: 5,
			Pointers: []log.Pointer{{
				Length: 16, Samples: 2, Alignment: uint64(telem.NewAlignment(2, 0)),
				First: 40, Last: 50,
			}},
		}
		a   *appended
		cfg state.Config
	)
	create := func(version uint16) *log.Create {
		return &log.Create{Segment: 9, Key: 5, Clock: 5, Version: version, DataType: 7}
	}
	BeforeEach(func() {
		a = &appended{}
		cfg = config(a)
		cfg.Versions = set.New[uint16](1, 2)
	})
	// open opens the sample state, replays each of entries from sequence 10, and runs
	// it.
	open := func(entries ...[]log.Record) *state.State {
		GinkgoHelper()
		s := MustSucceed(state.Open(cfg, sample()))
		for i, e := range entries {
			Expect(s.Replay(uint64(10+i), e)).To(Succeed())
		}
		a.s, a.seq = s, uint64(9+len(entries))
		run(s)
		return s
	}
	// find returns the record of segment id in a snapshot of s.
	find := func(s *state.State, id uint64) state.Segment {
		GinkgoHelper()
		for _, ch := range MustSucceed(s.Snapshot()).Channels {
			for _, seg := range ch.Segments {
				if seg.ID == id {
					return seg
				}
			}
		}
		Fail("no segment")
		return state.Segment{}
	}
	// appendedAfter waits until s appended entry n, and returns its records.
	appendedAfter := func(n int) []log.Record {
		GinkgoHelper()
		count := func() int {
			a.mu.Lock()
			defer a.mu.Unlock()
			return len(a.entries)
		}
		Eventually(count).Should(BeNumerically(">", n))
		a.mu.Lock()
		defer a.mu.Unlock()
		return decode(a.entries[n])
	}

	It("Should name a sealed clock segment with 2 blocks or more live", func() {
		s := open([]log.Record{seal4})
		Expect(MustSucceed(s.Snapshot()).Candidates).To(Equal([]uint64{4}))
	})

	It("Should not name a clock segment with less than 2 blocks live", func() {
		cfg.Block = 16
		s := open([]log.Record{seal4})
		Expect(MustSucceed(s.Snapshot()).Candidates).To(BeEmpty())
	})

	DescribeTable(
		"Should name a segment only when its rewrite, padded to whole blocks, frees more "+
			"than the threshold",
		func(block uint32, want []uint64) {
			cfg.Block, cfg.Threshold = block, 0.5
			head := sample()
			head.Channels[0].Segments[0].Pointers = []state.Pointer{
				pointer(1, 0, 10, 10, 0, 8, 1),
			}
			s := MustSucceed(state.Open(cfg, head))
			run(s)
			Expect(MustSucceed(s.Snapshot()).Candidates).To(Equal(want))
		},
		Entry("8 of 24 bytes live in blocks of 4", uint32(4), []uint64{2}),
		Entry("8 of 24 bytes live in blocks of 16", uint32(16), []uint64(nil)),
	)

	It("Should take a Kept segment out of the candidates, live and in replay", func() {
		s := open([]log.Record{seal4})
		n := len(a.entries)
		kept := make(chan error, 1)
		go func() { kept <- s.Keep(4) }()
		Expect(appendedAfter(n)).To(Equal([]log.Record{&log.Kept{Segment: 4}}))
		Consistently(kept).ShouldNot(Receive())
		Expect(a.durable([]log.Record{&log.Kept{Segment: 4}})).To(Succeed())
		Eventually(kept).Should(Receive(BeNil()))
		Expect(MustSucceed(s.Snapshot()).Candidates).To(BeEmpty())
		Expect(find(s, 4).Kept).To(BeTrue())
		r := open([]log.Record{seal4}, []log.Record{&log.Kept{Segment: 4}})
		Expect(MustSucceed(r.Snapshot())).To(Equal(MustSucceed(s.Snapshot())))
	})

	It("Should return ErrStale from Keep for a segment that is no candidate", func() {
		s := open([]log.Record{seal4})
		Expect(s.Keep(3)).To(MatchError(state.ErrStale))
		Expect(s.Keep(2)).To(MatchError(state.ErrStale))
	})

	DescribeTable(
		"Should return ErrStale from Keep while a delete or a Drop may read the segment",
		func(change func(s *state.State), message string) {
			s := open([]log.Record{seal4})
			change(s)
			Expect(s.Keep(4)).To(SatisfyAll(
				MatchError(state.ErrStale),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry("a delete of its clock", func(s *state.State) {
			MustSucceed(s.BeginDelete(5, 42, 48))
		}, "delete of key 5 from 42 reads segment 4"),
		Entry("a Drop of its clock", func(s *state.State) {
			Expect(s.BeginDrop(5)).To(Succeed())
		}, "drop of key 5 reads segment 4"),
	)

	DescribeTable(
		"Should refuse a Kept of a segment that is no candidate of encoding",
		func(before []log.Record, id uint64) {
			s := MustSucceed(state.Open(cfg, sample()))
			Expect(s.Replay(10, append(before, &log.Kept{Segment: id}))).To(SatisfyAll(
				refused(),
				MatchError(ContainSubstring("which is no candidate of encoding")),
			))
		},
		Entry("an open clock segment", nil, uint64(4)),
		Entry("a data segment", []log.Record{&log.Seal{
			Segment: 3, Length: 12, TailCRC: 3,
		}}, uint64(3)),
		Entry("a Kept segment", nil, uint64(2)),
	)

	It("Should apply a Replace by progressions whose pointers end past its length, "+
		"and open on its snapshot", func() {
		s := open([]log.Record{seal4}, []log.Record{create(2), encoded})
		seg := find(s, 9)
		Expect(seg.Version).To(BeEquivalentTo(2))
		Expect(seg.Sealed).To(BeTrue())
		Expect(seg.Length).To(BeEquivalentTo(4))
		Expect(seg.Pointers[0].Length).To(BeEquivalentTo(16))
		snap := MustSucceed(s.Snapshot())
		Expect(snap.Candidates).To(BeEmpty())
		reopened := MustSucceed(state.Open(cfg, snap))
		run(reopened)
		Expect(MustSucceed(reopened.Snapshot()).Channels).To(Equal(snap.Channels))
	})

	It("Should append the Create of a version 2 segment for a Replace by progressions",
		func() {
			s := open([]log.Record{seal4})
			n := len(a.entries)
			replaced := make(chan error, 1)
			go func() { replaced <- s.Replace(encoded, 2) }()
			records := appendedAfter(n)
			Expect(records).To(HaveLen(2))
			Expect(records[0]).To(Equal(create(2)))
			Expect(a.durable(records)).To(Succeed())
			Eventually(replaced).Should(Receive(BeNil()))
		})

	It("Should refuse a Replace by progressions of a Kept segment", func() {
		s := MustSucceed(state.Open(cfg, sample()))
		Expect(s.Replay(10, []log.Record{seal4, &log.Kept{Segment: 4}})).To(Succeed())
		Expect(s.Replay(11, []log.Record{create(2), encoded})).To(SatisfyAll(
			refused(),
			MatchError(ContainSubstring(
				"replace of segment 4 by the progressions of 9, which is no candidate "+
					"of encoding",
			)),
		))
	})

	It("Should keep the Kept mark in a version 1 rewrite of a Kept segment", func() {
		head := sample()
		head.Channels[0].Segments[0].Pointers = []state.Pointer{
			pointer(1, 0, 10, 10, 0, 8, 1),
		}
		s := MustSucceed(state.Open(cfg, head))
		Expect(s.Replay(10, []log.Record{create(1), &log.Replace{
			Old: 2, New: 9, Length: 8, TailCRC: 5, Pointers: []log.Pointer{{
				Length: 8, Samples: 1, Alignment: uint64(telem.NewAlignment(1, 0)),
				First: 10, Last: 10,
			}},
		}})).To(Succeed())
		run(s)
		Expect(find(s, 9).Kept).To(BeTrue())
	})

	It("Should refuse a head that marks a segment Kept that is no sealed clock "+
		"segment", func() {
		head := sample()
		head.Channels[1].Segments[0].Kept = true
		Expect(state.Open(cfg, head)).Error().To(SatisfyAll(
			refused(),
			MatchError(ContainSubstring("head names segment 3, kept but no sealed "+
				"clock segment of version 1")),
		))
	})
})
