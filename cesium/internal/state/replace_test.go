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
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rewrites", func() {
	var (
		// seal seals segment 3 of key 6 at its 12 bytes.
		seal = &log.Seal{Segment: 3, Length: 12, TailCRC: 3}
		// cut takes the middle sample out of segment 3, which leaves 8 of its 12 bytes.
		cut = &log.Tombstone{
			Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
		}
	)
	// candidates returns the candidates of a rewrite in a snapshot of s.
	candidates := func(s *state.State) []uint64 {
		GinkgoHelper()
		return MustSucceed(s.Snapshot()).Candidates
	}
	created := &log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 4}
	// replace returns a Replace of segment 3 by segment 9 that packs its two live
	// samples, changed by edit.
	replace := func(edit func(r *log.Replace)) *log.Replace {
		r := &log.Replace{Old: 3, New: 9, Length: 8, TailCRC: 7}
		r.Pointers = []log.Pointer{
			{
				Length: 4, Samples: 1, Alignment: uint64(telem.NewAlignment(1, 0)),
				First: 10, Last: 10,
			},
			{
				Offset: 4, Length: 4, Samples: 1,
				Alignment: uint64(telem.NewAlignment(1, 2)), First: 30, Last: 30,
			},
		}
		if edit != nil {
			edit(r)
		}
		return r
	}

	Describe("Candidates", func() {
		// open opens the sample state with cfg, replays records, and runs it.
		open := func(cfg state.Config, records ...log.Record) *state.State {
			GinkgoHelper()
			s := MustSucceed(state.Open(cfg, sample()))
			Expect(s.Replay(10, records)).To(Succeed())
			run(s)
			return s
		}

		It("Should name a sealed segment that a Tombstone leaves above the dead share",
			func() {
				Expect(candidates(open(config(&appended{}), seal, cut))).
					To(Equal([]uint64{3}))
			})

		It("Should name a segment that the replay creates, seals, and cuts", func() {
			Expect(candidates(open(
				config(&appended{}),
				&log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 4},
				commit(1, 3, 0, 60, 60, slot(9, 4)),
				commit(1, 3, 1, 70, 70, slot(9, 8)),
				&log.Seal{Segment: 9, Length: 8, TailCRC: 24},
				&log.Tombstone{Key: 6, Start: 60, End: 61, Right: &log.Cut{
					Segment:   9,
					Offset:    4,
					Alignment: uint64(telem.NewAlignment(3, 1)),
					Stamp:     70,
				}},
			))).To(Equal([]uint64{9}))
		})

		It("Should not name a segment whose dead share is at most the threshold",
			func() {
				cfg := config(&appended{})
				cfg.Threshold = 0.5
				Expect(candidates(open(cfg, seal, cut))).To(BeEmpty())
			})

		DescribeTable(
			"Should name a segment that a Tombstone cut while open once it seals",
			func(sealing log.Record, want []uint64) {
				s := open(config(&appended{}), cut)
				Expect(candidates(s)).To(BeEmpty())
				Expect(s.Durable(11, []log.Record{sealing})).To(Succeed())
				Expect(candidates(s)).To(Equal(want))
			},
			Entry("by a Seal", seal, []uint64{3}),
			// The Restart also seals clock segment 4, a candidate of encoding.
			Entry("by a Restart", &log.Restart{}, []uint64{3, 4}),
		)

		It("Should name a segment of the head above the dead share at open", func() {
			head := sample()
			head.Channels[0].Segments[0].Pointers = []state.Pointer{
				pointer(1, 0, 10, 10, 0, 8, 1),
			}
			s := MustSucceed(state.Open(config(&appended{}), head))
			run(s)
			Expect(candidates(s)).To(Equal([]uint64{2}))
		})

		It("Should not name a segment of another format version", func() {
			head := sample()
			seg := &head.Channels[0].Segments[0]
			seg.Version, seg.Kept = 2, false
			seg.Pointers = []state.Pointer{pointer(1, 0, 10, 10, 0, 8, 1)}
			cfg := config(&appended{})
			cfg.Versions = set.New[uint16](1, 2)
			s := MustSucceed(state.Open(cfg, head))
			run(s)
			Expect(candidates(s)).To(BeEmpty())
		})

		DescribeTable("Should take out a candidate that leaves the state",
			func(leave log.Record) {
				s := open(config(&appended{}), seal, cut)
				Expect(s.Durable(11, []log.Record{leave})).To(Succeed())
				Expect(candidates(s)).To(BeEmpty())
			},
			Entry("by a Tombstone that empties it",
				&log.Tombstone{Key: 6, Start: 0, End: 100}),
			Entry("by a Drop of its key", &log.Drop{Key: 6}),
		)
	})

	Describe("Replace", func() {
		var s *state.State
		BeforeEach(func() {
			s = MustSucceed(state.Open(config(&appended{}), sample()))
			Expect(s.Replay(10, []log.Record{seal, cut})).To(Succeed())
		})
		// expectReplaced expects segment 9 in place of segment 3.
		expectReplaced := func() {
			GinkgoHelper()
			MustSucceed(s.Snapshot())
			Expect(s.Channel(6).Segments).To(Equal([]state.Segment{{
				ID: 9, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 8,
				TailCRC: 7, Pointers: []state.Pointer{
					pointer(1, 0, 10, 10, 0, 4, 1),
					pointer(1, 2, 30, 30, 4, 4, 1),
				},
			}}))
			Expect(s.TakeDead()).To(Equal([]state.Dead{{Segment: 3, Key: 6}}))
			Expect(candidates(s)).To(BeEmpty())
		}

		It("Should seal the new segment with the copied pointers in place of the old "+
			"one", func() {
			run(s)
			Expect(candidates(s)).To(Equal([]uint64{3}))
			Expect(s.Durable(11, []log.Record{created, replace(nil)})).To(Succeed())
			expectReplaced()
		})

		It("Should apply a Replace in replay as at runtime", func() {
			Expect(s.Replay(11, []log.Record{created, replace(nil)})).To(Succeed())
			run(s)
			expectReplaced()
		})

		DescribeTable(
			"Should refuse a Replace that breaks a rule of the copy",
			func(before []log.Record, edit func(r *log.Replace), message string) {
				records := append(slices.Clone(before), replace(edit))
				Expect(s.Replay(11, records)).To(SatisfyAll(
					refused(),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry(
				"by a new segment of another key",
				[]log.Record{
					&log.Create{Segment: 9, Key: 5, Clock: 5, Version: 1, DataType: 4},
				},
				nil,
				"replace of segment 3 by 9, which differs in key, clock, type, or version",
			),
			Entry(
				"by a new segment of another clock",
				[]log.Record{
					&log.Create{Segment: 9, Key: 6, Clock: 8, Version: 1, DataType: 4},
				},
				nil,
				"replace of segment 3 by 9, which differs in key, clock, type, or version",
			),
			Entry(
				"by a new segment of another data type",
				[]log.Record{
					&log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 7},
				},
				nil,
				"replace of segment 3 by 9, which differs in key, clock, type, or version",
			),
			Entry("by a segment that no Create made", nil, nil,
				"replace of segment 3 by 9, which is not a new segment"),
			Entry("by a new segment that holds bytes",
				[]log.Record{created, commit(1, 3, 0, 60, 60, slot(9, 4))}, nil,
				"replace of segment 3 by 9, which is not a new segment"),
			Entry("of an unknown old segment", []log.Record{created},
				func(r *log.Replace) { r.Old = 99 },
				"replace of segment 99, which is not sealed"),
			Entry("of an open old segment", []log.Record{created},
				func(r *log.Replace) { r.Old = 4 },
				"replace of segment 4, which is not sealed"),
			Entry("by other samples in a pointer", []log.Record{created},
				func(r *log.Replace) { r.Pointers[1].Samples = 2 },
				"replace of segment 3, whose pointers differ from the copy"),
			Entry("by other stamps in a pointer", []log.Record{created},
				func(r *log.Replace) { r.Pointers[0].Last = 11 },
				"replace of segment 3, whose pointers differ from the copy"),
			Entry("by another count of pointers", []log.Record{created},
				func(r *log.Replace) { r.Pointers = r.Pointers[:1] },
				"replace of segment 3, whose pointers differ from the copy"),
			Entry("by pointers that overlap", []log.Record{created},
				func(r *log.Replace) { r.Pointers[1].Offset = 2 },
				"replace of segment 3 by 9, whose pointers overlap or fall out of "+
					"stream order"),
			Entry("by pointers out of stream order", []log.Record{created},
				func(r *log.Replace) {
					r.Pointers[0].Offset, r.Pointers[1].Offset = 4, 0
				},
				"replace of segment 3 by 9, whose pointers overlap or fall out of "+
					"stream order"),
			Entry("by pointers that end past its length", []log.Record{created},
				func(r *log.Replace) { r.Length = 7 },
				"replace of segment 3 by 9, whose pointers end past its length 7"),
		)

		DescribeTable("Should refuse a Replace by a segment that a writer made",
			func(closed bool) {
				a := &appended{}
				s := MustSucceed(state.Open(config(a), sample()))
				Expect(s.Replay(10, []log.Record{seal, cut})).To(Succeed())
				a.s, a.seq = s, 10
				run(s)
				w := MustSucceed(s.OpenWriter(state.WriterConfig{
					Clock: 5,
					Start: 100,
					Keys:  []state.WriterKey{{Key: 6, DataType: 4}},
				}))
				id := MustSucceed(w.NewSegment(6))
				records := make([]log.Record, 0, 2)
				if closed {
					Expect(w.Close()).To(Succeed())
				} else {
					records = append(records, &log.Create{
						Segment: id, Key: 6, Clock: 5, Version: 1, DataType: 4,
					})
				}
				records = append(records, replace(func(r *log.Replace) { r.New = id }))
				Expect(a.durable(records)).To(Succeed())
				Eventually(s.Err).Should(SatisfyAll(
					refused(),
					MatchError(ContainSubstring(fmt.Sprintf(
						"replace of segment 3 by %d, which is not a new segment",
						id,
					))),
				))
			},
			Entry("that holds it", false),
			Entry("that closed with no Create of it", true),
		)
	})

	Describe("Replace in flight", func() {
		var (
			a *appended
			s *state.State
		)
		BeforeEach(func() {
			a = &appended{}
			s = MustSucceed(state.Open(config(a), sample()))
			Expect(s.Replay(10, []log.Record{seal, cut})).To(Succeed())
			a.s, a.seq = s, 10
			run(s)
		})
		// count returns the number of entries that s appended.
		count := func() int {
			a.mu.Lock()
			defer a.mu.Unlock()
			return len(a.entries)
		}
		// begin calls Replace with r, and returns the channel of its error once s
		// appended the entry.
		begin := func(r *log.Replace) chan error {
			GinkgoHelper()
			n := count()
			done := make(chan error, 1)
			go func() { done <- s.Replace(r, 1) }()
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
				if rep, ok := r.(*log.Replace); ok {
					copied := *rep
					copied.Pointers = slices.Clone(rep.Pointers)
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

		It("Should list each candidate with its key and why it is one", func() {
			Expect(
				s.Candidates(),
			).To(Equal([]state.Candidate{{Key: 6, Segment: 3, Reclaimable: true}}))
		})

		It("Should give out the identity of a new segment under a durable Reserve",
			func() {
				Expect(s.NewID()).To(Equal(uint64(20)))
				Expect(s.NewID()).To(Equal(uint64(21)))
				a.refuse(errors.New("log failed"))
				for range 200 {
					if _, err := s.NewID(); err != nil {
						Expect(err).To(MatchError("log failed"))
						return
					}
				}
				Fail("no identity waited for a refused Reserve")
			})

		It("Should append a Create of the new segment and the Replace in one entry, "+
			"and return once the entry applies", func() {
			done := begin(replace(nil))
			Expect(last()).To(Equal([]log.Record{created, replace(nil)}))
			Consistently(done).ShouldNot(Receive())
			Expect(a.durable(last())).To(Succeed())
			Eventually(done).Should(Receive(BeNil()))
			Expect(s.Candidates()).To(BeEmpty())
			Expect(s.Channel(6).Segments[0].ID).To(Equal(uint64(9)))
		})

		It("Should return the error of a refused entry and keep the old segment",
			func() {
				done := begin(replace(nil))
				Expect(s.Refused(last(), errors.New("log failed"))).To(Succeed())
				Eventually(done).Should(Receive(MatchError("log failed")))
				Expect(s.Candidates()).
					To(Equal([]state.Candidate{{Key: 6, Segment: 3, Reclaimable: true}}))
			})

		DescribeTable("Should refuse a Replace as stale when its segment may change",
			func(change func(), r *log.Replace, message string) {
				change()
				Expect(s.Replace(r, 1)).To(SatisfyAll(
					MatchError(state.ErrStale),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("by pointers that differ from the copy", func() {},
				replace(func(r *log.Replace) { r.Pointers = r.Pointers[:1] }),
				"segment 3 changed after its copy"),
			Entry("by leaving the state", func() {
				Expect(a.durable([]log.Record{&log.Tombstone{Key: 6, End: 100}})).
					To(Succeed())
			}, replace(nil), "segment 3 changed after its copy"),
			Entry("by a delete of its key",
				func() { MustSucceed(s.BeginDelete(6, 25, 35)) },
				replace(nil), "delete of key 6 from 25 reads segment 3"),
			Entry("by a delete of its clock",
				func() { MustSucceed(s.BeginDelete(5, 25, 35)) },
				replace(nil), "delete of key 5 from 25 reads segment 3"),
			Entry("by a Drop of its key",
				func() { Expect(s.BeginDrop(6)).To(Succeed()) },
				replace(nil), "drop of key 6 reads segment 3"),
			Entry("by a Drop of its clock",
				func() { Expect(s.BeginDrop(5)).To(Succeed()) },
				replace(nil), "drop of key 5 reads segment 3"),
		)

		It("Should refuse a second Replace while one is in flight", func() {
			begin(replace(nil))
			Expect(s.Replace(replace(nil), 1)).To(SatisfyAll(
				Not(MatchError(state.ErrStale)),
				MatchError(ContainSubstring("replace of segment 3 while the "+
					"replace of segment 3 is in flight")),
			))
		})

		DescribeTable("Should defer a request whose check reads the old segment until "+
			"the Replace applies",
			func(request func() error) {
				done := begin(replace(nil))
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

		It("Should not defer a delete of a key that the old segment does not use",
			func() {
				begin(replace(nil))
				called := later(func() error {
					_, err := s.BeginDelete(8, 0, 10)
					return err
				})
				Eventually(called).Should(Receive(BeNil()))
			})

		It("Should run a deferred request once the log refuses the Replace", func() {
			done := begin(replace(nil))
			called := later(func() error { return s.BeginDrop(6) })
			Consistently(called).ShouldNot(Receive())
			Expect(s.Refused(last(), errors.New("log failed"))).To(Succeed())
			Eventually(done).Should(Receive(HaveOccurred()))
			Eventually(called).Should(Receive(BeNil()))
		})

		It("Should fail the Replace in flight and the requests it deferred at close",
			func() {
				done := begin(replace(nil))
				called := later(func() error { return s.BeginDrop(6) })
				Consistently(called).ShouldNot(Receive())
				Expect(s.Close()).To(Succeed())
				Eventually(
					done,
				).Should(Receive(MatchError(ContainSubstring("state is closed"))))
				Eventually(
					called,
				).Should(Receive(MatchError(ContainSubstring("state is closed"))))
			})

		DescribeTable("Should replay a request between the check and the apply as it "+
			"ran live",
			func(request func() error, after log.Record) {
				done := begin(replace(nil))
				replaced := last()
				called := later(request)
				Consistently(called).ShouldNot(Receive())
				Expect(a.durable(replaced)).To(Succeed())
				Eventually(done).Should(Receive(BeNil()))
				Eventually(called).Should(Receive(BeNil()))
				Expect(a.durable([]log.Record{after})).To(Succeed())
				live := MustSucceed(s.Snapshot())
				r := MustSucceed(state.Open(config(&appended{}), sample()))
				Expect(r.Replay(10, []log.Record{seal, cut})).To(Succeed())
				Expect(r.Replay(11, replaced)).To(Succeed())
				Expect(r.Replay(12, []log.Record{after})).To(Succeed())
				run(r)
				Expect(r.Snapshot()).To(Equal(live))
			},
			Entry("a delete", func() error {
				_, err := s.BeginDelete(6, 5, 15)
				return err
			}, &log.Tombstone{Key: 6, Start: 5, End: 15}),
			Entry("a Drop", func() error { return s.BeginDrop(6) }, &log.Drop{Key: 6}),
		)
	})
})

var _ = Describe("Parted candidates", func() {
	It("Should name a parted segment that a Tombstone puts back in the head, and "+
		"name it to the next switch", func() {
		head := sample()
		head.Channels[1] = &state.Channel{
			Key:   6,
			Parts: &state.Parts{Oldest: 7, Newest: 7},
		}
		a := &appended{}
		cfg := config(a)
		cfg.Generation = 5
		s := MustSucceed(state.Open(cfg, head))
		a.s, a.seq = s, head.Sequence
		run(s)
		d := MustSucceed(s.BeginDelete(6, 15, 25, state.Preload{
			Key:        6,
			Generation: 5,
			Segments: []state.Segment{{
				ID: 3, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 12,
				TailCRC: 3, Part: 7,
				Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 12, 3)},
			}},
		}))
		t := &log.Tombstone{
			Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
		}
		Expect(d.Check(t)).To(Succeed())
		Expect(a.durable([]log.Record{t})).To(Succeed())
		Expect(d.Release()).To(Succeed())
		snap := MustSucceed(s.Snapshot())
		Expect(snap.Candidates).To(Equal([]uint64{3}))
		Expect(snap.Kills).To(ContainElement(state.Kill{
			Key: 6, Segment: 3, Generation: 5,
		}))
		Expect(s.Channel(6).Segments).To(HaveLen(1))
		Expect(s.Channel(6).Segments[0].Part).To(BeZero())
		Expect(s.Switch(6)).To(Succeed())
		Eventually(a.switched).Should(HaveLen(1))
		Expect(a.switched()[0].Candidates).To(Equal([]uint64{3}))
	})
})

var _ = Describe("Replace of a put-back record", func() {
	var s *state.State
	// replace replaces segment 3 by segment 9 with the same samples.
	replace := &log.Replace{
		Old: 3, New: 9, Length: 12, TailCRC: 3,
		Pointers: []log.Pointer{{
			Length: 12, Samples: 3, Alignment: uint64(telem.NewAlignment(1, 0)),
			First: 10, Last: 30,
		}},
	}
	BeforeEach(func() {
		head := sample()
		head.Channels[1] = &state.Channel{
			Key:   6,
			Parts: &state.Parts{Oldest: 7, Newest: 7},
		}
		cfg := config(&appended{})
		cfg.Generation = 5
		s = MustSucceed(state.Open(cfg, head))
		run(s)
		record := state.Segment{
			ID: 3, Clock: 5, Version: 1, DataType: 4, Sealed: true, Length: 12,
			TailCRC: 3, Part: 7,
			Pointers: []state.Pointer{pointer(1, 0, 10, 30, 0, 12, 3)},
		}
		MustSucceed(s.BeginDelete(6, 15, 25, state.Preload{
			Key: 6, Generation: 5, Segments: []state.Segment{record},
		}))
	})
	message := "replace of segment 3, which is no candidate of a rewrite"

	It("Should refuse the old segment while a part holds its record", func() {
		Expect(s.Durable(10, []log.Record{
			&log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 4},
			replace,
		})).To(Succeed())
		Eventually(s.Err).Should(SatisfyAll(
			refused(),
			MatchError(ContainSubstring(message)),
		))
	})

	It("Should refuse the check of a Replace of the record as a defect", func() {
		Expect(s.Replace(replace, 1)).To(SatisfyAll(
			Not(MatchError(state.ErrStale)),
			MatchError(ContainSubstring(message)),
		))
	})
})

var _ = Describe("Replace across a switch", func() {
	seal := &log.Seal{Segment: 3, Length: 12, TailCRC: 3}
	created := &log.Create{Segment: 9, Key: 6, Clock: 5, Version: 1, DataType: 4}
	// across seals segment 3 of key 6, applies before, and starts the switch to head 2.
	// Before the Parted of that switch, it applies r live. It then replays r from head
	// 2, whose part 7 holds segment 3 when parted is true. It returns the error of
	// each.
	across := func(
		before []log.Record,
		r *log.Replace,
		parted bool,
	) (live, replayed error) {
		GinkgoHelper()
		a := &appended{}
		s := MustSucceed(state.Open(config(a), sample()))
		Expect(s.Replay(10, append([]log.Record{seal}, before...))).To(Succeed())
		a.s, a.seq = s, 10
		run(s)
		Expect(s.Switch(2)).To(Succeed())
		Eventually(a.switched).Should(HaveLen(1))
		snap := a.switched()[0]
		Expect(a.durable([]log.Record{created, r})).To(Succeed())
		_, live = s.Snapshot()
		head := &state.Snapshot{
			Sequence:   snap.Sequence,
			Limits:     snap.Limits,
			Candidates: snap.Candidates,
		}
		for _, ch := range snap.Channels {
			if ch.Key == 6 && parted {
				ch = &state.Channel{Key: 6, Parts: &state.Parts{
					Oldest: 7, Newest: 7, Generation: 2,
				}}
			}
			head.Channels = append(head.Channels, ch)
		}
		cfg := config(&appended{})
		cfg.Generation = 2
		replay := MustSucceed(state.Open(cfg, head))
		return live, replay.Replay(snap.Sequence+1, []log.Record{created, r})
	}

	It(
		"Should refuse, live and in replay, a Replace of a segment that is no candidate, "+
			"which the switch may part",
		func() {
			live, replayed := across(nil, &log.Replace{
				Old: 3, New: 9, Length: 12, TailCRC: 3,
				Pointers: []log.Pointer{{
					Length: 12, Samples: 3, Alignment: uint64(telem.NewAlignment(1, 0)),
					First: 10, Last: 30,
				}},
			}, true)
			Expect(live).To(refused())
			Expect(replayed).To(refused())
		},
	)

	It("Should apply, live and in replay, a Replace of a candidate, which the switch "+
		"keeps in the head", func() {
		Expect(across([]log.Record{&log.Tombstone{
			Key: 6, Start: 15, End: 25, Left: cutOf(1, 10), Right: cutOf(2, 30),
		}}, &log.Replace{
			Old: 3, New: 9, Length: 8, TailCRC: 7,
			Pointers: []log.Pointer{
				{
					Length: 4, Samples: 1, Alignment: uint64(telem.NewAlignment(1, 0)),
					First: 10, Last: 10,
				},
				{
					Offset: 4, Length: 4, Samples: 1,
					Alignment: uint64(telem.NewAlignment(1, 2)), First: 30, Last: 30,
				},
			},
		}, false)).To(Succeed())
	})
})
