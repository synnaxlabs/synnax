// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iter_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/iter"
)

var _ = Describe("Iter", func() {
	Describe("Endlessly", func() {
		It("Should allow the caller to iterate over a collection of values indefinitely", func() {
			iterations := 10
			n := 5
			values := make([]int, n)
			i := iter.Endlessly(values)
			ctx := context.Background()
			for range iterations {
				for k := range n {
					v, ok := i.Next(ctx)
					Expect(ok).To(BeTrue())
					Expect(v).To(Equal(values[k]))
				}
			}
		})
	})
	Describe("All", func() {
		It("Should allow the caller to iterate over a collection of values once", func() {
			n := 5
			values := make([]int, n)
			i := iter.All(values)
			ctx := context.Background()
			for j := range n {
				v, ok := i.Next(ctx)
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal(values[j]))
			}
			v, ok := i.Next(ctx)
			Expect(ok).To(BeFalse())
			Expect(v).To(Equal(0))
		})
	})
	Describe("MapToSliceWithFilter", func() {
		It("Should map and filter values from an iterator", func() {
			values := []int{1, 2, 3, 4, 5, 6}
			i := iter.All(values)
			// Filter for even numbers and double them
			result := iter.MapToSliceWithFilter(ctx, i, func(v int) (int, bool) {
				if v%2 == 0 {
					return v * 2, true
				}
				return 0, false
			})
			Expect(result).To(Equal([]int{4, 8, 12}))
		})
		It("Should return an empty slice when all values are filtered out", func() {
			values := []int{1, 3, 5, 7, 9}
			i := iter.All(values)
			// Filter for even numbers (none exist)
			result := iter.MapToSliceWithFilter(ctx, i, func(v int) (int, bool) {
				if v%2 == 0 {
					return v, true
				}
				return 0, false
			})
			Expect(result).To(BeEmpty())
		})
		It("Should handle empty input", func() {
			values := []int{}
			i := iter.All(values)
			result := iter.MapToSliceWithFilter(ctx, i, func(v int) (int, bool) {
				return v * 2, true
			})
			Expect(result).To(BeEmpty())
		})
		It("Should apply transformation to all values when filter always returns true", func() {
			values := []int{1, 2, 3}
			i := iter.All(values)
			// Always accept values and square them
			result := iter.MapToSliceWithFilter(ctx, i, func(v int) (int, bool) {
				return v * v, true
			})
			Expect(result).To(Equal([]int{1, 4, 9}))
		})
		It("Should work with different types", func() {
			values := []string{"hello", "world", "test", "go"}
			i := iter.All(values)
			// Filter strings longer than 3 chars and get their lengths
			result := iter.MapToSliceWithFilter(ctx, i, func(v string) (int, bool) {
				if len(v) > 3 {
					return len(v), true
				}
				return 0, false
			})
			Expect(result).To(Equal([]int{5, 5, 4}))
		})
	})
})
