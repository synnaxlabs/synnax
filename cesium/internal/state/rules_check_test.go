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

var _ = Describe("Check: state", func() {
	var (
		a   *appended
		s   *state.State
		seq uint64
		fed int
	)

	start := func(h *state.Snapshot) {
		GinkgoHelper()
		a, fed = &appended{}, 0
		s = MustSucceed(state.Open(config(a), h))
		seq = h.Sequence
		run(s)
	}

	// log hands the state every Reserve that it appended, then an entry of records, as
	// the log owner does after their frames are synced.
	logged := func(records ...log.Record) {
		GinkgoHelper()
		for _, r := range a.reserves()[fed:] {
			seq++
			Expect(s.Durable(seq, []log.Record{&r})).To(Succeed())
			fed++
		}
		seq++
		Expect(s.Durable(seq, records)).To(Succeed())
		MustSucceed(s.Snapshot())
	}

	live := func(start int64, bounded bool, key uint32) state.WriterConfig {
		return state.WriterConfig{
			Keys:    []state.WriterKey{{Key: key, DataType: 4}},
			Clock:   key,
			Start:   telem.TimeStamp(start),
			Bounded: bounded,
		}
	}

	It("Should give out segment identities from 1 on a head that reserved none",
		func() {
			start(&state.Snapshot{})
			w := MustSucceed(s.OpenWriter(live(100, false, 5)))
			Expect(w.NewSegment(5)).To(BeNumerically(">=", 1))
		})

	It("Should refuse a commit over stored data that a backfill wrote earlier in "+
		"the segment", func() {
		start(head(5, 3))
		w := MustSucceed(s.OpenWriter(live(100, false, 5)))
		id := MustSucceed(w.NewSegment(5))
		n1 := MustSucceed(w.NewAlignment())
		Expect(w.Commit(commit(2, n1, 0, 100, 110, slot(id, 16)))).To(Succeed())
		n2 := MustSucceed(w.NewAlignment())
		Expect(w.Commit(commit(2, n2, 0, 50, 60, slot(id, 32)))).To(Succeed())
		n3 := MustSucceed(w.NewAlignment())
		Expect(w.Commit(commit(2, n3, 0, 55, 58, slot(id, 48)))).To(SatisfyAll(
			MatchError(state.ErrConflict),
			MatchError(ContainSubstring("overlaps its stored data from 50 to 60")),
		))
	})

	It("Should refuse a writer that starts inside stored data that a backfill wrote",
		func() {
			start(head(5, 3))
			w := MustSucceed(s.OpenWriter(live(100, false, 5)))
			id := MustSucceed(w.NewSegment(5))
			n1 := MustSucceed(w.NewAlignment())
			c1 := commit(2, n1, 0, 100, 110, slot(id, 16))
			Expect(w.Commit(c1)).To(Succeed())
			n2 := MustSucceed(w.NewAlignment())
			c2 := commit(2, n2, 0, 50, 60, slot(id, 32))
			Expect(w.Commit(c2)).To(Succeed())
			logged(create(id, 5, 5), c1, c2)
			Expect(w.Close()).To(Succeed())
			Expect(s.OpenWriter(live(55, true, 5))).Error().To(SatisfyAll(
				MatchError(state.ErrConflict),
				MatchError(ContainSubstring("inside its stored data from 50 to 60")),
			))
		})

	It("Should refuse a commit whose stamps go back inside its stretch", func() {
		start(head(5, 3))
		w := MustSucceed(s.OpenWriter(live(100, false, 5)))
		id := MustSucceed(w.NewSegment(5))
		n := MustSucceed(w.NewAlignment())
		Expect(w.Commit(commit(2, n, 0, 100, 110, slot(id, 16)))).To(Succeed())
		err := w.Commit(commit(1, n, 2, 50, 50, slot(id, 24)))
		Expect(s.Channel(5).Segments[0].Pointers).To(HaveEach(
			Satisfy(func(p state.Pointer) bool { return p.First <= p.Last }),
		))
		Expect(err).To(SatisfyAny(
			MatchError(state.ErrConflict),
			MatchError(state.ErrRefused),
		))
	})

	DescribeTable("Should open again on the head of a state that it replayed",
		func(records ...log.Record) {
			r := MustSucceed(state.Open(config(&appended{}), head(20, 10)))
			if r.Replay(1, append([]log.Record{create(5, 5, 5)}, records...)) != nil {
				return
			}
			run(r)
			snap := MustSucceed(r.Snapshot())
			Expect(state.Open(config(&appended{}), snap)).Error().ToNot(HaveOccurred())
		},
		Entry("a commit over a backfill in the same segment",
			commit(2, 1, 0, 100, 110, slot(5, 16)),
			commit(2, 2, 0, 50, 60, slot(5, 32)),
			commit(2, 3, 0, 55, 58, slot(5, 48)),
		),
		Entry("a commit whose stamps go back inside its stretch",
			commit(2, 1, 0, 100, 110, slot(5, 16)),
			commit(1, 1, 2, 50, 50, slot(5, 24)),
		),
		Entry("a commit of fewer bytes than samples",
			commit(4, 1, 0, 100, 110, slot(5, 2)),
		),
	)

	It("Should release every open segment that a writer took over at its Close",
		func() {
			start(head(5, 3))
			first := MustSucceed(s.OpenWriter(live(10, true, 5)))
			idA := MustSucceed(first.NewSegment(5))
			nA := MustSucceed(first.NewAlignment())
			cA := commit(2, nA, 0, 10, 20, slot(idA, 16))
			Expect(first.Commit(cA)).To(Succeed())
			second := MustSucceed(s.OpenWriter(live(100, true, 5)))
			idB := MustSucceed(second.NewSegment(5))
			nB := MustSucceed(second.NewAlignment())
			cB := commit(2, nB, 0, 100, 110, slot(idB, 16))
			Expect(second.Commit(cB)).To(Succeed())
			logged(create(idA, 5, 5), create(idB, 5, 5), cA, cB)
			Expect(first.Close()).To(Succeed())
			Expect(second.Close()).To(Succeed())
			later := MustSucceed(s.OpenWriter(live(200, false, 5)))
			Expect(later.Close()).To(Succeed())
			seals := MustSucceed(s.SealUnheld())
			Expect(seals).To(HaveLen(2))
		})

	It("Should keep the state when the log takes the Seals of a switch after a "+
		"writer whose commit was refused closes", func() {
		start(head(5, 3))
		MustSucceed(s.OpenWriter(live(100, false, 5)))
		b := MustSucceed(s.OpenWriter(live(50, true, 5)))
		id := MustSucceed(b.NewSegment(5))
		n := MustSucceed(b.NewAlignment())
		Expect(b.Commit(commit(2, n, 0, 60, 120, slot(id, 16)))).
			To(MatchError(state.ErrConflict))
		Expect(b.Close()).To(Succeed())
		// The switch logs a Seal for each open segment that no writer holds.
		seals := MustSucceed(s.SealUnheld())
		records := make([]log.Record, len(seals))
		for i := range seals {
			records[i] = &seals[i]
		}
		logged(records...)
		Expect(s.Err()).ToNot(HaveOccurred())
	})

	It("Should keep the state when the log takes the Seals of a take back of a "+
		"segment whose Create it never got", func() {
		start(head(5, 3))
		w := MustSucceed(s.OpenWriter(live(100, false, 5)))
		id := MustSucceed(w.NewSegment(5))
		n := MustSucceed(w.NewAlignment())
		Expect(w.Commit(commit(2, n, 0, 100, 110, slot(id, 16)))).To(Succeed())
		// The sync of the segment fails: the writer sends no Commit, takes its samples
		// back, and logs the Seals that TakeBack returns.
		seals := MustSucceed(w.TakeBack())
		records := make([]log.Record, len(seals))
		for i := range seals {
			records[i] = &seals[i]
		}
		logged(records...)
		Expect(s.Err()).ToNot(HaveOccurred())
	})
})
