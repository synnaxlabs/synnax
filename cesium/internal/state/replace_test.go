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

var _ = Describe("Rewrites", func() {
	var (
		// seal seals segment 3 of key 6 at its 12 bytes.
		seal = &log.Seal{Segment: 3, Length: 12, TailCRC: 3}
		// cut takes the middle sample out of segment 3, which leaves 8 of its 12 bytes.
		cut = &log.Tombstone{
			Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
		}
	)
	// candidates returns the candidates of a rewrite in a snapshot of s.
	candidates := func(s *state.State) []uint64 {
		GinkgoHelper()
		return MustSucceed(s.Snapshot()).Candidates
	}

	Describe("Candidates", func() {
		// open opens the sample state with cfg, replays records, and runs it.
		open := func(cfg state.Config, records ...log.Record) *state.State {
			GinkgoHelper()
			s := MustSucceed(state.Open(cfg, sample()))
			Expect(s.Replay(10, records)).To(Succeed())
			run(s)
			return s
		}

		It("Should name a sealed segment that a Tombstone leaves above the dead share",
			func() {
				Expect(candidates(open(config(&appended{}), seal, cut))).
					To(Equal([]uint64{3}))
			})

		It("Should not name a segment whose dead share is at most the threshold",
			func() {
				cfg := config(&appended{})
				cfg.Threshold = 0.5
				Expect(candidates(open(cfg, seal, cut))).To(BeEmpty())
			})

		DescribeTable(
			"Should name a segment that a Tombstone cut while open once it seals",
			func(sealing log.Record) {
				s := open(config(&appended{}), cut)
				Expect(candidates(s)).To(BeEmpty())
				Expect(s.Durable(11, []log.Record{sealing})).To(Succeed())
				Expect(candidates(s)).To(Equal([]uint64{3}))
			},
			Entry("by a Seal", seal),
			Entry("by a Restart", &log.Restart{}),
		)

		It("Should name a segment of the head above the dead share at open", func() {
			head := sample()
			head.Channels[0].Segments[0].Pointers = []state.Pointer{
				pointer(1, 0, 10, 10, 0, 8, 1),
			}
			s := MustSucceed(state.Open(config(&appended{}), head))
			run(s)
			Expect(candidates(s)).To(Equal([]uint64{2}))
		})

		It("Should not name a segment of another format version", func() {
			head := sample()
			seg := &head.Channels[0].Segments[0]
			seg.Version = 2
			seg.Pointers = []state.Pointer{pointer(1, 0, 10, 10, 0, 8, 1)}
			cfg := config(&appended{})
			cfg.Versions = set.New[uint16](1, 2)
			s := MustSucceed(state.Open(cfg, head))
			run(s)
			Expect(candidates(s)).To(BeEmpty())
		})

		DescribeTable("Should take out a candidate that leaves the state",
			func(leave log.Record) {
				s := open(config(&appended{}), seal, cut)
				Expect(s.Durable(11, []log.Record{leave})).To(Succeed())
				Expect(candidates(s)).To(BeEmpty())
			},
			Entry("by a Tombstone that empties it",
				&log.Tombstone{Key: 6, Start: 0, End: 100}),
			Entry("by a Drop of its key", &log.Drop{Key: 6}),
		)
	})

	Describe("Replace", func() {
		var s *state.State
		BeforeEach(func() {
			s = MustSucceed(state.Open(config(&appended{}), sample()))
			Expect(s.Replay(10, []log.Record{seal, cut})).To(Succeed())
		})
		created := &log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 4}
		// replace returns a Replace of segment 3 by segment 9 that packs its two live
		// samples, changed by edit.
		replace := func(edit func(r *log.Replace)) *log.Replace {
			r := &log.Replace{Old: 3, New: 9, Length: 8, TailCRC: 7}
			r.Pointers = []log.Pointer{
				{
					Length: 4, Samples: 1, Alignment: uint64(telem.NewAlignment(1, 0)),
					First: 10, Last: 10,
				},
				{
					Offset: 4, Length: 4, Samples: 1,
					Alignment: uint64(telem.NewAlignment(1, 2)), First: 30, Last: 30,
				},
			}
			if edit != nil {
				edit(r)
			}
			return r
		}
		// expectReplaced expects segment 9 in place of segment 3.
		expectReplaced := func() {
			GinkgoHelper()
			MustSucceed(s.Snapshot())
			Expect(s.Channel(6).Segments).To(Equal([]state.Segment{{
				ID: 9, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 8,
				TailCRC: 7, Pointers: []state.Pointer{
					pointer(1, 0, 10, 10, 0, 4, 1),
					pointer(1, 2, 30, 30, 4, 4, 1),
				},
			}}))
			Expect(s.TakeDead()).To(Equal([]state.Dead{{Segment: 3, Key: 6}}))
			Expect(candidates(s)).To(BeEmpty())
		}

		It("Should seal the new segment with the copied pointers in place of the old "+
			"one", func() {
			run(s)
			Expect(candidates(s)).To(Equal([]uint64{3}))
			Expect(s.Durable(11, []log.Record{created, replace(nil)})).To(Succeed())
			expectReplaced()
		})

		It("Should apply a Replace in replay as at runtime", func() {
			Expect(s.Replay(11, []log.Record{created, replace(nil)})).To(Succeed())
			run(s)
			expectReplaced()
		})

		DescribeTable(
			"Should refuse a Replace whose new segment differs from the old one",
			func(create *log.Create, edit func(r *log.Replace), message string) {
				Expect(s.Replay(11, []log.Record{create, replace(edit)})).To(SatisfyAll(
					MatchError(state.ErrRefused),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("in key",
				&log.Create{Segment: 9, Key: 5, Clock: 5, Version: 1, DataType: 4}, nil,
				"replace of segment 3 by 9, which differs in key, clock, or type"),
			Entry("in data type",
				&log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 7}, nil,
				"replace of segment 3 by 9, which differs in key, clock, or type"),
			Entry("in the samples of a pointer", created,
				func(r *log.Replace) { r.Pointers[1].Samples = 2 },
				"replace of segment 3, whose pointers differ from the copy"),
			Entry("in the stamps of a pointer", created,
				func(r *log.Replace) { r.Pointers[0].Last = 11 },
				"replace of segment 3, whose pointers differ from the copy"),
			Entry("in its count of pointers", created,
				func(r *log.Replace) { r.Pointers = r.Pointers[:1] },
				"replace of segment 3, whose pointers differ from the copy"),
		)
	})
})

var _ = Describe("Replace of a put-back record", func() {
	It("Should refuse the old segment while a part holds its record", func() {
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
		MustSucceed(s.BeginDelete(6, 15, 25, state.Preload{
			Key: 6, Generation: 5, Segments: []state.Segment{record},
		}))
		Expect(s.Durable(10, []log.Record{
			&log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 4},
			&log.Replace{
				Old: 3, New: 9, Length: 12, TailCRC: 3,
				Pointers: []log.Pointer{{
					Length: 12, Samples: 3, Alignment: uint64(telem.NewAlignment(1, 0)),
					First: 10, Last: 30,
				}},
			},
		})).To(Succeed())
		Eventually(s.Err).Should(SatisfyAll(
			MatchError(state.ErrRefused),
			MatchError(ContainSubstring("replace of segment 3, whose record part 7 "+
				"holds")),
		))
	})
})
