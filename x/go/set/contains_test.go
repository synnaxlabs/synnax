// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package set_test

import (
	"fmt"
	"slices"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Contains", func() {
	DescribeTable("Should report membership",
		func(n int) {
			vals := make([]int, n)
			for i := range vals {
				vals[i] = i * 2
			}
			contains := set.Contains(vals...)
			for _, v := range vals {
				Expect(contains(v)).To(BeTrue())
			}
			Expect(contains(1)).To(BeFalse())
			Expect(contains(-2)).To(BeFalse())
		},
		Entry("With no values", 0),
		Entry("At the linear scan limit", 8),
		Entry("Above the linear scan limit", 9),
		Entry("Well above the linear scan limit", 100),
	)
})

func benchLookups[T comparable](b *testing.B, vals, probes []T) {
	b.Run("linear", func(b *testing.B) {
		for b.Loop() {
			for _, p := range probes {
				_ = slices.Contains(vals, p)
			}
		}
	})
	b.Run("set", func(b *testing.B) {
		s := set.New(vals...)
		for b.Loop() {
			for _, p := range probes {
				_ = s.Contains(p)
			}
		}
	})
}

func BenchmarkContainsStrings(b *testing.B) {
	for _, n := range []int{1, 2, 4, 6, 8, 12, 16, 24, 32} {
		vals := make([]string, n)
		for i := range vals {
			vals[i] = fmt.Sprintf("channel_name_%03d", i)
		}
		probes := make([]string, 64)
		for i := range probes {
			probes[i] = fmt.Sprintf("channel_name_%03d", i%(2*n))
		}
		b.Run(fmt.Sprint(n), func(b *testing.B) { benchLookups(b, vals, probes) })
	}
}

func BenchmarkContainsUint32(b *testing.B) {
	for _, n := range []int{1, 2, 4, 8, 16, 24, 32, 48, 64, 96, 128} {
		vals := make([]uint32, n)
		for i := range vals {
			vals[i] = uint32(i * 7)
		}
		probes := make([]uint32, 64)
		for i := range probes {
			probes[i] = uint32((i % (2 * n)) * 7)
		}
		b.Run(fmt.Sprint(n), func(b *testing.B) { benchLookups(b, vals, probes) })
	}
}
