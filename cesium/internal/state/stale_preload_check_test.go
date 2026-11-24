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

// The check pass loads a preload with no root lock, so a delete can bury a part record
// of the preload before the owner runs the step that takes it, in the same head.
var _ = Describe("Check: a preload that a delete makes stale in its head", func() {
	var (
		a   *appended
		s   *state.State
		pre state.Preload
	)
	BeforeEach(func() {
		a = &appended{}
		s = MustSucceed(state.Open(config(a), converted()))
		a.s, a.seq = s, 9
		run(s)
		MustSucceed(s.Snapshot())
		record := s.Channel(6).Segments[0]
		record.Part = 7
		Expect(s.Parted(
			5,
			[]state.Tag{{Part: 7, Segment: record}},
			[]state.Range{{Key: 6, Oldest: 7, Newest: 7}},
			func() {},
		)).To(Succeed())
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Segments).To(BeEmpty())
		pre = state.Preload{Key: 6, Generation: 5, Segments: []state.Segment{record}}
		d := MustSucceed(s.BeginDelete(6, 0, 100, pre))
		t := &log.Tombstone{Key: 6, End: 100}
		Expect(d.Check(t)).To(Succeed())
		Expect(a.durable([]log.Record{t})).To(Succeed())
		Expect(d.Release()).To(Succeed())
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Segments).To(BeEmpty())
	})

	It("Should refuse a Check of the buried segment as stale, and keep it out of the "+
		"state", func() {
		Expect(s.Check(&log.Check{Segment: 3, Key: 6}, pre)).To(SatisfyAll(
			MatchError(state.ErrStale),
			MatchError(ContainSubstring("segment 3 of key 6 left the state")),
		))
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Segments).To(BeEmpty())
	})

	It("Should refuse a delete that takes the buried record from the preload as stale",
		func() {
			Expect(s.BeginDelete(6, 20, 25, pre)).Error().To(MatchError(state.ErrStale))
			MustSucceed(s.Snapshot())
			Expect(s.Channel(6).Segments).To(BeEmpty())
		})
})
