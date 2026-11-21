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
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/search"
)

// weightedOracle ranks entries as D-247 states: per word, the best match over the
// primary tokens (100, 10, 1) and the secondary tokens (10, 1, 0); a fuzzy secondary
// match still counts; ties break on the count of primary tokens.
type weightedOracle map[int][2][]string

func (o weightedOracle) set(key int, primary, secondary string) {
	p := oracleTokens(primary)
	var s []string
	for _, tok := range oracleTokens(secondary) {
		if !slices.Contains(p, tok) {
			s = append(s, tok)
		}
	}
	if len(p)+len(s) == 0 {
		delete(o, key)
		return
	}
	o[key] = [2][]string{p, s}
}

func (o weightedOracle) score(key int, words []string) (int, int) {
	var total, matched int
	for _, w := range words {
		best, hit := 0, false
		for _, tok := range o[key][0] {
			if wt := weight(w, tok); wt > 0 {
				best, hit = max(best, wt), true
			}
		}
		for _, tok := range o[key][1] {
			if wt := weight(w, tok); wt > 0 {
				best, hit = max(best, wt/10), true
			}
		}
		if hit {
			total, matched = total+best, matched+1
		}
	}
	return total, matched
}

func (o weightedOracle) expectRanked(keys []int, term string) {
	GinkgoHelper()
	words := oracleWords(term)
	for i := 1; i < len(keys); i++ {
		prev, _ := o.score(keys[i-1], words)
		next, _ := o.score(keys[i], words)
		Expect(cmp.Or(
			cmp.Compare(next, prev),
			cmp.Compare(len(o[keys[i-1]][0]), len(o[keys[i]][0])),
		)).To(
			BeNumerically("<=", 0),
			"term %q ranks %d %v (score %d) before %d %v (score %d)",
			term, keys[i-1], o[keys[i-1]], prev, keys[i], o[keys[i]], next,
		)
	}
}

var _ = Describe("Secondary text breaker", func() {
	var idx *search.Index[int]
	BeforeEach(func() { idx = search.New[int]() })
	find := func(term string) []int {
		return idx.Search(search.Query[int]{Term: term}).Keys
	}

	It("Should rank a name prefix match above a description-only prefix match",
		func() {
			idx.SetText(1, []string{"pt"}, []string{"pressure at the inlet"})
			idx.SetText(2, []string{"pressurizer"}, []string{"pressure loop heater"})
			// 2 matches "press" by a prefix of its name, 1 only by its description.
			Expect(find("press")).To(Equal([]int{2, 1}))
		})

	It("Should score a primary prefix match as a primary prefix when a secondary "+
		"token is also a prefix of the word", func() {
		idx.SetText(1, []string{"temperature"}, nil)
		idx.SetText(2, []string{"tempest"}, []string{"temperature"})
		idx.SetText(3, []string{"tempest"}, nil)
		// All three match "temp" by a primary prefix and hold one primary token.
		Expect(find("temp")).To(Equal([]int{1, 2, 3}))
	})

	It("Should score a primary fuzzy match as a primary fuzzy match when a "+
		"secondary token is also within one edit of the word", func() {
		idx.SetText(1, []string{"tans"}, nil)
		idx.SetText(2, []string{"tanx"}, []string{"tans"})
		idx.SetText(3, []string{"tanx"}, nil)
		// All three match "tank" within one edit of a primary token.
		Expect(find("tank")).To(Equal([]int{1, 2, 3}))
	})

	seeds := make([]TableEntry, 0, 32)
	for seed := range 32 {
		seeds = append(seeds, Entry(fmt.Sprintf("seed %d", seed), int64(seed)))
	}
	DescribeTable("Agreeing with a brute-force weighted search",
		func(seed int64) {
			rng := rand.New(rand.NewPCG(uint64(seed), 0))
			o := weightedOracle{}
			for range 400 {
				key := rng.IntN(40)
				if rng.IntN(4) == 0 {
					idx.Delete(key)
					delete(o, key)
					continue
				}
				primary, secondary := randomText(rng), ""
				if rng.IntN(2) == 0 {
					secondary = randomText(rng)
				}
				idx.SetText(key, []string{primary}, []string{secondary})
				o.set(key, primary, secondary)
				term := randomText(rng)
				o.expectRanked(find(term), term)
			}
		},
		seeds,
	)
})
