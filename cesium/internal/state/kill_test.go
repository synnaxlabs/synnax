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

var _ = Describe("Kills", func() {
	var s *state.State
	BeforeEach(func() {
		cfg := config(&appended{})
		cfg.Generation = 4
		s = MustSucceed(state.Open(cfg, sample()))
		Expect(s.Replay(10, []log.Record{
			&log.Seal{Segment: 3, Length: 12, TailCRC: 3},
		})).To(Succeed())
		run(s)
	})
	// kills returns the kills of the snapshot once the owner ran every queued request.
	kills := func() []state.Kill {
		GinkgoHelper()
		return MustSucceed(s.Snapshot()).Kills
	}
	cut := &log.Tombstone{
		Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
	}

	It("Should kill a sealed segment that a tombstone cuts, with the generation of "+
		"its log", func() {
		Expect(s.Durable(11, []log.Record{cut})).To(Succeed())
		want := []state.Kill{{Key: 6, Segment: 3, Generation: 4}}
		Expect(kills()).To(Equal(want))
		Expect(s.Channel(6).Kills).To(Equal(want))
	})

	It("Should not kill an open segment that a tombstone cuts", func() {
		Expect(s.Durable(11, []log.Record{
			&log.Tombstone{Key: 5, Start: 35, End: 60},
		})).To(Succeed())
		Expect(kills()).To(BeEmpty())
	})

	It("Should kill every segment of a dropped key", func() {
		Expect(s.Durable(11, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
		Expect(kills()).To(Equal([]state.Kill{{Key: 6, All: true, Generation: 4}}))
	})

	It("Should stamp a kill after a switch with the generation of the new log", func() {
		Expect(s.Switch(5)).To(Succeed())
		Expect(s.Durable(11, []log.Record{cut})).To(Succeed())
		Expect(kills()).To(Equal([]state.Kill{{Key: 6, Segment: 3, Generation: 5}}))
	})

	It("Should drop the kills from before the generation of a committed head and keep "+
		"the rest", func() {
		Expect(s.Durable(11, []log.Record{cut})).To(Succeed())
		Expect(s.Switch(5)).To(Succeed())
		Expect(s.Durable(12, []log.Record{
			&log.Tombstone{Key: 6, Start: 30, End: 31},
		})).To(Succeed())
		Expect(s.Parted(5, nil, nil)).To(Succeed())
		want := []state.Kill{{Key: 6, Segment: 3, Generation: 5}}
		Expect(kills()).To(Equal(want))
		Expect(s.Channel(6).Kills).To(Equal(want))
	})

	It("Should give each key its part range, and take the range of a key that a later "+
		"head leaves out", func() {
		Expect(s.Parted(5, nil, []state.Range{{Key: 6, Oldest: 2, Newest: 3}})).
			To(Succeed())
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Oldest).To(Equal(uint64(2)))
		Expect(s.Channel(6).Newest).To(Equal(uint64(3)))
		Expect(s.Parted(6, nil, nil)).To(Succeed())
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Oldest).To(BeZero())
		Expect(s.Channel(6).Newest).To(BeZero())
	})
})
