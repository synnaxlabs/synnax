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

// converted is a durable state of old files: clock 5 in old file 1 as segment 2, and
// key
// 6 on clock 5 in old file 1 as segment 3, both sealed version 0 segments.
func converted() *state.Snapshot {
	return &state.Snapshot{
		Sequence: 9,
		Limits:   state.Limits{Segments: 20, Alignments: 10},
		Channels: []*state.Channel{
			{Key: 5, Segments: []state.Segment{{
				ID: 2, Clock: 5, File: 1, DataType: 7, Sealed: true, Length: 24,
				Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 24, 3)},
			}}},
			{Key: 6, Segments: []state.Segment{{
				ID: 3, Clock: 5, File: 1, DataType: 4, Sealed: true, Length: 12,
				Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 12, 3)},
			}}},
		},
	}
}

// published returns the Channel of key once the first step of Run publishes it.
func published(s *state.State, key uint32) *state.Channel {
	GinkgoHelper()
	Eventually(func() *state.Channel { return s.Channel(key) }).ShouldNot(BeNil())
	return s.Channel(key)
}

var _ = Describe("Check", func() {
	It("Should turn on the checks of a sealed version 0 segment, and kill the record "+
		"that a part may hold", func() {
		s := MustSucceed(state.Open(config(&appended{}), converted()))
		Expect(s.Replay(10, []log.Record{&log.Check{Segment: 3}})).To(Succeed())
		run(s)
		Expect(published(s, 6).Segments[0].Checked).To(BeTrue())
		Expect(published(s, 5).Segments[0].Checked).To(BeFalse())
		snap := MustSucceed(s.Snapshot())
		Expect(snap.Channels[1].Segments[0].Checked).To(BeTrue())
		Expect(snap.Kills).To(Equal([]state.Kill{{Key: 6, Segment: 3, Generation: 1}}))
	})

	It("Should open on a head whose version 0 segment has its checks", func() {
		head := converted()
		head.Channels[1].Segments[0].Checked = true
		s := MustSucceed(state.Open(config(&appended{}), head))
		run(s)
		Expect(published(s, 6).Segments[0].Checked).To(BeTrue())
	})

	DescribeTable("Should refuse a Check of a segment that cannot take one", func(
		head func() *state.Snapshot,
		records []log.Record,
		msg string,
	) {
		s := MustSucceed(state.Open(config(&appended{}), head()))
		Expect(s.Replay(10, records)).To(SatisfyAll(
			MatchError(state.ErrRefused),
			MatchError(ContainSubstring(msg)),
		))
	},
		Entry("no create made", converted, []log.Record{&log.Check{Segment: 9}},
			"check of segment 9, which no create made"),
		Entry("a version 1 segment", sample, []log.Record{&log.Check{Segment: 2}},
			"check of segment 2, which is not a sealed version 0 segment"),
		Entry("checks already on", converted, []log.Record{
			&log.Check{Segment: 3},
			&log.Check{Segment: 3},
		}, "check of segment 3, which has its checks"),
	)

	It(
		"Should refuse a head that gives the check flag to a segment that is not version "+
			"0",
		func() {
			head := sample()
			head.Channels[0].Segments[0].Checked = true
			Expect(state.Open(config(&appended{}), head)).Error().To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(ContainSubstring(
					"segment 2 of version 1 has the check flag of an old file",
				)),
			))
		},
	)
})
