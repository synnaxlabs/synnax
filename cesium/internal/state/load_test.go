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

	It("Should count a take back of the samples of a key, and no commit", func() {
		a := &appended{}
		s := MustSucceed(state.Open(config(a), sample()))
		a.s, a.seq = s, sample().Sequence
		run(s)
		w := MustSucceed(s.OpenWriter(context.Background(), clockWriter(100, false, 9)))
		id := MustSucceed(w.NewSegment(9))
		n := MustSucceed(w.NewAlignment())
		Expect(w.Commit(commit(1, n, 0, 100, 100, slot(id, 8)))).To(Succeed())
		Expect(a.durable([]log.Record{create(id, 9, 9)})).To(Succeed())
		Expect(s.Channel(9).Cuts).To(BeZero())
		Expect(w.TakeBack()).To(HaveLen(1))
		Expect(s.Channel(9).Segments[0].Pointers).To(BeEmpty())
		Expect(s.Channel(9).Cuts).To(Equal(uint64(1)))
	})
})
