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
			merged := search.Merge(0, a.Search("pressure", 0), b.Search("pressure", 0))
			Expect(merged.Keys).To(Equal([]string{"a1", "b1", "a2", "b2"}))
		})

		It(
			"Should keep only the keys that match every word when any result has them",
			func() {
				a := newIndex(map[string]string{"a1": "my blob"})
				b := newIndex(map[string]string{"b1": "blob tree"})
				Expect(
					search.Merge(
						0,
						a.Search("blob tree", 0),
						b.Search("blob tree", 0),
					).Keys,
				).
					To(Equal([]string{"b1"}))
			},
		)

		It(
			"Should keep the keys that match any word when no result matches every word",
			func() {
				a := newIndex(map[string]string{"a1": "my blob"})
				b := newIndex(map[string]string{"b1": "your tree"})
				Expect(
					search.Merge(
						0,
						a.Search("blob tree", 0),
						b.Search("blob tree", 0),
					).Keys,
				).
					To(ConsistOf("a1", "b1"))
			},
		)

		It("Should keep the order of results for equal matches", func() {
			a := newIndex(map[string]string{"a1": "valve"})
			b := newIndex(map[string]string{"b1": "valve"})
			Expect(search.Merge(0, b.Search("valve", 0), a.Search("valve", 0)).Keys).
				To(Equal([]string{"b1", "a1"}))
		})

		It("Should return a result that merges again", func() {
			a := newIndex(map[string]string{"a1": "pressure feed"})
			b := newIndex(map[string]string{"b1": "pressure"})
			c := newIndex(map[string]string{"c1": "pressures"})
			ab := search.Merge(0, a.Search("pressure", 0), b.Search("pressure", 0))
			Expect(search.Merge(0, c.Search("pressure", 0), ab).Keys).
				To(Equal([]string{"b1", "a1", "c1"}))
		})

		It("Should return no keys for no results", func() {
			Expect(search.Merge[string](0).Keys).To(BeEmpty())
		})
	})

	DescribeTable("Should keep the closest matches up to the limit",
		func(limit int, expected []string) {
			a := newIndex(map[string]string{"a1": "pressure", "a2": "pressures"})
			b := newIndex(map[string]string{"b1": "pressure feed", "b2": "pressur"})
			Expect(
				search.Merge(
					limit,
					a.Search("pressure", 0),
					b.Search("pressure", 0),
				).Keys,
			).To(Equal(expected))
		},
		Entry("fewer than every match", 3, []string{"a1", "b1", "a2"}),
		Entry("more than every match", 10, []string{"a1", "b1", "a2", "b2"}),
		Entry("every match", 0, []string{"a1", "b1", "a2", "b2"}),
	)

	It("Should keep the rank of each match a limited search kept", func() {
		a := newIndex(map[string]string{
			"a": "pressure", "b": "pressure feed", "c": "pressures",
		})
		other := newIndex(map[string]string{"d": "pressure inlet feed"})
		Expect(
			search.Merge(0, a.Search("pressure", 2), other.Search("pressure", 0)).Keys,
		).
			To(Equal([]string{"a", "b", "d"}))
	})

	Describe("MapKeys", func() {
		It("Should replace each key and keep the rank of each", func() {
			a := search.New[int]()
			a.Set(1, "pressure feed")
			a.Set(2, "pressure")
			b := newIndex(map[string]string{"1.5": "pressures"})
			mapped := search.MapKeys(a.Search("pressure", 0), strconv.Itoa)
			Expect(mapped.Keys).To(Equal([]string{"2", "1"}))
			Expect(search.Merge(0, b.Search("pressure", 0), mapped).Keys).
				To(Equal([]string{"2", "1", "1.5"}))
		})
	})
})
