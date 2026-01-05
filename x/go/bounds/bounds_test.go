// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bounds_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/bounds"
)

var _ = Describe("Bounds", func() {
	DescribeTable("Contains", func(b bounds.Bounds[int], value int, expected bool) {
		Expect(b.Contains(value)).To(Equal(expected))
	},
		Entry("In Middle", bounds.Bounds[int]{Lower: 5, Upper: 10}, 7, true),
		Entry("At Start", bounds.Bounds[int]{Lower: 5, Upper: 10}, 5, true),
		Entry("At End", bounds.Bounds[int]{Lower: 5, Upper: 10}, 10, false),
		Entry("Both The Same", bounds.Bounds[int]{Lower: 10, Upper: 10}, 10, false),
		Entry("Lower Higher than Upper", bounds.Bounds[int]{Lower: 10, Upper: 5}, 10, false),
	)
	Describe("String", func() {
		It("Should return a formatted string", func() {
			Expect(bounds.Bounds[int]{Lower: 3, Upper: 5}.String()).To(Equal("Bounds[3, 5)"))
		})
	})

	Describe("Span", func() {
		It("Should return the distance between the upper and lower value of the bounds", func() {
			Expect(bounds.Bounds[int]{Lower: 3, Upper: 5}.Span()).To(Equal(2))
		})
	})
})
