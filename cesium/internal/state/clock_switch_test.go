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

var _ = Describe("Clock users across a switch", func() {
	// tombstone deletes stamp 20 of segment 2 of clock 5, which segment 3 of key 6
	// uses from 10 to 30.
	tombstone := &log.Tombstone{
		Key:   5,
		Start: 15,
		End:   25,
		Left: &log.Cut{
			Segment: 2, Offset: 8, Alignment: uint64(telem.NewAlignment(1, 1)),
			Stamp: 10,
		},
		Right: &log.Cut{
			Segment: 2, Offset: 16, Alignment: uint64(telem.NewAlignment(1, 2)),
			Stamp: 30,
		},
	}
	DescribeTable("Should refuse the Tombstone of a clock that a sealed user holds "+
		"data in", func(switched bool) {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		Expect(s.Replay(10, []log.Record{
			&log.Seal{Segment: 3, Length: 12, TailCRC: 3},
		})).To(Succeed())
		run(s)
		d := MustSucceed(s.BeginDelete(5, 15, 25, state.Preload{Key: 5, Generation: 1}))
		if switched {
			record := s.Channel(6).Segments[0]
			record.Part = 7
			Expect(s.Parted(
				2,
				[]state.Tag{{Part: 7, Segment: record}},
				[]state.Range{{Key: 6, Oldest: 7, Newest: 7}},
				func() {},
			)).To(Succeed())
		}
		Expect(d.Check(tombstone)).To(SatisfyAll(
			refused(),
			MatchError(ContainSubstring("which key 6 uses from 10 to 30")),
		))
		Expect(d.Release()).To(Succeed())
	},
		Entry("with no switch", false),
		Entry("when a switch parts the user after the claim", true),
	)
})
