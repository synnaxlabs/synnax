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

var _ = Describe("Tombstone, broken", func() {
	It("Should remove a segment that a replayed Seal sealed and a replayed Tombstone "+
		"left with no live pointer", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		Expect(s.Replay(10, []log.Record{
			&log.Seal{Segment: 4, Length: 16, TailCRC: 2},
			&log.Tombstone{Key: 5, Start: 35, End: 60},
		})).To(Succeed())
		run(s)
		MustSucceed(s.Snapshot())
		Expect(s.TakeDead()).To(Equal([]state.Dead{{Segment: 4, Key: 5, Sequence: 10}}))
		Expect(s.Channel(5).Segments).To(HaveLen(1))
	})

	It("Should remove a segment that a Tombstone left with no live pointer once a "+
		"later Seal seals it", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		Expect(s.Replay(10, []log.Record{
			&log.Tombstone{Key: 5, Start: 35, End: 60},
			&log.Seal{Segment: 4, Length: 16, TailCRC: 2},
		})).To(Succeed())
		run(s)
		MustSucceed(s.Snapshot())
		Expect(s.TakeDead()).To(Equal([]state.Dead{{Segment: 4, Key: 5, Sequence: 10}}))
		Expect(s.Channel(5).Segments).To(HaveLen(1))
	})
})
