// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/search"
)

func newIndex(texts map[string]string) *search.Index[string] {
	idx := search.New[string]()
	for key, text := range texts {
		idx.Set(key, text)
	}
	return idx
}

var _ = Describe("Result", func() {
	Describe("Merge", func() {
		It("Should rank the keys of every result together", func() {
			a := newIndex(map[string]string{"a1": "pressure", "a2": "pressures"})
			b := newIndex(map[string]string{"b1": "pressure feed", "b2": "pressur"})
			merged := search.Merge(a.Search("pressure"), b.Search("pressure"))
			Expect(merged.Keys).To(Equal([]string{"a1", "b1", "a2", "b2"}))
		})

		It(
			"Should keep only the keys that match every word when any result has them",
			func() {
				a := newIndex(map[string]string{"a1": "my blob"})
				b := newIndex(map[string]string{"b1": "blob tree"})
				Expect(search.Merge(a.Search("blob tree"), b.Search("blob tree")).Keys).
					To(Equal([]string{"b1"}))
			},
		)

		It(
			"Should keep the keys that match any word when no result matches every word",
			func() {
				a := newIndex(map[string]string{"a1": "my blob"})
				b := newIndex(map[string]string{"b1": "your tree"})
				Expect(search.Merge(a.Search("blob tree"), b.Search("blob tree")).Keys).
					To(ConsistOf("a1", "b1"))
			},
		)

		It("Should keep the order of results for equal matches", func() {
			a := newIndex(map[string]string{"a1": "valve"})
			b := newIndex(map[string]string{"b1": "valve"})
			Expect(search.Merge(b.Search("valve"), a.Search("valve")).Keys).
				To(Equal([]string{"b1", "a1"}))
		})

		It("Should return a result that merges again", func() {
			a := newIndex(map[string]string{"a1": "pressure feed"})
			b := newIndex(map[string]string{"b1": "pressure"})
			c := newIndex(map[string]string{"c1": "pressures"})
			ab := search.Merge(a.Search("pressure"), b.Search("pressure"))
			Expect(search.Merge(c.Search("pressure"), ab).Keys).
				To(Equal([]string{"b1", "a1", "c1"}))
		})

		It("Should return no keys for no results", func() {
			Expect(search.Merge[string]().Keys).To(BeEmpty())
		})
	})

	Describe("First", func() {
		var res search.Result[string]
		BeforeEach(func() {
			res = newIndex(map[string]string{
				"a": "pressure", "b": "pressure feed", "c": "pressures",
			}).Search("pressure")
		})

		DescribeTable("Should keep the closest matches",
			func(n int, expected []string) {
				Expect(res.First(n).Keys).To(Equal(expected))
			},
			Entry("fewer than every match", 2, []string{"a", "b"}),
			Entry("every match", 3, []string{"a", "b", "c"}),
			Entry("more than every match", 10, []string{"a", "b", "c"}),
			Entry("none", 0, []string{}),
		)

		It("Should keep the rank of each kept match", func() {
			other := newIndex(map[string]string{"d": "pressure inlet feed"}).
				Search("pressure")
			Expect(search.Merge(res.First(2), other).Keys).
				To(Equal([]string{"a", "b", "d"}))
		})
	})

	Describe("MapKeys", func() {
		It("Should replace each key and keep the rank of each", func() {
			a := search.New[int]()
			a.Set(1, "pressure feed")
			a.Set(2, "pressure")
			b := newIndex(map[string]string{"1.5": "pressures"})
			mapped := search.MapKeys(a.Search("pressure"), strconv.Itoa)
			Expect(mapped.Keys).To(Equal([]string{"2", "1"}))
			Expect(search.Merge(b.Search("pressure"), mapped).Keys).
				To(Equal([]string{"2", "1", "1.5"}))
		})
	})
})
