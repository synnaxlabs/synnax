// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BoolBitPacking", func() {
	// The reference vector below must produce the exact same packed bytes in every
	// language's frame codec. A divergence here is a cross-language wire-format bug.
	Describe("Reference Vector", func() {
		samples := []byte{1, 0, 1, 1, 0, 0, 0, 1, 1}
		expected := []byte{0x8D, 0x01}

		It("packs LSB-first", func() {
			Expect(packBoolBits(samples)).To(Equal(expected))
		})

		It("unpacks LSB-first", func() {
			Expect(unpackBoolBits(expected, len(samples))).To(Equal(samples))
		})
	})

	Describe("Round-Trip", func() {
		DescribeTable("preserves canonical bytes",
			func(samples []byte) {
				packed := packBoolBits(samples)
				Expect(packed).To(HaveLen((len(samples) + 7) / 8))
				Expect(unpackBoolBits(packed, len(samples))).To(Equal(samples))
			},
			Entry("empty", []byte{}),
			Entry("one bit", []byte{1}),
			Entry("seven bits", []byte{1, 0, 1, 1, 0, 0, 1}),
			Entry("eight bits", []byte{1, 0, 1, 0, 1, 0, 1, 0}),
			Entry("nine bits", []byte{1, 0, 1, 0, 1, 0, 1, 0, 1}),
			Entry("sixteen bits", []byte{1, 0, 1, 1, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1}),
		)
	})
})
