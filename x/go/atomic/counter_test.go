// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package atomic_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/atomic"
)

var _ = Describe("SeqNum", func() {
	Describe("Int32Counter", func() {
		It("Should increment the counter atomically", func() {
			wg := sync.WaitGroup{}
			c := atomic.Int32Counter{}
			wg.Add(10)
			for i := 0; i < 10; i++ {
				go func() {
					defer wg.Done()
					for i := 0; i < 1000; i++ {
						if i == 0 {
							c.Add(1)
						} else if i == 1 {
							c.Add(1)
						} else {
							c.Add(1)
						}
					}
				}()
			}
			wg.Wait()
			Expect(c.Value()).To(Equal(int32(10000)))
		})
	})
	Describe("Int64Counter", func() {
		It("Should increment the counter atomically", func() {
			wg := sync.WaitGroup{}
			c := atomic.Int64Counter{}
			wg.Add(10)
			for i := 0; i < 10; i++ {
				go func() {
					defer wg.Done()
					for i := 0; i < 1000; i++ {
						c.Add(1)
					}
				}()
			}
			wg.Wait()
			Expect(c.Value()).To(Equal(int64(10000)))
		})
	})
})
