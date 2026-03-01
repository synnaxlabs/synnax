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

var _ = Describe("Flag", func() {
	Describe("FlagPos", func() {
		It("Should set a bit flag at a particular position", func() {
			var (
				b    byte
				flag bit.FlagPos = 7
			)
			b = flag.Set(b, true)
			Expect(flag.Get(b)).To(BeTrue())
			Expect(b).To(Equal(byte(128)))
		})
		It("Should be a no-op when setting out of bounds", func() {
			var (
				b    byte
				flag bit.FlagPos = 128
			)
			b = flag.Set(b, true)
			Expect(b).To(Equal(byte(0)))
			Expect(flag.Get(b)).To(BeFalse())
		})
	})

})
