// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stringer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/stringer"
)

var _ = Describe("Stringer Slice", func() {
	Describe("TruncateAndFormatSlice", func() {
		It("should return the full slice if its length is less than maxDisplayValues", func() {
			slice := []int{1, 2, 3}
			result := stringer.TruncateAndFormatSlice(slice, 5)
			Expect(result).To(Equal("[1 2 3]"))
		})

		It("should return the full slice if its length is equal to maxDisplayValues", func() {
			slice := []int{1, 2, 3, 4, 5}
			result := stringer.TruncateAndFormatSlice(slice, 5)
			Expect(result).To(Equal("[1 2 3 4 5]"))
		})

		It("should truncate and show first and last elements for long slices (even split)", func() {
			slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			result := stringer.TruncateAndFormatSlice(slice, 6)
			// startCount = 3, endCount = 3
			Expect(result).To(Equal("[1 2 3 ... 8 9 10]"))
		})

		It("should truncate and show first and last elements for long slices (odd split)", func() {
			slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			result := stringer.TruncateAndFormatSlice(slice, 5)
			// startCount = 2, endCount = 3
			Expect(result).To(Equal("[1 2 ... 8 9 10]"))
		})

		It("should handle maxDisplayValues of 1", func() {
			slice := []int{1, 2, 3, 4, 5}
			result := stringer.TruncateAndFormatSlice(slice, 1)
			// startCount = 0, endCount = 1
			Expect(result).To(Equal("[ ... 5]"))
		})

		It("should handle maxDisplayValues of 0", func() {
			slice := []int{1, 2, 3, 4, 5}
			result := stringer.TruncateAndFormatSlice(slice, 0)
			Expect(result).To(Equal("[1 2 3 4 5]"))
		})

		It("should handle empty slices", func() {
			var slice []int
			result := stringer.TruncateAndFormatSlice(slice, 5)
			Expect(result).To(Equal("[]"))
		})

		It("should handle slices of strings", func() {
			slice := []string{"a", "b", "c", "d", "e", "f", "g"}
			result := stringer.TruncateAndFormatSlice(slice, 4)
			// startCount = 2, endCount = 2
			Expect(result).To(Equal("[a b ... f g]"))
		})

		It("should handle maxDisplayValues greater than slice length", func() {
			slice := []int{1, 2, 3}
			result := stringer.TruncateAndFormatSlice(slice, 10)
			Expect(result).To(Equal("[1 2 3]"))
		})
	})
})
