// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bit_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/bit"
)

var _ = Describe("Mask Test", func() {
	Describe("Set & Get", func() {
		It("Should properly set & get positions", func() {
			mask := bit.Mask128{}
			for i := range mask.Cap() {
				mask = mask.Set(i, i%2 == 0)
			}
			for i := range mask.Cap() {
				if i%2 == 0 {
					Expect(mask.Get(i)).To(BeTrue())
				} else {
					Expect(mask.Get(i)).To(BeFalse())
				}
			}
		})
		It("Should panic on out of bounds", func() {
			mask := &bit.Mask128{}
			Expect(func() { mask.Set(-1, true) }).To(Panic())
			Expect(func() { mask.Set(128, true) }).To(Panic())
			Expect(func() { mask.Get(-1) }).To(Panic())
			Expect(func() { mask.Get(128) }).To(Panic())
		})
	})
	Describe("TrueCount", func() {
		It("Should return the number of true values in the mask", func() {
			mask := bit.Mask128{}
			for i := range mask.Cap() {
				mask = mask.Set(i, i%2 == 0)
			}
			Expect(mask.TrueCount()).To(Equal(64))
		})
	})
	Describe("Swap", func() {
		It("Should swap two bits in the mask", func() {
			mask := bit.Mask128{}
			mask = mask.Set(10, true)
			mask = mask.Set(20, false)
			mask = mask.Swap(10, 20)
			Expect(mask.Get(10)).To(BeFalse())
			Expect(mask.Get(20)).To(BeTrue())
		})

		It("Should swap bits across different bytes", func() {
			mask := bit.Mask128{}
			mask = mask.Set(7, true)  // Last bit in first byte
			mask = mask.Set(8, false) // First bit in second byte
			mask = mask.Swap(7, 8)
			Expect(mask.Get(7)).To(BeFalse())
			Expect(mask.Get(8)).To(BeTrue())
		})

		It("Should handle swapping a bit with itself", func() {
			mask := bit.Mask128{}
			mask = mask.Set(15, true)
			mask = mask.Swap(15, 15)
			Expect(mask.Get(15)).To(BeTrue())

			mask = mask.Set(25, false)
			mask.Swap(25, 25)
			Expect(mask.Get(25)).To(BeFalse())
		})

		It("Should panic on out of bounds", func() {
			mask := bit.Mask128{}
			Expect(func() { mask.Swap(-1, 10) }).To(Panic())
			Expect(func() { mask.Swap(10, 128) }).To(Panic())
			Expect(func() { mask.Swap(200, 10) }).To(Panic())
			Expect(func() { mask.Swap(10, -5) }).To(Panic())
		})
	})
})
