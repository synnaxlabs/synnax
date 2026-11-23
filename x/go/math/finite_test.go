// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xmath "github.com/synnaxlabs/x/math"
)

var _ = Describe("Finite", func() {
	DescribeTable("Should report whether a float is finite",
		func(v float64, expected bool) { Expect(xmath.Finite(v)).To(Equal(expected)) },
		Entry("zero", 0.0, true),
		Entry("the largest float", math.MaxFloat64, true),
		Entry("NaN", math.NaN(), false),
		Entry("positive infinity", math.Inf(1), false),
		Entry("negative infinity", math.Inf(-1), false),
	)

	It("Should report an infinite float32 as not finite", func() {
		Expect(xmath.Finite(float32(math.Inf(1)))).To(BeFalse())
	})

	It("Should report every integer as finite", func() {
		Expect(xmath.Finite(int64(math.MinInt64))).To(BeTrue())
	})
})
