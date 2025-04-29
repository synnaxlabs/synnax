// Copyright 2025 Synnax Labs, Inc.
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
	"sync/atomic"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xatomic "github.com/synnaxlabs/x/atomic"
)

var _ = Describe("Counter", func() {
	Describe("Int32Counter", func() {
		It("Should increment the counter atomically", func() {
			wg := sync.WaitGroup{}
			c := xatomic.Int32Counter{}
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
			c := xatomic.Int64Counter{}
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

		Describe("Set", func() {
			It("Should set the counter value", func() {
				c := xatomic.Int64Counter{}
				c.Set(42)
				Expect(c.Value()).To(Equal(int64(42)))
			})
		})
	})

})

func BenchmarkABC(b *testing.B) {
	ch := make(chan struct{})
	for i := 0; i < b.N; i++ {
		select {
		case <-ch:
		default:
		}
	}
}

func BenchmarkBCD(b *testing.B) {
	v := &atomic.Bool{}
	v.Store(true)
	for i := 0; i < b.N; i++ {
		v.Load()
	}
}
