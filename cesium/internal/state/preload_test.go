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
	"context"

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
		Expect(
			s.OpenWriter(context.Background(), writer, preload(5, record)),
		).Error().
			To(SatisfyAll(
				conflict(),
				MatchError(ContainSubstring("inside its stored data from 10 to 30")),
			))
	})

	It("Should take the records of a parted segment from the preload alone", func() {
		w := MustSucceed(s.OpenWriter(context.Background(), writer, preload(5)))
		Expect(w.Close()).To(Succeed())
	})

	It("Should refuse a preload from the parts of an older head as stale", func() {
		Expect(
			s.OpenWriter(context.Background(), writer, preload(4, record)),
		).Error().
			To(SatisfyAll(
				MatchError(state.ErrStale),
				MatchError(ContainSubstring("key 6 reads the parts of head 5, not 4")),
			))
		Expect(s.BeginDelete(6, 15, 25, preload(4, record))).Error().
			To(MatchError(state.ErrStale))
	})

	It("Should refuse a step with no preload of a key that parts hold", func() {
		Expect(s.OpenWriter(context.Background(), writer)).Error().To(SatisfyAll(
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
