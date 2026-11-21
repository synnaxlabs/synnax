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
	"strconv"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var (
		a *appended
		s *state.State
	)

	start := func(h *state.Snapshot) {
		GinkgoHelper()
		a = &appended{}
		s = MustSucceed(state.Open(config(a), h))
		a.s, a.seq = s, h.Sequence
		run(s)
	}

	// durable hands the state records as durable at the next sequence, as the log owner
	// does after their frame is synced.
	durable := func(records ...log.Record) {
		GinkgoHelper()
		Expect(a.durable(records)).To(Succeed())
		seq := a.last()
		Expect(MustSucceed(s.Snapshot()).Sequence).To(BeNumerically(">=", seq))
		Expect(s.Err()).ToNot(HaveOccurred())
	}

	open := func(cfg state.WriterConfig) *state.Writer {
		GinkgoHelper()
		return MustSucceed(s.OpenWriter(cfg))
	}

	clockWriter := func(start int64, bounded bool, keys ...uint32) state.WriterConfig {
		cfg := state.WriterConfig{
			Clock: keys[0], Start: telem.TimeStamp(start),
			Bounded: bounded,
		}
		for _, k := range keys {
			cfg.Keys = append(cfg.Keys, state.WriterKey{Key: k, DataType: 4})
		}
		return cfg
	}

	It("Should give out segment identities from the durable limit, after a Reserve of "+
		"16 and then of 1,024", func() {
		start(head(5, 3))
		keys := make([]uint32, 0, 20)
		for k := range uint32(20) {
			keys = append(keys, k+1)
		}
		w := open(clockWriter(0, false, keys...))
		for i, k := range keys {
			Expect(w.NewSegment(k)).To(Equal(uint64(5 + i)))
		}
		Expect(a.reserves()).To(Equal([]log.Reserve{
			{Segments: 21, Alignments: 19},
			{Segments: 1045, Alignments: 19},
		}))
	})

	It("Should give out segment 1 and alignment number 1 first in a new database",
		func() {
			start(head(0, 0))
			w := open(clockWriter(0, false, 1))
			id := MustSucceed(w.NewSegment(1))
			Expect(id).To(Equal(uint64(1)))
			Expect(w.NewAlignment()).To(Equal(uint32(1)))
			durable(create(id, 1, 1))
		})

	It("Should give out alignment numbers from the durable limit, after a Reserve",
		func() {
			start(head(5, 3))
			w := open(clockWriter(0, false, 1))
			for i := range uint32(17) {
				Expect(w.NewAlignment()).To(Equal(3 + i))
			}
			Expect(a.reserves()).To(Equal([]log.Reserve{
				{Segments: 21, Alignments: 19},
				{Segments: 21, Alignments: 1043},
			}))
		})

	It("Should refuse a new stretch once every stored alignment number is out", func() {
		start(head(5, 1<<31-1))
		w := open(clockWriter(0, false, 1))
		Expect(w.NewAlignment()).To(Equal(uint32(1<<31 - 1)))
		Expect(w.NewAlignment()).Error().To(MatchError(state.ErrExhausted))
	})

	It("Should fail the requests that wait for a Reserve that the log refuses for "+
		"room, and let no commit name its values", func() {
		start(head(5, 3))
		a.refuse(errors.Wrap(log.ErrFull, "no room"))
		w := open(clockWriter(0, false, 1))
		Expect(w.NewAlignment()).Error().To(MatchError(log.ErrFull))
		Expect(w.NewSegment(1)).Error().To(MatchError(log.ErrFull))
		Expect(w.Commit(commit(1, 3, 0, 100, 100, slot(5, 8)))).To(SatisfyAll(
			MatchError(state.ErrRefused),
			MatchError(ContainSubstring("alignment number 3, not from 1 to below the "+
				"limit 3")),
		))
		a.refuse(nil)
		Expect(w.NewAlignment()).To(Equal(uint32(3)))
		Expect(w.NewSegment(1)).To(Equal(uint64(5)))
		Expect(a.reserves()).To(HaveExactElements(
			log.Reserve{Segments: 21, Alignments: 19},
			log.Reserve{Segments: 21, Alignments: 19},
			log.Reserve{Segments: 21, Alignments: 19},
		), "two refused, then one durable")
		Expect(s.Err()).ToNot(HaveOccurred())
	})

	Describe("Commit", func() {
		var (
			w         *state.Writer
			clk, data uint64
			number    uint32
		)
		BeforeEach(func() {
			start(head(5, 3))
			w = open(clockWriter(100, false, 5, 6))
			clk, data = MustSucceed(w.NewSegment(5)), MustSucceed(w.NewSegment(6))
			number = MustSucceed(w.NewAlignment())
		})

		first := func() *log.Commit {
			return commit(2, number, 0, 100, 110, slot(clk, 16), slot(data, 8))
		}

		It("Should make a commit visible at once, and durable once the log has it",
			func() {
				before := s.Channel(5)
				Expect(w.Commit(first())).To(Succeed())
				Expect(s.Channel(5).Segments[0].Pointers).
					To(Equal([]state.Pointer{pointer(number, 0, 100, 110, 0, 16, 2)}))
				Expect(before.Segments[0].Pointers).To(BeEmpty(), "copy on write")
				Expect(MustSucceed(s.Snapshot()).Channels).To(BeEmpty())
				durable(create(clk, 5, 5), create(data, 6, 5), first())
				snap := MustSucceed(s.Snapshot())
				Expect(snap.Channels).To(HaveLen(2))
				Expect(snap.Channels[1].Segments[0].Pointers).
					To(Equal([]state.Pointer{pointer(number, 0, 100, 110, 0, 8, 2)}))
			})

		It("Should extend the pointer of a commit whose alignment follows it", func() {
			Expect(w.Commit(first())).To(Succeed())
			Expect(w.Commit(commit(1, number, 2, 120, 120, slot(clk, 24),
				slot(data, 12)))).To(Succeed())
			Expect(s.Channel(6).Segments[0].Pointers).
				To(Equal([]state.Pointer{pointer(number, 0, 100, 120, 0, 12, 3)}))
		})

		It("Should refuse a commit over stored data", func() {
			Expect(w.Commit(first())).To(Succeed())
			n := MustSucceed(w.NewAlignment())
			Expect(w.Commit(commit(1, n, 0, 110, 110, slot(clk, 24),
				slot(data, 12)))).To(SatisfyAll(
				MatchError(state.ErrConflict),
				MatchError(
					ContainSubstring("overlaps its stored data from 100 to 110"),
				),
			))
		})

		It("Should refuse a commit over stored data that a backfill wrote earlier in "+
			"its segment", func() {
			Expect(w.Commit(first())).To(Succeed())
			n := MustSucceed(w.NewAlignment())
			Expect(w.Commit(commit(1, n, 0, 50, 50, slot(clk, 24), slot(data, 12)))).
				To(Succeed())
			n = MustSucceed(w.NewAlignment())
			Expect(w.Commit(commit(1, n, 0, 50, 50, slot(clk, 32), slot(data, 16)))).
				To(SatisfyAll(
					MatchError(state.ErrConflict),
					MatchError(
						ContainSubstring("overlaps its stored data from 50 to 50"),
					),
				))
		})

		It("Should refuse a writer that starts inside stored data that a backfill "+
			"wrote", func() {
			Expect(w.Commit(first())).To(Succeed())
			n := MustSucceed(w.NewAlignment())
			Expect(w.Commit(commit(2, n, 0, 40, 60, slot(clk, 32), slot(data, 16)))).
				To(Succeed())
			Expect(w.Close()).To(Succeed())
			Expect(s.OpenWriter(clockWriter(50, true, 5))).Error().To(SatisfyAll(
				MatchError(state.ErrConflict),
				MatchError(ContainSubstring("inside its stored data from 40 to 60")),
			))
		})

		DescribeTable(
			"Should refuse a commit that breaks a value rule",
			func(c func() *log.Commit, msg string) {
				Expect(w.Commit(c())).To(SatisfyAll(
					MatchError(state.ErrRefused),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("no slot", func() *log.Commit { return commit(1, number, 0, 1, 1) },
				"commit with no slot"),
			Entry("one sample with two stamps", func() *log.Commit {
				return commit(1, number, 0, 100, 101, slot(clk, 8))
			}, "commit of 1 samples from 100 to 101"),
			Entry("a number the state did not give out", func() *log.Commit {
				return commit(1, 900, 0, 100, 100, slot(clk, 8))
			}, "alignment number 900, not from 1 to below the limit"),
			Entry("a segment the writer does not hold", func() *log.Commit {
				return commit(1, number, 0, 100, 100, slot(99, 8))
			}, "names segment 99, which the writer does not hold"),
			Entry("a length that does not grow", func() *log.Commit {
				return commit(1, number, 0, 100, 100, slot(clk, 0))
			}, "length 0, not above 0"),
			Entry("fewer bytes than samples", func() *log.Commit {
				return commit(4, number, 0, 100, 110, slot(clk, 2))
			}, "pointer of 4 samples in 2 bytes"),
		)

		It("Should refuse a commit whose stamps go back inside its stretch, and keep "+
			"its pointer", func() {
			Expect(w.Commit(first())).To(Succeed())
			Expect(w.Commit(commit(1, number, 2, 50, 50, slot(clk, 24),
				slot(data, 12)))).To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(ContainSubstring("at 50 goes back inside its stretch, "+
					"which reaches 110")),
			))
			Expect(s.Channel(5).Segments[0].Pointers).
				To(Equal([]state.Pointer{pointer(number, 0, 100, 110, 0, 16, 2)}))
		})

		It("Should take back a segment whose Create the log never got, with no Seal",
			func() {
				Expect(w.Commit(first())).To(Succeed())
				durable(create(clk, 5, 5))
				Expect(w.TakeBack()).To(Equal([]log.Seal{{Segment: clk}}))
				Expect(s.Channel(6).Segments).To(BeEmpty())
				Expect(s.TakeDead()).To(Equal([]uint64{data}))
				durable(&log.Seal{Segment: clk})
			})

		It("Should seal the held segment of a key at its committed length", func() {
			Expect(w.Commit(first())).To(Succeed())
			Expect(w.Seal(6)).To(Equal(log.Seal{Segment: data, Length: 8, TailCRC: 24}))
			Expect(s.Channel(6).Segments[0].Sealed).To(BeTrue())
			Expect(w.Seal(6)).Error().To(MatchError(ContainSubstring(
				"writer holds no open segment of key 6",
			)))
			Expect(w.NewSegment(6)).To(BeNumerically(">", data))
		})

		It("Should take visible samples that no record covers back, and seal at the "+
			"durable length", func() {
			Expect(w.Commit(first())).To(Succeed())
			durable(create(clk, 5, 5), create(data, 6, 5), first())
			Expect(w.Commit(commit(1, number, 2, 120, 120, slot(clk, 24),
				slot(data, 12)))).To(Succeed())
			Expect(w.TakeBack()).To(Equal([]log.Seal{
				{Segment: clk, Length: 16, TailCRC: 48},
				{Segment: data, Length: 8, TailCRC: 24},
			}))
			seg := s.Channel(6).Segments[0]
			Expect(seg.Sealed).To(BeTrue())
			Expect(seg.Length).To(Equal(uint32(8)))
			Expect(seg.Pointers).
				To(Equal([]state.Pointer{pointer(number, 0, 100, 110, 0, 8, 2)}))
		})

		It(
			"Should fail the state on a durable commit whose create the log lacks",
			func() {
				Expect(w.Commit(first())).To(Succeed())
				durable(create(clk, 5, 5))
				Expect(a.durable([]log.Record{first()})).To(Succeed())
				Eventually(s.Err).Should(MatchError(ContainSubstring(
					"commit names segment " + strconv.FormatUint(data, 10) +
						", which no create made",
				)))
			},
		)

		It("Should refuse a commit to a segment that the writer sealed", func() {
			Expect(w.Commit(first())).To(Succeed())
			MustSucceed(w.Seal(6))
			Expect(w.Commit(commit(1, number, 2, 120, 120, slot(clk, 24),
				slot(data, 12)))).To(SatisfyAll(
				MatchError(state.ErrRefused),
				MatchError(ContainSubstring("which the writer does not hold")),
			))
		})

		It("Should fail the state on a durable record that breaks a rule", func() {
			Expect(a.durable([]log.Record{create(0, 9, 9)})).To(Succeed())
			Eventually(s.Err).Should(MatchError(state.ErrRefused))
			Expect(w.Commit(first())).To(MatchError(ContainSubstring("state failed")))
			Expect(a.durable(nil)).To(MatchError(state.ErrRefused))
		})
	})

	Describe("Claims", func() {
		BeforeEach(func() { start(head(5, 3)) })

		It("Should refuse a second writer that is not bounded on a key", func() {
			open(clockWriter(100, false, 5))
			Expect(s.OpenWriter(clockWriter(200, false, 5))).Error().To(SatisfyAll(
				MatchError(state.ErrConflict),
				MatchError(
					ContainSubstring("reaches the claim of another writer from 100"),
				),
			))
		})

		It("Should refuse a bounded writer that starts inside another claim", func() {
			open(clockWriter(100, false, 5))
			Expect(s.OpenWriter(clockWriter(150, true, 5))).Error().
				To(MatchError(state.ErrConflict))
		})

		It(
			"Should open a bounded writer before every claim, and refuse its "+
				"commit that "+
				"reaches one",
			func() {
				open(clockWriter(100, false, 5))
				b := open(clockWriter(50, true, 5))
				id := MustSucceed(b.NewSegment(5))
				n := MustSucceed(b.NewAlignment())
				Expect(b.Commit(commit(2, n, 0, 60, 70, slot(id, 16)))).To(Succeed())
				Expect(b.Commit(commit(2, n, 2, 90, 100, slot(id, 32)))).
					To(MatchError(ContainSubstring(
						"reaches the claim of another writer from 100",
					)))
			},
		)

		It(
			"Should open a live writer over a bounded claim, and refuse the "+
				"commit that "+
				"reaches it",
			func() {
				open(clockWriter(50, true, 5))
				w := open(clockWriter(40, false, 5))
				id := MustSucceed(w.NewSegment(5))
				n := MustSucceed(w.NewAlignment())
				Expect(w.Commit(commit(2, n, 0, 45, 50, slot(id, 16)))).
					To(MatchError(state.ErrConflict))
			},
		)

		It("Should refuse a writer that starts inside stored data", func() {
			w := open(clockWriter(100, false, 5))
			id := MustSucceed(w.NewSegment(5))
			n := MustSucceed(w.NewAlignment())
			Expect(w.Commit(commit(2, n, 0, 100, 110, slot(id, 16)))).To(Succeed())
			Expect(w.Close()).To(Succeed())
			Expect(s.OpenWriter(clockWriter(105, true, 5))).Error().To(SatisfyAll(
				MatchError(state.ErrConflict),
				MatchError(ContainSubstring("inside its stored data from 100 to 110")),
			))
		})

		It("Should move a live claim back to its first stamp, over time no one wrote",
			func() {
				w := open(clockWriter(100, false, 5))
				id := MustSucceed(w.NewSegment(5))
				n := MustSucceed(w.NewAlignment())
				Expect(w.Commit(commit(2, n, 0, 50, 60, slot(id, 16)))).To(Succeed())
				Expect(s.OpenWriter(clockWriter(70, true, 5))).Error().
					To(MatchError(state.ErrConflict))
			})

		It("Should let another writer claim a key after Close", func() {
			w := open(clockWriter(100, false, 5))
			Expect(w.Close()).To(Succeed())
			open(clockWriter(200, false, 5))
		})

		It("Should pin the clock of a writer of data against stored stamps, with no "+
			"conflict with the clock's writer", func() {
			open(clockWriter(100, false, 5))
			d := open(state.WriterConfig{
				Keys:  []state.WriterKey{{Key: 6, DataType: 4}},
				Clock: 5,
				Start: 100,
			})
			id := MustSucceed(d.NewSegment(6))
			n := MustSucceed(d.NewAlignment())
			Expect(d.Commit(commit(1, n, 0, 150, 150, slot(id, 8)))).To(Succeed())
		})
	})

	Describe("Takeover", func() {
		BeforeEach(func() { start(head(5, 3)) })

		It("Should give a later writer the open segment of an earlier one once its "+
			"commits are durable", func() {
			w := open(clockWriter(100, false, 5))
			id := MustSucceed(w.NewSegment(5))
			n := MustSucceed(w.NewAlignment())
			c := commit(2, n, 0, 100, 110, slot(id, 16))
			Expect(w.Commit(c)).To(Succeed())
			durable(create(id, 5, 5), c)
			Expect(w.Close()).To(Succeed())
			later := open(clockWriter(200, false, 5))
			held, ok := MustSucceed2(later.Held(5))
			Expect(ok).To(BeTrue())
			Expect(held.ID).To(Equal(id))
			Expect(held.Length).To(Equal(uint32(16)))
			n2 := MustSucceed(later.NewAlignment())
			Expect(later.Commit(commit(1, n2, 0, 200, 200, slot(id, 24)))).To(Succeed())
		})

		It(
			"Should not give a later writer a segment with commits the log lacks",
			func() {
				w := open(clockWriter(100, false, 5))
				id := MustSucceed(w.NewSegment(5))
				n := MustSucceed(w.NewAlignment())
				Expect(w.Commit(commit(2, n, 0, 100, 110, slot(id, 16)))).To(Succeed())
				Expect(w.Close()).To(Succeed())
				later := open(clockWriter(200, false, 5))
				_, ok := MustSucceed2(later.Held(5))
				Expect(ok).To(BeFalse())
			},
		)

		It("Should seal the open segments that no writer holds at a switch, and log "+
			"their seals", func() {
			w := open(clockWriter(100, false, 5, 6))
			a5, a6 := MustSucceed(w.NewSegment(5)), MustSucceed(w.NewSegment(6))
			n := MustSucceed(w.NewAlignment())
			c := commit(1, n, 0, 100, 100, slot(a5, 8), slot(a6, 4))
			Expect(w.Commit(c)).To(Succeed())
			durable(create(a5, 5, 5), create(a6, 6, 5), c)
			Expect(w.Seal(6)).To(Equal(log.Seal{Segment: a6, Length: 4, TailCRC: 12}))
			Expect(w.Close()).To(Succeed())
			holder := open(clockWriter(200, false, 7))
			MustSucceed(holder.NewSegment(7))
			Expect(s.Switch(4)).To(Succeed())
			Eventually(a.switched).Should(HaveLen(1))
			Expect(
				a.seals(),
			).To(Equal([]log.Seal{{Segment: a5, Length: 8, TailCRC: 24}}))
			Expect(s.Channel(5).Segments[0].Sealed).To(BeTrue())
			Expect(s.Channel(7).Segments[0].Sealed).To(BeFalse())
			Expect(a.switched()[0].Channels[0].Segments[0].Sealed).
				To(BeFalse(), "the seal is not durable at the switch")
		})

		It("Should take the snapshot of a switch after every durable entry before it "+
			"and before any after it", func() {
			w := open(clockWriter(100, false, 5))
			id := MustSucceed(w.NewSegment(5))
			n := MustSucceed(w.NewAlignment())
			c := commit(1, n, 0, 100, 100, slot(id, 8))
			Expect(w.Commit(c)).To(Succeed())
			Expect(a.durable([]log.Record{create(id, 5, 5), c})).To(Succeed())
			seq := a.last()
			Expect(s.Switch(4)).To(Succeed())
			Expect(a.durable([]log.Record{&log.Intent{Key: 9}})).To(Succeed())
			Eventually(a.switched).Should(HaveLen(1))
			snap := a.switched()[0]
			Expect(snap.Sequence).To(Equal(seq))
			Expect(snap.Channels).To(HaveLen(1))
			Expect(snap.Intents).To(BeEmpty())
		})

		It("Should drop at a switch an unheld open segment that no durable Create "+
			"made, and give it out as dead", func() {
			w := open(clockWriter(100, false, 5))
			id := MustSucceed(w.NewSegment(5))
			Expect(w.Close()).To(Succeed())
			Expect(s.Switch(4)).To(Succeed())
			Eventually(a.switched).Should(HaveLen(1))
			Expect(a.seals()).To(BeEmpty())
			Expect(s.Channel(5).Segments).To(BeEmpty())
			Expect(s.TakeDead()).To(Equal([]uint64{id}))
			Expect(s.TakeDead()).To(BeEmpty())
		})

		It("Should fail a request that comes after the state failed", func() {
			w := open(clockWriter(100, false, 5))
			fault := s.Durable(a.last()+5, []log.Record{&log.Intent{Key: 1}})
			Expect(fault).To(Succeed())
			Eventually(s.Err).Should(MatchError(ContainSubstring("follows entry")))
			Expect(s.Snapshot()).Error().To(MatchError(s.Err()))
			Expect(w.Held(5)).Error().To(MatchError(s.Err()))
		})
	})

	It("Should keep its durable view exact with writers, the log, and readers at once",
		func() {
			start(head(1, 1))
			var (
				wg      sync.WaitGroup
				mu      sync.Mutex
				entries [][]log.Record
			)
			for k := range uint32(4) {
				wg.Go(func() {
					defer GinkgoRecover()
					w := open(clockWriter(0, false, 10+k))
					id := MustSucceed(w.NewSegment(10 + k))
					n := MustSucceed(w.NewAlignment())
					recs := make([]log.Record, 0, 51)
					recs = append(recs, create(id, 10+k, 10+k))
					for i := range uint32(50) {
						c := commit(1, n, i, int64(i), int64(i), slot(id, 8*(i+1)))
						Expect(w.Commit(c)).To(Succeed())
						recs = append(recs, c)
						Expect(s.Channel(10 + k)).ToNot(BeNil())
					}
					mu.Lock()
					entries = append(entries, recs)
					mu.Unlock()
				})
			}
			wg.Wait()
			for _, e := range entries {
				durable(e...)
			}
			snap := MustSucceed(s.Snapshot())
			Expect(snap.Channels).To(HaveLen(4))
			for _, ch := range snap.Channels {
				Expect(ch.Segments[0].Pointers).To(HaveLen(1))
				Expect(ch.Segments[0].Pointers[0].Samples).To(Equal(uint32(50)))
				Expect(s.Channel(ch.Key).Segments).To(Equal(ch.Segments))
			}
		})
})
