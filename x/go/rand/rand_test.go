// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rand_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/rand"
)

var _ = Describe("Rand", func() {
	Describe("ApplySink", func() {
		var m map[int]int
		BeforeEach(func() {
			m = map[int]int{
				1: 2,
				3: 4,
				5: 6,
			}
		})
		Describe("Keys", func() {
			It("Should return a random key", func() {
				key := rand.MapKey(m)
				Expect(key > 0).To(BeTrue())
				Expect(key < 6).To(BeTrue())
			})
			It("Should return the zero type when the map is empty", func() {
				m = map[int]int{}
				key := rand.MapKey(m)
				Expect(key).To(BeZero())
			})
		})
		Describe("Value", func() {
			It("Should return a random value", func() {
				value := rand.MapValue(m)
				Expect(value > 0).To(BeTrue())
				Expect(value <= 6).To(BeTrue())
			})
		})
		Describe("Element", func() {
			It("Should return a random element", func() {
				key, value := rand.MapElem(m)
				Expect(key > 0).To(BeTrue())
				Expect(value <= 6).To(BeTrue())
				Expect(key < 6).To(BeTrue())
				Expect(value > 0).To(BeTrue())
			})
		})
		Describe("SubMap", func() {
			It("Should return a random sub map of the provided size", func() {
				m := rand.SubMap(m, 2)
				Expect(len(m)).To(Equal(2))
			})
		})
	})

	Describe("Slice", func() {
		Describe("sub Slice", func() {
			It("Should return random sub-slice", func() {
				value := rand.SubSlice([]int{1, 2, 3, 4, 5, 6}, 2)
				Expect(len(value)).To(Equal(2))
				Expect(value[0] != value[1]).To(BeTrue())
			})
			It("Should return the slice itself", func() {
				slc := []int{1, 2, 3, 4, 5, 6}
				value := rand.SubSlice(slc, 20)
				Expect(len(value)).To(Equal(6))
				Expect(value).To(Equal(slc))
			})
		})
	})

	Describe("Element", func() {
		It("Should not introduce any duplicate indexes", func() {
			slc := make([]int, 1000)
			for i := range 1000 {
				slc[i] = i
			}
			sub := rand.SubSlice(slc, 800)
			Expect(lo.Uniq(sub)).To(Equal(sub))
		})
	})
})
