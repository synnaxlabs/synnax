// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state

import (
	"context"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Pair between its loads", func() {
	It("Should load both Channels again when a step publishes between its loads",
		func() {
			// Key 6 holds data that its clock, key 5, times.
			ptr := Pointer{
				First: 10, Last: 30, Alignment: telem.NewAlignment(1, 0), Samples: 3,
			}
			clockPtr, dataPtr := ptr, ptr
			clockPtr.Length, dataPtr.Length = 24, 12
			s := MustSucceed(Open(Config{
				Versions:   set.New[uint16](1),
				Version:    1,
				Generation: 1,
				Threshold:  0.2,
				Append:     func(*log.Entry) {},
				Executor:   func(f func()) { go f() },
				OnHead:     func(uint64, *Snapshot) {},
			}, &Snapshot{
				Sequence: 9,
				Limits:   Limits{Segments: 20, Alignments: 10},
				Channels: []*Channel{
					{Key: 5, Segments: []Segment{{
						ID: 2, Clock: 5, Version: 1, DataType: 7, Sealed: true,
						Length: 24, TailCRC: 1, Pointers: []Pointer{clockPtr},
					}}},
					{Key: 6, Segments: []Segment{{
						ID: 3, Clock: 5, Version: 1, DataType: 4, Length: 12,
						TailCRC: 3, Pointers: []Pointer{dataPtr},
					}}},
				},
			}))
			// One step drops the data key and then its clock, after Pair loaded the
			// data key.
			var once sync.Once
			s.betweenLoads = func() {
				once.Do(func() {
					Expect(s.Durable(10, []log.Record{
						&log.Drop{Key: 6},
						&log.Drop{Key: 5},
					})).To(Succeed())
					Eventually(func() *Channel { return s.Channel(5) }).Should(BeNil())
				})
			}
			ctx, cancel := context.WithCancel(context.Background())
			ran := make(chan error, 1)
			go func() { ran <- s.Run(ctx) }()
			DeferCleanup(func() {
				Expect(s.Close()).To(Succeed())
				Eventually(ran).Should(Receive(BeNil()))
				cancel()
			})
			Eventually(func() *Channel { return s.Channel(6) }).ShouldNot(BeNil())
			data, clocks, count := s.Pair(6, 5)
			Expect(data).To(BeNil(), "the data of a step before its clock")
			Expect(clocks).To(BeNil())
			Expect(s.Unchanged(count)).To(BeTrue())
		})
})
