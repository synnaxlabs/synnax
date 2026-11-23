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

var _ = Describe("Check: a Replace that a head could not hold", func() {
	// The sample state holds segment 3 of key 6. The Seal seals it at 12 bytes, the
	// Tombstone takes out its middle sample, and the Create makes segment 9 for the
	// copy of the two samples left.
	records := func(r *log.Replace) []log.Record {
		return []log.Record{
			&log.Seal{Segment: 3, Length: 12, TailCRC: 3},
			&log.Tombstone{
				Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
			},
			&log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 4},
			r,
		}
	}
	// replace returns a Replace of segment 3 by segment 9 of length bytes, with the
	// second sample at offset second.
	replace := func(second, length uint32) *log.Replace {
		return &log.Replace{
			Old: 3, New: 9, Length: length, TailCRC: 7, Pointers: []log.Pointer{
				{
					Length: 4, Samples: 1, Alignment: uint64(telem.NewAlignment(1, 0)),
					First: 10, Last: 10,
				},
				{
					Offset: second, Length: 4, Samples: 1,
					Alignment: uint64(telem.NewAlignment(1, 2)), First: 30, Last: 30,
				},
			},
		}
	}

	It(
		"Should open from the snapshot of a state that applied a packed Replace",
		func() {
			s := MustSucceed(state.Open(config(&appended{}), sample()))
			Expect(s.Replay(10, records(replace(4, 8)))).To(Succeed())
			run(s)
			again := MustSucceed(
				state.Open(config(&appended{}), MustSucceed(s.Snapshot())),
			)
			run(again)
			MustSucceed(again.Snapshot())
			Expect(again.Channel(6).Segments).To(ConsistOf(HaveField("ID", uint64(9))))
		},
	)

	DescribeTable("Should refuse a Replace whose pointers do not fit its new segment",
		func(second, length uint32) {
			s := MustSucceed(state.Open(config(&appended{}), sample()))
			Expect(s.Replay(10, records(replace(second, length)))).To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(ContainSubstring("replace of segment 3")),
			))
		},
		Entry("with a pointer past its length", uint32(6), uint32(8)),
		Entry("with two pointers that overlap", uint32(2), uint32(8)),
		Entry("with a length short of its pointers", uint32(4), uint32(6)),
	)
})
