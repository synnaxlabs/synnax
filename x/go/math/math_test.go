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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/types"
)

func testIntPow[T types.Numeric]() {
	DescribeTable("Should correctly compute integer powers",
		func(x int, n int, expected int) {
			Expect(math.IntPow(T(x), n)).To(BeEquivalentTo(expected))
		},
		Entry("0^0 = 1", 0, 0, 1),
		Entry("0^1 = 0", 0, 1, 0),
		Entry("0^2 = 0", 0, 2, 0),
		Entry("1^-2 = 1", 1, -2, 1),
		Entry("1^-1 = 1", 1, -1, 1),
		Entry("1^0 = 1", 1, 0, 1),
		Entry("1^1 = 1", 1, 1, 1),
		Entry("1^2 = 1", 1, 2, 1),
		Entry("2^0 = 1", 2, 0, 1),
		Entry("2^1 = 2", 2, 1, 2),
		Entry("2^2 = 4", 2, 2, 4),
	)
	Describe("Should panic for zero to a negative power", func() {
		Expect(func() { math.IntPow(T(0), -1) }).To(Panic())
	})
}

func testIntPowFloating[T types.Floating]() {
	DescribeTable("Should correctly compute integer powers",
		func(x float64, n int, expected float64) {
			Expect(math.IntPow(T(x), n)).To(BeEquivalentTo(expected))
		},
		Entry("-0.5^-2 = 4", -0.5, -2, 4.0),
		Entry("-0.5^-1 = 2", -0.5, -1, -2.0),
		Entry("-0.5^0 = 1", -0.5, 0, 1.0),
		Entry("-0.5^1 = -0.5", -0.5, 1, -0.5),
		Entry("-0.5^2 = 0.25", -0.5, 2, 0.25),
		Entry("0.5^-2 = 4", 0.5, -2, 4.0),
		Entry("0.5^-1 = 2", 0.5, -1, 2.0),
		Entry("0.5^0 = 1", 0.5, 0, 1.0),
		Entry("0.5^1 = 0.5", 0.5, 1, 0.5),
		Entry("0.5^2 = 0.25", 0.5, 2, 0.25),
		Entry("-2^-2 = 0.25", -2.0, -2, 0.25),
		Entry("-2^-1 = -0.5", -2.0, -1, -0.5),
		Entry("2^-2 = 0.25", 2.0, -2, 0.25),
		Entry("2^-1 = 0.5", 2.0, -1, 0.5),
	)
}

func testIntPowSignedInt[T types.SignedInteger]() {
	DescribeTable("Should correctly compute integer powers",
		func(x int, n int, expected int) {
			Expect(math.IntPow(T(x), n)).WithOffset(1).To(BeEquivalentTo(expected))
		},
		Entry("For integers, -2^-2 = 0", -2, -2, 0),
		Entry("For integers, -2^-1 = 0", -2, -1, 0),
		Entry("For integers, 2^-2 = 0", 2, -2, 0),
		Entry("For integers, 2^-1 = 0", 2, -1, 0),
	)
}

func testIntPowInt[T types.Integer]() {
	DescribeTable("Should correctly compute integer powers",
		func(x int, n int, expected int) {
			Expect(math.IntPow(T(x), n)).To(BeEquivalentTo(expected))
		},
		Entry("For integers, 2^-2 = 0", 2, -2, 0),
		Entry("For integers, 2^-1 = 0", 2, -1, 0),
	)
}

func testIntPowSigned[T types.Signed]() {
	DescribeTable("Should correctly compute integer powers",
		func(x int, n int, expected int) {
			Expect(math.IntPow(T(x), n)).To(BeEquivalentTo(expected))
		},
		Entry("-2^0 = 1", -2, 0, 1),
		Entry("-2^1 = -2", -2, 1, -2),
		Entry("-2^2 = 4", -2, 2, 4),
		Entry("-1^-2 = 1", -1, -2, 1),
		Entry("-1^-1 = -1", -1, -1, -1),
		Entry("-1^0 = 1", -1, 0, 1),
		Entry("-1^1 = -1", -1, 1, -1),
		Entry("-1^2 = 1", -1, 2, 1),
	)
}

func runSignedIntIntPowTest[T types.SignedInteger]() {
	testIntPow[T]()
	testIntPowInt[T]()
	testIntPowSignedInt[T]()
	testIntPowSigned[T]()
}

func runUnsignedIntIntPowTest[T types.UnsignedInteger]() {
	testIntPow[T]()
	testIntPowInt[T]()
}

func runFloatingIntPowTest[T types.Floating]() {
	testIntPow[T]()
	testIntPowFloating[T]()
}

var _ = Describe("Math", func() {
	Describe("IntPow", func() {
		runSignedIntIntPowTest[int]()
		runSignedIntIntPowTest[int8]()
		runSignedIntIntPowTest[int16]()
		runSignedIntIntPowTest[int32]()
		runSignedIntIntPowTest[int64]()
		runUnsignedIntIntPowTest[uint]()
		runUnsignedIntIntPowTest[uint8]()
		runUnsignedIntIntPowTest[uint16]()
		runUnsignedIntIntPowTest[uint32]()
		runUnsignedIntIntPowTest[uint64]()
		runFloatingIntPowTest[float32]()
		runFloatingIntPowTest[float64]()
	})

	Describe("MaxUint Constants", func() {
		It("Should define correct MaxUint20", func() {
			// MaxUint20 = 2^20 - 1 = 1048575
			Expect(math.MaxUint20).To(BeEquivalentTo(1048575))
		})

		It("Should define correct MaxUint12", func() {
			// MaxUint12 = 2^12 - 1 = 4095
			Expect(math.MaxUint12).To(BeEquivalentTo(4095))
		})
	})
})
