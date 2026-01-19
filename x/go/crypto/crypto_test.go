// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crypto_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/crypto"
)

var _ = Describe("Crypto", func() {
	Describe("Cipher", func() {
		Context("When the cipher is successful", func() {
			DescribeTable(
				"It computes the correct Caesar cipher",
				func(entry, distance, numDigits, expected int) {
					Expect(crypto.Cipher(entry, distance, numDigits)).
						To(BeEquivalentTo(expected))
				},
				Entry("simple shift without wraparound", 5, 2, 1, 3),
				Entry("simple shift with wraparound", 1, 2, 1, 9),
				Entry("no shift", 7, 0, 1, 7),
				Entry("larger numDigits without wraparound", 123, 23, 3, 100),
				Entry("larger numDigits with wraparound", 5, 7, 3, 998),
				Entry("negative distance", 5, -1, 1, 6),
				Entry("negative distance with multiple wraparounds", 5, -91, 1, 6),
			)
		},
		)
		When("The inputs are invalid", func() {
			DescribeTable(
				"it should return with error",
				func(entry, distance, numDigits int) {
					Expect(crypto.Cipher(entry, distance, numDigits)).
						Error().To(HaveOccurred())
				},
				Entry("negative entry", -1, 1, 1),
				Entry("negative numDigits", 1, 1, -2),
				Entry("entry exceeds limit for numDigits", 1000, 1, 3),
				Entry("zero numDigits", 5, 1, 0),
				Entry("negative numDigits", 1, 1, -1),
			)
		})
	})
})
