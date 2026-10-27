// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alignment_test

import (
	"math"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Alignment", func() {
	Describe("Leading", func() {
		It("Should offset the domain index from ZeroLeading", func() {
			Expect(alignment.Leading(3, 7)).To(Equal(
				telem.NewAlignment(alignment.ZeroLeading+3, 7),
			))
		})
	})

	Describe("Counter", func() {
		It("Should start at the domain after start", func() {
			c := alignment.NewCounter(alignment.ZeroLeading)
			Expect(c.Next()).To(Equal(alignment.Leading(1, 0)))
			Expect(c.Next()).To(Equal(alignment.Leading(2, 0)))
		})

		It("Should hand out the last domain index", func() {
			c := alignment.NewCounter(math.MaxUint32 - 1)
			Expect(c.Next()).To(Equal(telem.NewAlignment(math.MaxUint32, 0)))
		})

		It("Should return ErrExhausted once every domain is used", func() {
			c := alignment.NewCounter(math.MaxUint32 - 1)
			MustSucceed(c.Next())
			Expect(c.Next()).Error().To(MatchError(alignment.ErrExhausted))
			Expect(c.Next()).Error().To(MatchError(alignment.ErrExhausted))
		})

		It("Should hand out distinct domains to concurrent callers", func() {
			var (
				c        = alignment.NewCounter(alignment.ZeroLeading)
				mu       sync.Mutex
				wg       sync.WaitGroup
				received = make(set.Set[telem.Alignment])
			)
			for range 100 {
				wg.Go(func() {
					defer GinkgoRecover()
					a := MustSucceed(c.Next())
					mu.Lock()
					defer mu.Unlock()
					received.Add(a)
				})
			}
			wg.Wait()
			Expect(received).To(HaveLen(100))
		})
	})
})
