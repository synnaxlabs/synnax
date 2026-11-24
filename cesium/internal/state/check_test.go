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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
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
		Expect(s.Replay(10, []log.Record{&log.Check{Segment: 3, Key: 6}})).To(Succeed())
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

	DescribeTable(
		"Should refuse a Check of a segment that cannot take one",
		func(
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
		Entry("no create made", converted, []log.Record{&log.Check{Segment: 9, Key: 6}},
			"check of segment 9, which no create made"),
		Entry(
			"a version 1 segment",
			sample,
			[]log.Record{&log.Check{Segment: 2, Key: 5}},
			"check of segment 2, which is not a sealed version 0 segment",
		),
		Entry("checks already on", converted, []log.Record{
			&log.Check{Segment: 3, Key: 6},
			&log.Check{Segment: 3, Key: 6},
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

var _ = Describe("Keys", func() {
	It("Should return the keys that the state holds segments of, in rising order",
		func() {
			s := MustSucceed(state.Open(config(&appended{}), converted()))
			run(s)
			MustSucceed(s.Snapshot())
			Expect(s.Keys()).To(Equal([]uint32{5, 6}))
		})
})

var _ = Describe("Check in flight", func() {
	var (
		a *appended
		s *state.State
	)
	BeforeEach(func() {
		a = &appended{}
		s = MustSucceed(state.Open(config(a), converted()))
		a.s, a.seq = s, 9
		run(s)
		MustSucceed(s.Snapshot())
	})
	check := &log.Check{Segment: 3, Key: 6}
	// count returns the number of entries that s appended.
	count := func() int {
		a.mu.Lock()
		defer a.mu.Unlock()
		return len(a.entries)
	}
	// begin calls Check with r, and returns the channel of its error once s appended
	// the entry.
	begin := func(r *log.Check, pre ...state.Preload) chan error {
		GinkgoHelper()
		n := count()
		done := make(chan error, 1)
		go func() { done <- s.Check(r, pre...) }()
		Eventually(count).Should(Equal(n + 1))
		return done
	}
	// last returns the records of the last entry that s appended.
	last := func() []log.Record {
		GinkgoHelper()
		a.mu.Lock()
		defer a.mu.Unlock()
		var (
			d   log.Decoder
			out []log.Record
		)
		MustSucceed(d.Reset(a.entries[len(a.entries)-1]))
		for {
			r, ok := MustSucceed2(d.Next())
			if !ok {
				return out
			}
			if c, ok := r.(*log.Check); ok {
				copied := *c
				r = &copied
			}
			out = append(out, r)
		}
	}
	// later calls f, and returns the channel of its error.
	later := func(f func() error) chan error {
		called := make(chan error, 1)
		go func() { called <- f() }()
		return called
	}

	It("Should append the Check in an entry, and return once the entry applies",
		func() {
			done := begin(check)
			Expect(last()).To(Equal([]log.Record{check}))
			Consistently(done).ShouldNot(Receive())
			Expect(s.Channel(6).Segments[0].Checked).To(BeFalse())
			Expect(a.durable(last())).To(Succeed())
			Eventually(done).Should(Receive(BeNil()))
			Expect(s.Channel(6).Segments[0].Checked).To(BeTrue())
		})

	It("Should return the error of a refused entry and leave the checks off", func() {
		done := begin(check)
		Expect(s.Refused(last(), errors.New("log failed"))).To(Succeed())
		Eventually(done).Should(Receive(MatchError("log failed")))
		Expect(s.Channel(6).Segments[0].Checked).To(BeFalse())
	})

	DescribeTable("Should refuse a Check as stale when its segment may change",
		func(change func(), pre []state.Preload, message string) {
			change()
			checked := later(func() error { return s.Check(check, pre...) })
			Eventually(checked).Should(Receive(SatisfyAll(
				MatchError(state.ErrStale),
				MatchError(ContainSubstring(message)),
			)))
		},
		Entry("by leaving the state", func() {
			Expect(a.durable([]log.Record{&log.Tombstone{Key: 6, End: 100}})).
				To(Succeed())
		}, nil, "segment 3 of key 6 left the state"),
		Entry("by a delete of its key",
			func() { MustSucceed(s.BeginDelete(6, 25, 35)) }, nil,
			"delete of key 6 from 25 reads segment 3"),
		Entry("by a delete of its clock",
			func() { MustSucceed(s.BeginDelete(5, 25, 35)) }, nil,
			"delete of key 5 from 25 reads segment 3"),
		Entry("by a Drop of its key",
			func() { Expect(s.BeginDrop(6)).To(Succeed()) }, nil,
			"drop of key 6 reads segment 3"),
		Entry("by a Drop of its clock",
			func() { Expect(s.BeginDrop(5)).To(Succeed()) }, nil,
			"drop of key 5 reads segment 3"),
		Entry("by a stale preload", func() {},
			[]state.Preload{{Key: 6, Generation: 4}},
			"key 6 reads the parts of head 1, not 4"),
	)

	It("Should refuse a Check that names another key than the segment's", func() {
		Expect(s.Check(&log.Check{Segment: 2, Key: 6})).To(SatisfyAll(
			MatchError(state.ErrRefused),
			MatchError(ContainSubstring("check of segment 2 of key 6, which key 5 "+
				"holds")),
		))
	})

	It("Should refuse a Check of a segment that has its checks", func() {
		done := begin(check)
		Expect(a.durable(last())).To(Succeed())
		Eventually(done).Should(Receive(BeNil()))
		Expect(s.Check(check)).To(SatisfyAll(
			MatchError(state.ErrRefused),
			MatchError(ContainSubstring("check of segment 3, which has its checks")),
		))
	})

	It("Should refuse a second Check while one is in flight", func() {
		begin(check)
		Expect(s.Check(&log.Check{Segment: 2, Key: 5})).To(SatisfyAll(
			Not(MatchError(state.ErrStale)),
			MatchError(ContainSubstring("check of segment 2 while the check of "+
				"segment 3 is in flight")),
		))
	})

	DescribeTable("Should defer a request whose check reads the segment until the "+
		"Check applies",
		func(request func() error) {
			done := begin(check)
			called := later(request)
			Consistently(called).ShouldNot(Receive())
			Expect(a.durable(last())).To(Succeed())
			Eventually(done).Should(Receive(BeNil()))
			Eventually(called).Should(Receive(BeNil()))
		},
		Entry("a delete of its key", func() error {
			_, err := s.BeginDelete(6, 5, 15)
			return err
		}),
		Entry("a delete of its clock", func() error {
			_, err := s.BeginDelete(5, 40, 45)
			return err
		}),
		Entry("a Drop of its key", func() error { return s.BeginDrop(6) }),
		Entry("a Drop of its clock", func() error { return s.BeginDrop(5) }),
	)

	It("Should hold a Replace until the Check applies", func() {
		done := begin(check)
		called := later(func() error {
			return s.Replace(&log.Replace{Old: 3, New: 9})
		})
		Consistently(called).ShouldNot(Receive())
		Expect(a.durable(last())).To(Succeed())
		Eventually(done).Should(Receive(BeNil()))
		Eventually(called).Should(Receive(SatisfyAll(
			MatchError(state.ErrStale),
			MatchError(ContainSubstring("segment 3 changed after its copy")),
		)))
	})

	It("Should not defer a delete of a key that the segment does not use", func() {
		begin(check)
		called := later(func() error {
			_, err := s.BeginDelete(8, 0, 10)
			return err
		})
		Eventually(called).Should(Receive(BeNil()))
	})

	It("Should run a deferred request once the log refuses the Check", func() {
		done := begin(check)
		called := later(func() error { return s.BeginDrop(6) })
		Consistently(called).ShouldNot(Receive())
		Expect(s.Refused(last(), errors.New("log failed"))).To(Succeed())
		Eventually(done).Should(Receive(MatchError("log failed")))
		Eventually(called).Should(Receive(BeNil()))
	})

	It("Should keep the record of the segment at a switch until the Check applies",
		func() {
			done := begin(check)
			record := s.Channel(6).Segments[0]
			record.Part = 7
			Expect(s.Parted(
				5,
				[]state.Tag{{Part: 7, Segment: record}},
				[]state.Range{{Key: 6, Oldest: 7, Newest: 7}},
				func() {},
			)).To(Succeed())
			MustSucceed(s.Snapshot())
			Expect(s.Channel(6).Segments).To(HaveLen(1))
			Expect(a.durable(last())).To(Succeed())
			Eventually(done).Should(Receive(BeNil()))
			seg := s.Channel(6).Segments[0]
			Expect(seg.Checked).To(BeTrue())
			Expect(seg.Part).To(BeZero())
			Expect(MustSucceed(s.Snapshot()).Kills).
				To(ContainElement(state.Kill{Key: 6, Segment: 3, Generation: 1}))
		})

	It("Should fail the Check in flight and the requests it deferred at close", func() {
		done := begin(check)
		called := later(func() error { return s.BeginDrop(6) })
		Consistently(called).ShouldNot(Receive())
		Expect(s.Close()).To(Succeed())
		Eventually(done).Should(Receive(MatchError(state.ErrClosed)))
		Eventually(called).Should(Receive(MatchError(state.ErrClosed)))
	})
})

var _ = Describe("Replay of a Check", func() {
	// parted is the record of segment 3 of key 6, which part 7 holds and the state
	// lacks.
	var parted state.Segment
	BeforeEach(func() {
		parted = converted().Channels[1].Segments[0]
		parted.Part = 7
	})
	// open opens a state whose head holds key 6 only in part 7.
	open := func() *state.State {
		GinkgoHelper()
		head := converted()
		head.Channels[1] = &state.Channel{
			Key:   6,
			Parts: &state.Parts{Oldest: 7, Newest: 7},
		}
		return MustSucceed(state.Open(config(&appended{}), head))
	}

	It("Should turn on the checks of a parted segment through its preload", func() {
		s := open()
		Expect(s.Replay(
			10,
			[]log.Record{&log.Check{Segment: 3, Key: 6}},
			state.Preload{Key: 6, Segments: []state.Segment{parted}},
		)).To(Succeed())
		run(s)
		seg := published(s, 6).Segments[0]
		Expect(seg.Checked).To(BeTrue())
		Expect(seg.Part).To(BeZero())
		Expect(MustSucceed(s.Snapshot()).Kills).
			To(Equal([]state.Kill{{Key: 6, Segment: 3, Generation: 1}}))
	})

	It("Should refuse the Check of a parted segment with no preload", func() {
		Expect(open().Replay(10, []log.Record{&log.Check{Segment: 3, Key: 6}})).
			To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(
					ContainSubstring("check of segment 3, which no create made"),
				),
			))
	})
})

var _ = Describe("Preload of a killed record", func() {
	var (
		a      *appended
		s      *state.State
		record state.Segment
		pre    state.Preload
	)
	// A delete buries segment 3 of key 6, which part 7 of head 5 holds, after pre was
	// loaded.
	BeforeEach(func() {
		a = &appended{}
		s = MustSucceed(state.Open(config(a), converted()))
		a.s, a.seq = s, 9
		run(s)
		MustSucceed(s.Snapshot())
		record = s.Channel(6).Segments[0]
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
		Expect(a.durable([]log.Record{&log.Tombstone{Key: 6, End: 100}})).To(Succeed())
		Expect(d.Release()).To(Succeed())
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Segments).To(BeEmpty())
	})

	It("Should refuse a Check of the buried segment as stale", func() {
		checked := make(chan error, 1)
		go func() { checked <- s.Check(&log.Check{Segment: 3, Key: 6}, pre) }()
		Eventually(checked).Should(Receive(SatisfyAll(
			MatchError(state.ErrStale),
			MatchError(ContainSubstring("segment 3 of key 6 left the state")),
		)))
		Expect(s.Channel(6).Segments).To(BeEmpty())
	})

	It("Should refuse a delete that would put the buried record back as stale",
		func() {
			Expect(s.BeginDelete(6, 0, 100, pre)).Error().To(SatisfyAll(
				MatchError(state.ErrStale),
				MatchError(
					ContainSubstring("key 6 reads the part record of segment 3, "+
						"which a kill took after the preload"),
				),
			))
			Expect(s.Channel(6).Segments).To(BeEmpty())
		})
})

var _ = Describe("Rewrite of version 0 segments", func() {
	// head returns converted with segment 3 of key 6 at half dead, checked as given.
	head := func(checked bool) *state.Snapshot {
		h := converted()
		seg := &h.Channels[1].Segments[0]
		seg.Length, seg.Checked = 24, checked
		return h
	}
	// open opens the state of h, with version 0 rewritable as given, and runs it.
	open := func(a *appended, h *state.Snapshot, rewritable bool) *state.State {
		GinkgoHelper()
		cfg := config(a)
		cfg.Version0Rewritable = rewritable
		s := MustSucceed(state.Open(cfg, h))
		a.s, a.seq = s, 9
		run(s)
		return s
	}

	DescribeTable("Should make a version 0 segment with dead bytes a candidate only "+
		"when it has its checks and version 0 is rewritable",
		func(checked, rewritable bool, want []state.Candidate) {
			s := open(&appended{}, head(checked), rewritable)
			Expect(s.Candidates()).To(Equal(want))
		},
		Entry("checked and rewritable", true, true,
			[]state.Candidate{{Key: 6, Segment: 3}}),
		Entry("checked, not rewritable", true, false, []state.Candidate{}),
		Entry("unchecked and rewritable", false, true, []state.Candidate{}),
	)

	It("Should make a version 0 segment a candidate once its Check applies", func() {
		a := &appended{}
		s := open(a, head(false), true)
		Expect(a.durable([]log.Record{&log.Check{Segment: 3, Key: 6}})).To(Succeed())
		Expect(s.Candidates()).To(Equal([]state.Candidate{{Key: 6, Segment: 3}}))
	})

	It("Should create the copy of a version 0 segment at the version of new segments",
		func() {
			a := &appended{}
			s := open(a, head(true), true)
			id := MustSucceed(s.NewID())
			done := make(chan error, 1)
			go func() {
				done <- s.Replace(&log.Replace{
					Old:    3,
					New:    id,
					Length: 12,
					Pointers: []log.Pointer{{
						Length:    12,
						Samples:   3,
						Alignment: uint64(telem.NewAlignment(1, 0)),
						First:     10,
						Last:      30,
					}},
				})
			}()
			Eventually(func() int {
				a.mu.Lock()
				defer a.mu.Unlock()
				return len(a.entries)
			}).Should(BeNumerically(">=", 2))
			a.mu.Lock()
			var d log.Decoder
			MustSucceed(d.Reset(a.entries[len(a.entries)-1]))
			a.mu.Unlock()
			r, ok := MustSucceed2(d.Next())
			Expect(ok).To(BeTrue())
			created, ok := r.(*log.Create)
			Expect(ok).To(BeTrue())
			Expect(created.Version).To(Equal(uint16(1)))
			Expect(s.Close()).To(Succeed())
			Eventually(done).Should(Receive(MatchError(state.ErrClosed)))
		})
})
