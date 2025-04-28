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
