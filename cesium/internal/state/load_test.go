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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Load", func() {
	It("Should report a publish after the loads", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		run(s)
		Eventually(func() *state.Channel { return s.Channel(5) }).ShouldNot(BeNil())
		chs := make([]*state.Channel, 2)
		count := s.Load([]uint32{6, 5}, chs)
		Expect(chs[0].Key).To(Equal(uint32(6)))
		Expect(chs[1].Key).To(Equal(uint32(5)))
		Expect(s.Count()).To(Equal(count))
		Expect(s.Durable(10, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
		Eventually(s.Count).ShouldNot(Equal(count))
	})

	It("Should set the Channel of a key with no segment to nil", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		run(s)
		Eventually(func() *state.Channel { return s.Channel(6) }).ShouldNot(BeNil())
		chs := []*state.Channel{nil, {Key: 9}}
		s.Load([]uint32{6, 9}, chs)
		Expect(chs[0].Key).To(Equal(uint32(6)))
		Expect(chs[1]).To(BeNil())
	})
})

var _ = Describe("Cuts", func() {
	It("Should count the Tombstones of a key in what readers see", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		run(s)
		MustSucceed(s.Snapshot())
		Expect(s.Channel(6).Cuts).To(BeZero())
		Expect(s.Durable(10, []log.Record{&log.Tombstone{
			Key: 6, Start: 0, End: 25,
			Right: cutOf(2, 30),
		}})).To(Succeed())
		Eventually(func() uint64 { return s.Channel(6).Cuts }).Should(Equal(uint64(1)))
		Expect(s.Channel(5).Cuts).To(BeZero())
	})

	Describe("Of a clock with a new segment", func() {
		var (
			a  *appended
			s  *state.State
			w  *state.Writer
			id uint64
		)
		BeforeEach(func() {
			a = &appended{}
			s = MustSucceed(state.Open(config(a), sample()))
			a.s, a.seq = s, sample().Sequence
			run(s)
			w = MustSucceed(
				s.OpenWriter(context.Background(), clockWriter(100, false, 9)),
			)
			id = MustSucceed(w.NewSegment(9))
		})
		commitOne := func() *log.Commit {
			GinkgoHelper()
			n := MustSucceed(w.NewAlignment())
			c := commit(1, n, 0, 100, 100, slot(id, 8))
			Expect(w.Commit(c)).To(Succeed())
			Expect(s.Channel(9).Segments[0].Pointers).To(HaveLen(1))
			return c
		}

		DescribeTable("Should count each take back of visible samples, and no other "+
			"change",
			func(change func(), cuts uint64) {
				change()
				Expect(s.Channel(9).Cuts).To(Equal(cuts))
			},
			Entry("a take back of a segment with a durable Create", func() {
				commitOne()
				Expect(a.durable([]log.Record{create(id, 9, 9)})).To(Succeed())
				Expect(w.TakeBack()).To(HaveLen(1))
				Expect(s.Channel(9).Segments[0].Pointers).To(BeEmpty())
			}, uint64(1)),
			Entry("a take back of a segment with no durable Create", func() {
				commitOne()
				Expect(w.TakeBack()).To(BeEmpty())
				Expect(s.Channel(9).Segments).To(BeEmpty())
			}, uint64(1)),
			Entry("an unknown take back of a segment with no durable Create", func() {
				commitOne()
				Expect(w.TakeBackUnknown()).To(Succeed())
				Expect(s.Channel(9).Segments[0].Pointers).To(BeEmpty())
			}, uint64(1)),
			Entry("a take back of a segment whose commits are all durable", func() {
				c := commitOne()
				Expect(a.durable([]log.Record{create(id, 9, 9), c})).To(Succeed())
				Expect(w.TakeBack()).To(HaveLen(1))
				Expect(s.Channel(9).Segments[0].Pointers).To(HaveLen(1))
			}, uint64(0)),
			Entry("a take back of a segment with no durable Create and no sample",
				func() {
					Expect(w.TakeBack()).To(BeEmpty())
					Expect(s.Channel(9).Segments).To(BeEmpty())
				}, uint64(0)),
			Entry("an unknown take back of a segment with no sample", func() {
				Expect(w.TakeBackUnknown()).To(Succeed())
				Expect(s.Channel(9).Segments[0].Sealed).To(BeTrue())
			}, uint64(0)),
			Entry("a commit", func() { commitOne() }, uint64(0)),
			Entry("a switch that drops the segment, which holds no sample, as no "+
				"durable Create made it", func() {
				Expect(w.Close()).To(Succeed())
				Expect(s.Switch(4)).To(Succeed())
				Eventually(a.switched).Should(HaveLen(1))
				Expect(s.Channel(9).Segments).To(BeEmpty())
			}, uint64(0)),
		)

		It("Should give a read of the clock a new load with the same Cuts after a "+
			"switch drops the segment, so the read keeps loading", func() {
			chs := make([]*state.Channel, 1)
			before := s.Load([]uint32{9}, chs)
			cuts := chs[0].Cuts
			Expect(w.Close()).To(Succeed())
			Expect(s.Switch(4)).To(Succeed())
			Eventually(a.switched).Should(HaveLen(1))
			Expect(s.Load([]uint32{9}, chs)).To(BeNumerically(">", before))
			Expect(chs[0].Segments).To(BeEmpty())
			Expect(chs[0].Cuts).To(Equal(cuts))
		})
	})
})

var _ = Describe("Timed", func() {
	It("Should return the keys of the segments on a clock that a durable Create made",
		func() {
			a := &appended{}
			s := MustSucceed(state.Open(config(a), sample()))
			a.s, a.seq = s, sample().Sequence
			run(s)
			Expect(s.Timed(5)).To(Equal(set.New[uint32](5, 6)))
			Expect(s.Timed(6)).To(BeEmpty())
			w := MustSucceed(
				s.OpenWriter(context.Background(), clockWriter(100, false, 9, 7)),
			)
			id := MustSucceed(w.NewSegment(7))
			Expect(s.Timed(9)).To(BeEmpty(), "no durable Create made the segment")
			Expect(a.durable([]log.Record{create(id, 7, 9)})).To(Succeed())
			Expect(s.Timed(9)).To(Equal(set.New[uint32](7)))
			Expect(a.durable([]log.Record{&log.Drop{Key: 6}})).To(Succeed())
			Expect(s.Timed(5)).To(Equal(set.New[uint32](5)))
		})
})
