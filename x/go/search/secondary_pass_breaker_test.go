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
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/search"
)

// These cases passed at eaed24fbd3.
var _ = Describe("Secondary text breaker (passing cases)", func() {
	var idx *search.Index[int]
	BeforeEach(func() { idx = search.New[int]() })
	find := func(term string) []int {
		return idx.Search(search.Query[int]{Term: term}).Keys
	}

	It("Should rank an exact name above a description mention and a longer name",
		func() {
			idx.SetText(1, []string{"pressure_2"}, nil)
			idx.SetText(2, []string{"tank"}, []string{"pressure of the tank"})
			idx.SetText(3, []string{"pressure"}, strings.Fields(
				strings.Repeat("pressure inlet valve sensor ", 40),
			))
			Expect(find("pressure")).To(Equal([]int{3, 1, 2}))
		})

	It("Should count a word in both texts once, in any case", func() {
		idx.SetText(1, []string{"Pressure"}, []string{"PRESSURE pressure"})
		idx.SetText(2, []string{"pressure"}, nil)
		Expect(find("pressure")).To(Equal([]int{1, 2}))
	})

	It("Should keep an NFD secondary form of an NFC primary word as its own token",
		func() {
			idx.SetText(1, []string{"caf\\u00e9"}, []string{"cafe\\u0301"})
			idx.SetText(2, []string{"caf\\u00e9"}, nil)
			Expect(find("caf\\u00e9")).To(Equal([]int{1, 2}))
			// No normalization, as on base: the NFD word is two edits from the NFC one.
			Expect(find("cafe\\u0301")).To(Equal([]int{1}))
		})

	It("Should hold no secondary tokens for secondary text of separators", func() {
		idx.SetText(1, []string{"tank"}, []string{" ... --- /// ___ "})
		idx.SetText(2, []string{"tank"}, nil)
		Expect(find("tank")).To(Equal([]int{1, 2}))
	})

	It("Should drop stale secondary tokens when the secondary text changes", func() {
		idx.SetText(1, []string{"tank"}, []string{"alpha beta"})
		idx.SetText(1, []string{"tank"}, []string{"gamma"})
		Expect(find("alpha")).To(BeEmpty())
		Expect(find("beta")).To(BeEmpty())
		Expect(find("gamma")).To(Equal([]int{1}))
		idx.Set(1, "tank")
		Expect(find("gamma")).To(BeEmpty())
	})

	It("Should keep its order between two indexes built the same way", func() {
		build := func() *search.Index[int] {
			x := search.New[int]()
			rng := rand.New(rand.NewPCG(7, 0))
			for range 300 {
				x.SetText(rng.IntN(40), []string{randomText(rng)},
					[]string{randomText(rng)})
			}
			return x
		}
		a, b := build(), build()
		rng := rand.New(rand.NewPCG(8, 0))
		for range 200 {
			term := randomText(rng)
			Expect(a.Search(search.Query[int]{Term: term}).Keys).
				To(Equal(b.Search(search.Query[int]{Term: term}).Keys))
		}
	})

	It("Should search while another goroutine writes secondary text", func() {
		var wg sync.WaitGroup
		wg.Go(func() {
			for i := range 2000 {
				idx.SetText(i%50, []string{fmt.Sprintf("name%d", i)},
					[]string{fmt.Sprintf("desc %d pressure", i)})
				if i%7 == 0 {
					idx.Delete(i % 50)
				}
			}
		})
		for range 2000 {
			idx.Search(search.Query[int]{Term: "pressure name1", Limit: 5})
		}
		wg.Wait()
	})

	seeds := make([]TableEntry, 0, 32)
	for seed := range 32 {
		seeds = append(seeds, Entry(fmt.Sprintf("seed %d", seed), int64(seed)))
	}
	DescribeTable("Agreeing with itself under limits, merges, and exceptions",
		func(seed int64) {
			var (
				rng   = rand.New(rand.NewPCG(uint64(seed), 1))
				split = [2]*search.Index[int]{search.New[int](), search.New[int]()}
				keys  = map[int]bool{}
			)
			for range 400 {
				key := rng.IntN(40)
				if rng.IntN(4) == 0 {
					idx.Delete(key)
					split[key%2].Delete(key)
					delete(keys, key)
					continue
				}
				primary, secondary := []string{randomText(rng)}, []string{}
				if rng.IntN(2) == 0 {
					secondary = append(secondary, randomText(rng))
				}
				idx.SetText(key, primary, secondary)
				split[key%2].SetText(key, primary, secondary)
				keys[key] = true
				term := randomText(rng)
				full := idx.Search(search.Query[int]{Term: term}).Keys
				n := 1 + rng.IntN(4)
				Expect(idx.Search(search.Query[int]{Term: term, Limit: n}).Keys).
					To(Equal(full[:min(n, len(full))]))
				merged := search.Merge(
					0,
					split[0].Search(search.Query[int]{Term: term}),
					split[1].Search(search.Query[int]{Term: term}),
				)
				Expect(merged.Keys).To(ConsistOf(full))
				Expect(search.Merge(
					n,
					split[0].Search(search.Query[int]{Term: term, Limit: n}),
					split[1].Search(search.Query[int]{Term: term, Limit: n}),
				).Keys).To(Equal(merged.Keys[:min(n, len(merged.Keys))]))
				except := rng.IntN(40)
				without := idx.Search(search.Query[int]{
					Term:   term,
					Except: func(k int) bool { return k == except },
				}).Keys
				Expect(without).NotTo(ContainElement(except))
			}
		},
		seeds,
	)
})
