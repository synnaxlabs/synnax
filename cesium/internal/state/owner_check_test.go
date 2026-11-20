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

var _ = Describe("State owner check", func() {
	keys := []state.WriterKey{{Key: 5, DataType: 4}}
	create := func(id uint64) *log.Create {
		return &log.Create{Segment: id, Key: 5, Clock: 9, Version: 1, DataType: 4}
	}
	// start opens and runs a state on h whose Reserves become durable as it appends
	// them.
	start := func(h *state.Snapshot) (*state.State, *appended) {
		GinkgoHelper()
		a := &appended{}
		s := MustSucceed(state.Open(config(a), h))
		a.s, a.seq = s, h.Sequence
		run(s)
		return s, a
	}
	// switchTo switches s to generation and returns the Seals that it appended.
	switchTo := func(s *state.State, a *appended, generation uint64) []log.Seal {
		GinkgoHelper()
		heads := len(a.switched())
		Expect(s.Switch(generation)).To(Succeed())
		Eventually(a.switched).Should(HaveLen(heads + 1))
		return a.seals()
	}

	It("Should not seal at a switch a segment that no durable Create made", func() {
		s, a := start(head(1, 1))
		w := MustSucceed(
			s.OpenWriter(state.WriterConfig{Keys: keys, Clock: 5, Start: 10}),
		)
		id := MustSucceed(w.NewSegment(5))
		Expect(w.Close()).To(Succeed())
		seals := switchTo(s, a, 2)
		records := make([]log.Record, len(seals))
		for i := range seals {
			Expect(seals[i].Segment).ToNot(Equal(id))
			records[i] = &seals[i]
		}
		if len(records) > 0 {
			Expect(a.durable(records)).To(Succeed())
		}
		MustSucceed(s.Snapshot())
		Expect(s.Err()).ToNot(HaveOccurred())
		Expect(MustSucceed(s.TakeDead())).To(ContainElement(id))
	})

	It("Should seal every open segment of a key after a takeover and a close", func() {
		s, a := start(head(1, 1))
		first := MustSucceed(s.OpenWriter(state.WriterConfig{
			Keys: keys, Clock: 9, Start: 100,
		}))
		second := MustSucceed(s.OpenWriter(state.WriterConfig{
			Keys: keys, Clock: 9, Start: 10, Bounded: true,
		}))
		ia, ib := MustSucceed(first.NewSegment(5)), MustSucceed(second.NewSegment(5))
		Expect(a.durable([]log.Record{create(ia), create(ib)})).To(Succeed())
		Expect(first.Close()).To(Succeed())
		Expect(second.Close()).To(Succeed())
		later := MustSucceed(s.OpenWriter(state.WriterConfig{
			Keys: keys, Clock: 9, Start: 200,
		}))
		Expect(later.Close()).To(Succeed())
		seals := switchTo(s, a, 2)
		ids := make([]uint64, len(seals))
		for i, seal := range seals {
			ids[i] = seal.Segment
		}
		Expect(ids).To(ConsistOf(ia, ib))
	})

	It("Should never give out segment 0", func() {
		a := &appended{}
		s, err := state.Open(config(a), &state.Snapshot{})
		if err != nil {
			Expect(err).To(MatchError(state.ErrRefused))
			return
		}
		a.s = s
		run(s)
		w := MustSucceed(
			s.OpenWriter(state.WriterConfig{Keys: keys, Clock: 5, Start: 10}),
		)
		Expect(MustSucceed(w.NewSegment(5))).ToNot(BeZero())
	})
})
