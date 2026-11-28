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
	"context"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Dead", func() {
	var (
		a *appended
		s *state.State
	)
	BeforeEach(func() {
		a = &appended{}
		s = MustSucceed(state.Open(config(a), sample()))
		a.s, a.seq = s, sample().Sequence
		run(s)
		MustSucceed(s.Snapshot())
	})

	// load returns the publish count of a load of keys 5, 6, and 9, and the segments
	// that it sees.
	load := func() (uint64, set.Set[uint64]) {
		chs := make([]*state.Channel, 3)
		count := s.Load([]uint32{5, 6, 9}, chs)
		ids := set.New[uint64]()
		for _, ch := range chs {
			if ch == nil {
				continue
			}
			for _, seg := range ch.Segments {
				ids.Add(seg.ID)
			}
		}
		return count, ids
	}
	durable := func(records ...log.Record) {
		GinkgoHelper()
		Expect(a.durable(records)).To(Succeed())
	}
	// newSegment returns a new open segment of key 9, on clock 9, with no durable
	// Create, and its writer.
	newSegment := func() (*state.Writer, uint64) {
		GinkgoHelper()
		w := MustSucceed(s.OpenWriter(context.Background(), clockWriter(100, false, 9)))
		return w, MustSucceed(w.NewSegment(9))
	}

	DescribeTable("Should give a segment the publish count at the end of the step "+
		"that takes it out of what readers see",
		func(remove func() []uint64) {
			before, _ := load()
			removed := remove()
			dead := a.takeDead(s)
			after, ids := load()
			Expect(dead).To(HaveLen(len(removed)))
			for i, d := range dead {
				Expect(d.Segment).To(Equal(removed[i]))
				Expect(d.Count).To(BeNumerically(">", before))
				Expect(d.Count).To(Equal(after))
				Expect(ids.Contains(d.Segment)).To(BeFalse())
			}
		},
		Entry("a Drop", func() []uint64 {
			durable(&log.Drop{Key: 6})
			return []uint64{3}
		}),
		Entry("a Tombstone that empties a sealed segment", func() []uint64 {
			durable(&log.Tombstone{Key: 6, Start: 0, End: 100})
			durable(&log.Tombstone{Key: 5, Start: 0, End: 35})
			return []uint64{2}
		}),
		Entry("the Seal of a segment that a Tombstone emptied", func() []uint64 {
			durable(&log.Tombstone{Key: 6, Start: 0, End: 100})
			seal := log.Seal{Segment: 3, Length: 12, TailCRC: 3}
			Eventually(a.seals).Should(Equal([]log.Seal{seal}))
			durable(&seal)
			return []uint64{3}
		}),
		Entry("a take back of a segment with no durable Create", func() []uint64 {
			w, id := newSegment()
			Expect(w.TakeBack()).To(BeEmpty())
			return []uint64{id}
		}),
		Entry("a switch of an open segment that no writer holds and no durable Create "+
			"made", func() []uint64 {
			w, id := newSegment()
			Expect(w.Close()).To(Succeed())
			Expect(s.Switch(4)).To(Succeed())
			Eventually(a.switched).Should(HaveLen(1))
			return []uint64{id}
		}),
	)
})

var _ = Describe("Settle", func() {
	It("Should return once the owner ran every request queued before it", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		run(s)
		Expect(s.Durable(10, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
		Expect(s.Settle()).To(Succeed())
		Expect(s.Channel(6)).To(BeNil())
	})

	It("Should return the error of a closed state", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		ctx, cancel := context.WithCancel(context.Background())
		ran := make(chan error, 1)
		go func() { ran <- s.Run(ctx) }()
		Expect(s.Close()).To(Succeed())
		Eventually(ran).Should(Receive(BeNil()))
		cancel()
		Expect(s.Settle()).To(MatchError(ContainSubstring("state is closed")))
	})
})

var _ = Describe("OnDead", func() {
	It("Should get each Dead once the request that took the segment out ends, at a "+
		"count at or past its Count", func() {
		a := &appended{}
		cfg := config(a)
		var (
			s      *state.State
			pushed = make(chan []state.Dead, 1)
			counts = make(chan uint64, 1)
		)
		cfg.OnDead = func(dead []state.Dead) {
			counts <- s.Count()
			pushed <- dead
		}
		s = MustSucceed(state.Open(cfg, sample()))
		run(s)
		Expect(s.Durable(10, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
		var dead []state.Dead
		Eventually(pushed).Should(Receive(&dead))
		Expect(dead).To(HaveExactElements(deadSegment(3, 6, 10)))
		Expect(<-counts).To(BeNumerically(">=", dead[0].Count))
	})

	It("Should get the Deads of a replayed entry before Run", func() {
		a := &appended{}
		s := MustSucceed(state.Open(config(a), sample()))
		Expect(s.Replay(10, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
		a.mu.Lock()
		defer a.mu.Unlock()
		Expect(a.dead).To(HaveExactElements(deadSegment(3, 6, 10)))
	})
})

var _ = Describe("Count", func() {
	It("Should be even between steps, at or below the count of a later Load, and "+
		"grow by two with each step that publishes", func() {
		a := &appended{}
		s := MustSucceed(state.Open(config(a), sample()))
		run(s)
		MustSucceed(s.Snapshot())
		count := s.Count()
		Expect(count % 2).To(BeZero())
		chs := make([]*state.Channel, 1)
		Expect(s.Load([]uint32{6}, chs)).To(Equal(count))
		Expect(s.Durable(10, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
		Eventually(s.Count).Should(Equal(count + 2))
		Expect(s.Load([]uint32{6}, chs)).To(Equal(count + 2))
		Expect(chs[0]).To(BeNil())
	})

	It("Should end after each request that publishes, not after the whole step",
		func() {
			a := &appended{}
			cfg := config(a)
			// gate, once set, holds the next step until it closes.
			var gate atomic.Pointer[chan struct{}]
			cfg.Executor = func(f func()) {
				go func() {
					if g := gate.Load(); g != nil {
						<-*g
					}
					f()
				}()
			}
			var (
				s    *state.State
				even = make(chan bool, 1)
			)
			cfg.OnHead = func(uint64, *state.Snapshot) { even <- s.Count()%2 == 0 }
			head := sample()
			head.Channels[0].Segments[1].Sealed = true
			s = MustSucceed(state.Open(cfg, head))
			run(s)
			MustSucceed(s.Snapshot())
			g := make(chan struct{})
			gate.Store(&g)
			Expect(s.Durable(10, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
			Expect(s.Switch(4)).To(Succeed())
			gate.Store(nil)
			close(g)
			var at bool
			Eventually(even).Should(Receive(&at))
			Expect(at).To(BeTrue(), "the switch runs at an even count")
		})
})
