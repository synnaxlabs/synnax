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

// cutOf returns a cut of segment 3 of key 6 in sample() at sample index i, whose
// stamps are 10, 20, and 30 at offsets 0, 4, and 8, with stamp.
func cutOf(i uint32, stamp int64) *log.Cut {
	return &log.Cut{
		Segment:   3,
		Offset:    4 * i,
		Alignment: uint64(telem.NewAlignment(1, i)),
		Stamp:     stamp,
	}
}

var _ = Describe("Tombstone", func() {
	var s *state.State
	BeforeEach(func() { s = MustSucceed(state.Open(config(&appended{}), sample())) })

	// pointersOf runs s and returns the visible pointers of the first segment of key.
	pointersOf := func(key uint32) []state.Pointer {
		GinkgoHelper()
		run(s)
		MustSucceed(s.Snapshot())
		return s.Channel(key).Segments[0].Pointers
	}

	DescribeTable(
		"Should take the samples of the range out of the pointers of the key",
		func(t *log.Tombstone, want []state.Pointer) {
			Expect(s.Replay(10, []log.Record{t})).To(Succeed())
			Expect(pointersOf(6)).To(Equal(want))
			Expect(MustSucceed(s.Snapshot()).Channels[1].Segments[0].Pointers).
				To(Equal(want))
		},
		Entry("the middle of a pointer",
			&log.Tombstone{
				Key: 6, Start: 15, End: 25, Left: cutOf(1, 10),
				Right: cutOf(2, 30),
			},
			[]state.Pointer{
				pointer(1, 0, 10, 10, 0, 4, 1),
				pointer(1, 2, 30, 30, 8, 4, 1),
			}),
		Entry("the head of a pointer",
			&log.Tombstone{Key: 6, Start: 0, End: 25, Right: cutOf(2, 30)},
			[]state.Pointer{pointer(1, 2, 30, 30, 8, 4, 1)}),
		Entry("the tail of a pointer",
			&log.Tombstone{Key: 6, Start: 15, End: 100, Left: cutOf(1, 10)},
			[]state.Pointer{pointer(1, 0, 10, 10, 0, 4, 1)}),
		Entry("a gap between samples, which splits the pointer",
			&log.Tombstone{
				Key: 6, Start: 21, End: 25, Left: cutOf(2, 20),
				Right: cutOf(2, 30),
			},
			[]state.Pointer{
				pointer(1, 0, 10, 20, 0, 8, 2),
				pointer(1, 2, 30, 30, 8, 4, 1),
			}),
		Entry("the whole pointer, which keeps its open segment",
			&log.Tombstone{Key: 6, Start: 0, End: 100}, []state.Pointer{}),
	)

	It("Should remove a sealed segment that no live pointer is left in", func() {
		Expect(s.Replay(10, []log.Record{
			&log.Tombstone{Key: 6, Start: 0, End: 100},
			&log.Tombstone{Key: 5, Start: 0, End: 35},
		})).To(Succeed())
		run(s)
		MustSucceed(s.Snapshot())
		Expect(s.Channel(5).Segments).To(HaveLen(1))
		Expect(s.Channel(5).Segments[0].ID).To(Equal(uint64(4)))
		Expect(s.TakeDead()).To(Equal([]uint64{2}))
		Expect(s.TakeDead()).To(BeEmpty())
	})

	DescribeTable(
		"Should refuse a tombstone that breaks a rule",
		func(t *log.Tombstone, msg string) {
			Expect(s.Replay(10, []log.Record{t})).To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("an empty range", &log.Tombstone{Key: 6, Start: 5, End: 5}, "is empty"),
		Entry("a key with no segment", &log.Tombstone{Key: 9, Start: 0, End: 5},
			"key 9, which holds no segment"),
		Entry("a clock that another key uses",
			&log.Tombstone{Key: 5, Start: 0, End: 35, Left: nil},
			"clock 5 from 0 to 35, which key 6 uses from 10 to 30"),
		Entry("no left cut", &log.Tombstone{Key: 6, Start: 15, End: 100},
			"no left cut, but segment 3 holds that end"),
		Entry("no right cut", &log.Tombstone{Key: 6, Start: 0, End: 25},
			"no right cut, but segment 3 holds that end"),
		Entry("a stale cut", &log.Tombstone{
			Key:   6,
			Start: 15,
			End:   100,
			Left: &log.Cut{
				Segment: 7,
				Offset:  4,
				Alignment: uint64(telem.NewAlignment(1,
					1)),
				Stamp: 10,
			},
		}, "stale left cut in segment 7"),
		Entry("a cut where no pointer holds the end",
			&log.Tombstone{Key: 6, Start: 0, End: 100, Left: cutOf(1, 10)},
			"stale left cut in segment 3"),
		Entry("a cut at the first sample", &log.Tombstone{
			Key: 6, Start: 15, End: 100,
			Left: cutOf(0, 10),
		}, "left cut at offset 0"),
		Entry("a cut stamp outside the pointer", &log.Tombstone{
			Key: 6, Start: 15,
			End: 100, Left: cutOf(1, 5),
		}, "left cut at offset 4"),
		Entry("a right cut before the left one", &log.Tombstone{
			Key: 6, Start: 15,
			End: 25, Left: cutOf(2, 20), Right: cutOf(1, 30),
		}, "cuts its right end"),
	)

	It("Should check a tombstone against the rule and the claims before it is logged",
		func() {
			run(s)
			Expect(s.CheckTombstone(&log.Tombstone{
				Key: 6, Start: 15, End: 25,
				Left: cutOf(1, 10), Right: cutOf(2, 30),
			})).To(Succeed())
			Expect(s.CheckTombstone(&log.Tombstone{Key: 6, Start: 15, End: 100})).
				To(MatchError(state.ErrRefused))
			MustSucceed(s.OpenWriter(state.WriterConfig{
				Keys:  []state.WriterKey{{Key: 6, DataType: 4}},
				Clock: 5,
				Start: 40,
			}))
			Expect(s.CheckTombstone(&log.Tombstone{Key: 6, Start: 35, End: 40})).
				To(Succeed())
			Expect(s.CheckTombstone(&log.Tombstone{Key: 6, Start: 35, End: 45})).
				To(SatisfyAll(
					MatchError(state.ErrConflict),
					MatchError(
						ContainSubstring("reaches the claim of a writer from 40"),
					),
				))
		})

	It("Should apply a durable tombstone to what readers see", func() {
		run(s)
		MustSucceed(s.Snapshot())
		Expect(s.Durable(10, []log.Record{&log.Tombstone{
			Key: 6, Start: 0, End: 25,
			Right: cutOf(2, 30),
		}})).To(Succeed())
		Eventually(func() []state.Pointer { return s.Channel(6).Segments[0].Pointers }).
			Should(Equal([]state.Pointer{pointer(1, 2, 30, 30, 8, 4, 1)}))
		Expect(s.Err()).ToNot(HaveOccurred())
	})
})
