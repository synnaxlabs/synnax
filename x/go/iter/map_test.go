// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iter_test

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/iter"
)

var _ = Describe("Map", func() {
	It("Should apply the function to each element", func() {
		input := slices.Values([]int{1, 2, 3})
		result := slices.Collect(iter.Map(input, func(v int) int { return v * 2 }))
		Expect(result).To(Equal([]int{2, 4, 6}))
	})

	It("Should handle empty iterators", func() {
		input := slices.Values([]int{})
		result := slices.Collect(iter.Map(input, func(v int) int { return v * 2 }))
		Expect(result).To(BeEmpty())
	})

	It("Should support type transformations", func() {
		input := slices.Values([]int{1, 2, 3})
		result := slices.Collect(iter.Map(input, func(v int) string {
			return string(rune('a' + v - 1))
		}))
		Expect(result).To(Equal([]string{"a", "b", "c"}))
	})

	It("Should stop early when consumer stops", func() {
		callCount := 0
		input := slices.Values([]int{1, 2, 3, 4, 5})
		mapped := iter.Map(input, func(v int) int {
			callCount++
			return v * 2
		})
		for v := range mapped {
			if v == 4 {
				break
			}
		}
		Expect(callCount).To(Equal(2))
	})
})
