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

var _ = Describe("Load", func() {
	It("Should report a publish after the loads", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		run(s)
		Eventually(func() *state.Channel { return s.Channel(5) }).ShouldNot(BeNil())
		chs := make([]*state.Channel, 2)
		count := s.Load([]uint32{6, 5}, chs)
		Expect(chs[0].Key).To(Equal(uint32(6)))
		Expect(chs[1].Key).To(Equal(uint32(5)))
		Expect(s.Unchanged(count)).To(BeTrue())
		Expect(s.Durable(10, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
		Eventually(func() bool { return s.Unchanged(count) }).Should(BeFalse())
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
