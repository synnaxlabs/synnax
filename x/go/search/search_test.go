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
	"maps"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/search"
)

// oracle ranks entries the way Index.Search must, by brute force over every token.
type oracle map[int][]string

func oracleWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return unicode.IsSpace(r) || r == '_' || r == '-' || r == '.'
	})
}

func oracleTokens(text string) []string {
	var tokens []string
	for _, tok := range oracleWords(text) {
		if !slices.Contains(tokens, tok) {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}

func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := range ra {
		cur := make([]int, len(rb)+1)
		cur[0] = i + 1
		for j := range rb {
			sub := prev[j]
			if ra[i] != rb[j] {
				sub++
			}
			cur[j+1] = min(prev[j+1]+1, cur[j]+1, sub)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func weight(word, token string) int {
	switch {
	case token == word:
		return 100
	case strings.HasPrefix(token, word):
		return 10
	case distance(word, token) <= 1:
		return 1
	}
	return 0
}

// score returns the score of key for words, and how many words it matches.
func (o oracle) score(key int, words []string) (int, int) {
	var total, matched int
	for _, w := range words {
		best := 0
		for _, tok := range o[key] {
			best = max(best, weight(w, tok))
		}
		if best > 0 {
			total, matched = total+best, matched+1
		}
	}
	return total, matched
}

func (o oracle) search(term string) []int {
	words := oracleWords(term)
	var all, any []int
	for key := range o {
		_, matched := o.score(key, words)
		if matched == len(words) && matched > 0 {
			all = append(all, key)
		}
		if matched > 0 {
			any = append(any, key)
		}
	}
	if len(all) > 0 {
		return all
	}
	return any
}

// expectRanked asserts that keys are exactly what the oracle matches for term, in an
// order the oracle accepts.
func (o oracle) expectRanked(keys []int, term string) {
	GinkgoHelper()
	Expect(keys).To(ConsistOf(o.search(term)), "term %q", term)
	words := oracleWords(term)
	for i := 1; i < len(keys); i++ {
		prev, _ := o.score(keys[i-1], words)
		next, _ := o.score(keys[i], words)
		Expect(cmp.Or(
			cmp.Compare(next, prev),
			cmp.Compare(len(o[keys[i-1]]), len(o[keys[i]])),
		)).To(BeNumerically("<=", 0), "term %q ranks %d before %d", term, keys[i-1], keys[i])
	}
}

// randomText joins random words. Its pieces include capitals, a rune whose lowercase
// form is longer ('İ'), and an invalid UTF-8 byte.
func randomText(rng *rand.Rand) string {
	const separators = " _-."
	pieces := []string{"a", "b", "c", "é", "A", "İ", "Σ", "\xff"}
	words := make([]string, 1+rng.IntN(3))
	for i := range words {
		var w strings.Builder
		for range 1 + rng.IntN(6) {
			w.WriteString(pieces[rng.IntN(len(pieces))])
		}
		words[i] = w.String()
	}
	return strings.Join(words, string(separators[rng.IntN(len(separators))]))
}

var _ = Describe("Index", func() {
	var idx *search.Index[int]
	BeforeEach(func() { idx = search.New[int]() })

	DescribeTable("Matching",
		func(text, term string) {
			idx.Set(1, text)
			Expect(idx.Search(term, 0).Keys).To(Equal([]int{1}))
		},
		Entry("an exact word", "test", "test"),
		Entry("a word in a multi-word entry", "October 28 Gooster", "Gooster"),
		Entry("a term with underscores", "gse_ai_15", "ai_15"),
		Entry("one replaced rune in capitals", "BBTPC", "BTTPC"),
		Entry("one replaced rune in lowercase", "BBTPC", "bttpc"),
		Entry("one replaced rune in a multi-word entry", "BBTPC Sim", "BTTPC"),
		Entry("a prefix", "Channel", "ch"),
		Entry("capitals with underscores", "DAQ_PT", "DAQ_PT"),
		Entry("a prefix of words with underscores", "DAQ_PT_1", "DAQ_PT"),
		Entry("one inserted rune", "sensor", "sensors"),
		Entry("one deleted rune", "sensor", "senor"),
		Entry("one replaced multi-byte rune", "résumé", "resumé"),
		Entry("a word split by '-'", "gse-ai", "ai"),
		Entry("a word split by '.'", "gse.ai", "ai"),
		Entry("a word split by a tab", "gse\tai", "ai"),
	)

	DescribeTable("Matching long words by their first 255 runes",
		func(text, term string, expected []int) {
			idx.Set(1, text)
			Expect(idx.Search(term, 0).Keys).To(Equal(expected))
		},
		Entry(
			"a word that agrees in its first 255 runes",
			strings.Repeat("é", 255)+"aa",
			strings.Repeat("é", 255)+"cc",
			[]int{1},
		),
		Entry(
			"not a word that differs within its first 255 runes",
			strings.Repeat("é", 255)+"aa",
			strings.Repeat("é", 200)+strings.Repeat("x", 57),
			[]int{},
		),
	)

	DescribeTable(
		"Reading the first 64 words of a term",
		func(term string, expected []int) {
			idx.Set(1, "alpha")
			Expect(idx.Search(term, 0).Keys).To(Equal(expected))
		},
		Entry(
			"a word within the first 64",
			strings.Repeat("zzz ", 63)+"alpha",
			[]int{1},
		),
		Entry(
			"not a word after the first 64",
			strings.Repeat("zzz ", 64)+"alpha",
			[]int{},
		),
	)

	DescribeTable("Not matching",
		func(text, term string) {
			idx.Set(1, text)
			Expect(idx.Search(term, 0).Keys).To(BeEmpty())
		},
		Entry("a different word", "test", "nope"),
		Entry("a short term two edits away", "Channel", "nn"),
		Entry("an empty term", "test", ""),
		Entry("a term of only separators", "test", " _-."),
	)

	DescribeTable(
		"Ranking",
		func(texts []string, term string, first int) {
			for i, text := range texts {
				idx.Set(i, text)
			}
			Expect(idx.Search(term, 0).Keys).To(HaveExactElements(
				append([]any{first}, Not(Equal(first)))...,
			))
		},
		Entry("an exact word before a prefix", []string{"test2", "test"}, "test", 1),
		Entry(
			"the most exact words first",
			[]string{"October 30 Gooster", "October 29 Gooster", "October 28 Gooster"},
			"October 28 Gooster",
			2,
		),
		Entry("a shared prefix", []string{"View B", "View A"}, "View A", 1),
		Entry(
			"fewer tokens on a tie",
			[]string{"pressure feed", "pressure"},
			"pressure",
			1,
		),
	)

	It("Should rank an exact word, then a prefix, then a near word", func() {
		idx.Set(1, "tost")
		idx.Set(2, "testing")
		idx.Set(3, "test")
		Expect(idx.Search("test", 0).Keys).To(Equal([]int{3, 2, 1}))
	})

	DescribeTable("Should keep the closest matches up to the limit",
		func(limit int, expected []int) {
			idx.Set(1, "pressure")
			idx.Set(2, "pressure feed")
			idx.Set(3, "pressures")
			Expect(idx.Search("pressure", limit).Keys).To(Equal(expected))
		},
		Entry("fewer than every match", 2, []int{1, 2}),
		Entry("more than every match", 10, []int{1, 2, 3}),
		Entry("every match", 0, []int{1, 2, 3}),
	)

	It(
		"Should leave out the keys that except returns true for before the limit",
		func() {
			idx.Set(1, "alpha")
			idx.Set(2, "alpha")
			idx.Set(3, "alpha")
			Expect(
				idx.SearchExcept("alpha", func(k int) bool { return k == 1 }, 1).Keys,
			).
				To(Equal([]int{2}))
		},
	)

	Describe("Several texts", func() {
		BeforeEach(func() { idx.Set(1, "My Device", "LabJack", "T7") })

		It("Should match a word of any text", func() {
			Expect(idx.Search("LabJack", 0).Keys).To(Equal([]int{1}))
		})

		It("Should match words across texts", func() {
			Expect(idx.Search("device t7", 0).Keys).To(Equal([]int{1}))
		})
	})

	Describe("Every word versus any word", func() {
		It("Should return only the entries that match every word", func() {
			idx.Set(1, "Pressure Sensor", "NI")
			idx.Set(2, "Pressure")
			idx.Set(3, "gse_doa_1")
			idx.Set(4, "gse_ai_12")
			Expect(idx.Search("Pressure Sensor", 0).Keys).To(Equal([]int{1}))
			Expect(idx.Search("gse_ai_12", 0).Keys).To(Equal([]int{4}))
		})

		It("Should return the entries that match any word when none match all", func() {
			idx.Set(1, "My Blob")
			idx.Set(2, "Your Tree")
			Expect(idx.Search("blob tree", 0).Keys).To(ConsistOf(1, 2))
		})
	})

	Describe("Set", func() {
		It("Should replace the tokens of an indexed key", func() {
			idx.Set(1, "alpha")
			idx.Set(1, "beta")
			Expect(idx.Search("alpha", 0).Keys).To(BeEmpty())
			Expect(idx.Search("beta", 0).Keys).To(Equal([]int{1}))
		})

		It("Should keep one entry when set twice with the same text", func() {
			idx.Set(1, "alpha beta")
			idx.Set(1, "beta alpha")
			Expect(idx.Search("alpha", 0).Keys).To(Equal([]int{1}))
		})

		It("Should count each repeated word of an entry once", func() {
			idx.Set(1, "valve inlet outlet")
			idx.Set(2, "valve inlet valve valve")
			Expect(idx.Search("valve", 0).Keys).To(Equal([]int{2, 1}))
		})

		It("Should index an entry with no tokens as matching nothing", func() {
			idx.Set(1, "")
			idx.Set(2, "alpha")
			Expect(idx.Search("alpha", 0).Keys).To(Equal([]int{2}))
		})
	})

	Describe("Delete", func() {
		It("Should remove a key from every match", func() {
			idx.Set(1, "alpha")
			idx.Set(2, "alpha")
			idx.Delete(1)
			Expect(idx.Search("alpha", 0).Keys).To(Equal([]int{2}))
		})

		It("Should do nothing for a key that is not indexed", func() {
			idx.Set(1, "alpha")
			idx.Delete(2)
			Expect(idx.Search("alpha", 0).Keys).To(Equal([]int{1}))
		})

		It("Should not match a deleted key after its slot is reused", func() {
			idx.Set(1, "alpha")
			idx.Delete(1)
			idx.Set(2, "beta")
			Expect(idx.Search("alpha", 0).Keys).To(BeEmpty())
			Expect(idx.Search("beta", 0).Keys).To(Equal([]int{2}))
		})
	})

	DescribeTable("Agreeing with a brute-force search",
		func(seed int64) {
			var (
				rng   = rand.New(rand.NewPCG(uint64(seed), 0))
				o     = oracle{}
				split = [2]*search.Index[int]{search.New[int](), search.New[int]()}
			)
			for range 400 {
				key := rng.IntN(40)
				if rng.IntN(4) == 0 {
					idx.Delete(key)
					split[key%2].Delete(key)
					delete(o, key)
					continue
				}
				text := randomText(rng)
				idx.Set(key, text)
				split[key%2].Set(key, text)
				if tokens := oracleTokens(text); len(tokens) > 0 {
					o[key] = tokens
				} else {
					delete(o, key)
				}
				term := randomText(rng)
				full := idx.Search(term, 0).Keys
				o.expectRanked(full, term)
				n := 1 + rng.IntN(4)
				Expect(idx.Search(term, n).Keys).To(Equal(full[:min(n, len(full))]))
				merged := search.Merge(
					0,
					split[0].Search(term, 0),
					split[1].Search(term, 0),
				)
				o.expectRanked(merged.Keys, term)
				Expect(
					search.Merge(
						n,
						split[0].Search(term, n),
						split[1].Search(term, n),
					).Keys,
				).To(Equal(merged.Keys[:min(n, len(merged.Keys))]))
				except := rng.IntN(40)
				without := maps.Clone(o)
				delete(without, except)
				without.expectRanked(
					idx.SearchExcept(
						term,
						func(k int) bool { return k == except },
						0,
					).Keys,
					term,
				)
			}
		},
		Entry("seed 1", int64(1)),
		Entry("seed 2", int64(2)),
		Entry("seed 3", int64(3)),
		Entry("seed 4", int64(4)),
	)

	It("Should agree with a brute-force search when its tokens fill several pages",
		func() {
			o := oracle{}
			set := func(key int, text string) {
				idx.Set(key, text)
				o[key] = oracleTokens(text)
			}
			for i := range 600 {
				set(i, fmt.Sprintf("w%04d", i))
			}
			for i := range 300 {
				idx.Delete(i)
				delete(o, i)
			}
			for i := range 300 {
				set(1000+i, fmt.Sprintf("x%04d", i))
			}
			for _, term := range []string{"w0599", "w0001", "x0150", "w05", "x"} {
				o.expectRanked(idx.Search(term, 0).Keys, term)
			}
		},
	)

	Describe("SearchExcept", func() {
		It("Should leave out the keys that except returns true for", func() {
			idx.Set(1, "alpha")
			idx.Set(2, "alpha")
			Expect(
				idx.SearchExcept("alpha", func(k int) bool { return k == 1 }, 0).Keys,
			).
				To(Equal([]int{2}))
		})

		It("Should match any word when only left-out keys match every word", func() {
			idx.Set(1, "alpha beta")
			idx.Set(2, "alpha")
			Expect(idx.Search("alpha beta", 0).Keys).To(Equal([]int{1}))
			Expect(
				idx.SearchExcept(
					"alpha beta",
					func(k int) bool { return k == 1 },
					0,
				).Keys,
			).
				To(Equal([]int{2}))
		})
	})

	It("Should allow searches while keys are set and deleted", func() {
		var wg sync.WaitGroup
		for w := range 4 {
			wg.Go(func() {
				for i := range 500 {
					key := w*1000 + i%50
					idx.Set(key, "sensor_"+strconv.Itoa(i))
					if i%3 == 0 {
						idx.Delete(key)
					}
				}
			})
			wg.Go(func() {
				for range 500 {
					idx.Search("sensor 1", 0)
				}
			})
		}
		wg.Wait()
		idx.Set(1, "sensor_final")
		Expect(idx.Search("final", 0).Keys).To(Equal([]int{1}))
	})
})
