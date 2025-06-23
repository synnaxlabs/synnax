<<<<<<< HEAD
// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/math"
)

var _ = Describe(
	"Math", func() {
		Describe(
			"IntPow", func() {
				Context(
					"when the exponent is nonnegative", func() {
						DescribeTable(
							"it computes math.IntPow(base, exponent) correctly",
							func(base, exponent, expected int) {
								Expect(math.IntPow(base, exponent)).To(Equal(expected))
							},
							func(base, exponent, expected int) string {
								return fmt.Sprintf(
									"%d ^ %d = %d", base, exponent, expected,
								)
							},
							Entry(nil, -2, 0, 1),
							Entry(nil, -1, 0, 1),
							Entry(nil, 0, 0, 1),
							Entry(nil, 1, 0, 1),
							Entry(nil, 2, 0, 1),
							Entry(nil, -2, 1, -2),
							Entry(nil, -1, 1, -1),
							Entry(nil, 0, 1, 0),
							Entry(nil, 1, 1, 1),
							Entry(nil, 2, 1, 2),
							Entry(nil, -2, 2, 4),
							Entry(nil, -1, 2, 1),
							Entry(nil, 0, 2, 0),
							Entry(nil, 1, 2, 1),
							Entry(nil, 2, 2, 4),
							Entry(nil, 10, 5, 100000),
						)
					},
				)
				Context(
					"when the exponent is negative", func() {
						It(
							"panics", func() {
								Expect(
									func() {
										math.IntPow(2, -2)
									},
								).To(Panic())
							},
						)
					},
				)
			},
		)
	},
)
=======
package math_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/math"
)

var _ = Describe("Math", func() {
	Describe("IntPow", func() {
		DescribeTable("Should correctly compute integer powers",
			func(x, n, expected int) {
				Expect(math.IntPow(x, n)).To(Equal(expected))
			},
			Entry("0^0 = 1", 0, 0, 1),
			Entry("0^1 = 0", 0, 1, 0),
			Entry("1^0 = 1", 1, 0, 1),
			Entry("1^1 = 1", 1, 1, 1),
			Entry("2^0 = 1", 2, 0, 1),
			Entry("2^1 = 2", 2, 1, 2),
			Entry("2^2 = 4", 2, 2, 4),
			Entry("2^3 = 8", 2, 3, 8),
			Entry("3^2 = 9", 3, 2, 9),
			Entry("3^3 = 27", 3, 3, 27),
			Entry("5^2 = 25", 5, 2, 25),
			Entry("-2^2 = 4", -2, 2, 4),
			Entry("-2^3 = -8", -2, 3, -8),
		)

		Context("Edge Cases", func() {
			It("Should handle large powers efficiently", func() {
				Expect(math.IntPow(2, 10)).To(Equal(1024))
			})

			It("Should handle negative bases with even exponents", func() {
				Expect(math.IntPow(-3, 2)).To(Equal(9))
			})

			It("Should handle negative bases with odd exponents", func() {
				Expect(math.IntPow(-3, 3)).To(Equal(-27))
			})
		})
	})

	Describe("MaxUint Constants", func() {
		It("Should define correct MaxUint20", func() {
			// MaxUint20 = 2^20 - 1 = 1048575
			Expect(int(math.MaxUint20)).To(Equal(1048575))
		})

		It("Should define correct MaxUint12", func() {
			// MaxUint12 = 2^12 - 1 = 4095
			Expect(int(math.MaxUint12)).To(Equal(4095))
		})
	})
})
>>>>>>> rc
