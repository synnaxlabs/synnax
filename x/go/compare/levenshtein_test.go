// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compare_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/compare"
)

var _ = Describe("LevenshteinDistance", func() {
	DescribeTable("distance calculations",
		func(a, b string, expected int) {
			Expect(compare.LevenshteinDistance(a, b)).To(Equal(expected))
		},
		Entry("identical strings", "hello", "hello", 0),
		Entry("empty first string", "", "hello", 5),
		Entry("empty second string", "hello", "", 5),
		Entry("both empty", "", "", 0),
		Entry("single substitution", "cat", "bat", 1),
		Entry("single insertion", "cat", "cats", 1),
		Entry("single deletion", "cats", "cat", 1),
		Entry("multiple edits", "kitten", "sitting", 3),
		Entry("completely different", "abc", "xyz", 3),
		Entry("case sensitive", "Hello", "hello", 1),
		Entry("longer strings", "algorithm", "altruistic", 6),
		Entry("transposition", "ab", "ba", 2),
		Entry("prefix match", "pre", "prefix", 3),
		Entry("suffix match", "fix", "prefix", 3),
		Entry("single character strings", "a", "b", 1),
		Entry("single character same", "a", "a", 0),
		Entry("one empty one char", "", "a", 1),
		Entry("unicode characters byte level", "café", "cafe", 2),
	)

	Describe("symmetry", func() {
		It("should return the same distance regardless of argument order", func() {
			Expect(compare.LevenshteinDistance("foo", "bar")).To(Equal(compare.LevenshteinDistance("bar", "foo")))
			Expect(compare.LevenshteinDistance("kitten", "sitting")).To(Equal(compare.LevenshteinDistance("sitting", "kitten")))
		})
	})

	Describe("triangle inequality", func() {
		It("should satisfy d(a,c) <= d(a,b) + d(b,c)", func() {
			a, b, c := "cat", "bat", "bar"
			dAC := compare.LevenshteinDistance(a, c)
			dAB := compare.LevenshteinDistance(a, b)
			dBC := compare.LevenshteinDistance(b, c)
			Expect(dAC).To(BeNumerically("<=", dAB+dBC))
		})
	})
})
