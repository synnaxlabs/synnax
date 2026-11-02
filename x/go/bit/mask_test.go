// Copyright 2026 Synnax Labs, Inc.
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
})
