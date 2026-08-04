// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/types/versions/v0"
)

var (
	dimNone     = v0.Dimensions{}
	dimLength   = v0.Dimensions{Length: 1}
	dimTime     = v0.Dimensions{Time: 1}
	dimVelocity = v0.Dimensions{Length: 1, Time: -1}
	dimPressure = v0.Dimensions{Mass: 1, Length: -1, Time: -2}
)

var _ = Describe("Dimensions", func() {
	Describe("Equal", func() {
		It("Should return true for identical dimensions", func() {
			Expect(dimVelocity.Equal(v0.Dimensions{Length: 1, Time: -1})).To(BeTrue())
		})

		It("Should return false for differing dimensions", func() {
			Expect(dimLength.Equal(dimTime)).To(BeFalse())
		})
	})

	Describe("IsZero", func() {
		It("Should return true for dimensionless", func() {
			Expect(dimNone.IsZero()).To(BeTrue())
		})

		It("Should return false for dimensioned", func() {
			Expect(dimLength.IsZero()).To(BeFalse())
			Expect(dimPressure.IsZero()).To(BeFalse())
		})
	})

	Describe("String", func() {
		It("Should format velocity", func() {
			s := dimVelocity.String()
			Expect(s).To(ContainSubstring("length^1"))
			Expect(s).To(ContainSubstring("time^-1"))
		})

		It("Should return dimensionless for zero", func() {
			Expect(dimNone.String()).To(Equal("dimensionless"))
		})
	})
})
