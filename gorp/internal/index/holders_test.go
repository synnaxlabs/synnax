// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	"cmp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
)

var _ = Describe("Holders of one value", func() {
	engines := map[string]func() engine{
		"Lookup": func() engine { return index.NewLookup[*scope, int, string]() },
		"Sorted": func() engine {
			return index.NewSorted[*scope, int, string](cmp.Compare[int])
		},
		"Multi": func() engine {
			return singleValue{index.NewMulti[*scope, int, string]()}
		},
	}
	for name, newEngine := range engines {
		Describe(name, func() {
			var e engine
			BeforeEach(func() {
				e = newEngine()
				insert, finish := e.Populate()
				for _, k := range []int{1, 2, 3} {
					insert(k, "a")
				}
				finish(nil)
			})

			It("Should keep the other holders when the first one leaves", func() {
				e.Delete(1)
				Expect(e.Get(nil, "a")).To(ConsistOf(2, 3))
				e.Delete(3)
				Expect(e.Get(nil, "a")).To(Equal([]int{2}))
				e.Delete(2)
				Expect(e.Get(nil, "a")).To(BeEmpty())
			})

			It("Should move a holder to another value", func() {
				e.Set(1, "b")
				Expect(e.Get(nil, "a")).To(ConsistOf(2, 3))
				Expect(e.Get(nil, "b")).To(Equal([]int{1}))
			})

			It("Should keep the holders of a value through every remove order", func() {
				for k := range 5 {
					e.Set(10+k, "h")
				}
				e.Delete(10)
				Expect(e.Get(nil, "h")).To(ConsistOf(11, 12, 13, 14))
				e.Delete(12)
				Expect(e.Get(nil, "h")).To(ConsistOf(11, 13, 14))
				e.Delete(14)
				e.Set(15, "h")
				Expect(e.Get(nil, "h")).To(ConsistOf(11, 13, 15))
				e.Delete(11)
				e.Delete(13)
				e.Delete(15)
				Expect(e.Get(nil, "h")).To(BeEmpty())
				e.Set(16, "h")
				Expect(e.Get(nil, "h")).To(ConsistOf(16))
				e.Set(0, "h")
				e.Delete(16)
				Expect(e.Get(nil, "h")).To(ConsistOf(0))
				e.Delete(99)
				Expect(e.Get(nil, "h")).To(ConsistOf(0))
			})

			It("Should keep the staged holders of a value when the first one leaves",
				func() {
					s := &scope{}
					for _, k := range []int{20, 21, 22} {
						e.Stage(s, k, "s")
					}
					e.Stage(s, 20, "t")
					Expect(e.Get(s, "s")).To(ConsistOf(21, 22))
					e.StageDelete(s, 22)
					Expect(e.Get(s, "s")).To(Equal([]int{21}))
					s.end(true)
					Expect(e.Get(nil, "s")).To(Equal([]int{21}))
					Expect(e.Get(nil, "t")).To(Equal([]int{20}))
				})
		})
	}
})
