// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"bytes"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

var errFilter = errors.New("filter error")

func idLT(v int32) gorp.Filter[int32, entry] {
	return gorp.Match(func(_ gorp.Context, e *entry) (bool, error) {
		return e.ID < v, nil
	})
}

func idGT(v int32) gorp.Filter[int32, entry] {
	return gorp.Match(func(_ gorp.Context, e *entry) (bool, error) {
		return e.ID > v, nil
	})
}

func idEQ(v int32) gorp.Filter[int32, entry] {
	return gorp.Match(func(_ gorp.Context, e *entry) (bool, error) {
		return e.ID == v, nil
	})
}

func dataEQ(v string) gorp.Filter[int32, entry] {
	return gorp.Match(func(_ gorp.Context, e *entry) (bool, error) {
		return e.Data == v, nil
	})
}

func errFilterFn() gorp.Filter[int32, entry] {
	return gorp.Match(func(_ gorp.Context, _ *entry) (bool, error) {
		return false, errFilter
	})
}

// bindCtx is a stand-in for a service-defined Retrieve type used to exercise
// the BoundFilter machinery: bound filters close over the threshold and
// compare each entry's ID against it.
type bindCtx struct{ threshold int32 }

func boundIDLT(v int32) gorp.BoundFilter[bindCtx, int32, entry] {
	return gorp.MatchBound(func(_ gorp.Context, b bindCtx, e *entry) (bool, error) {
		return e.ID < (b.threshold + v), nil
	})
}

func boundIDGT(v int32) gorp.BoundFilter[bindCtx, int32, entry] {
	return gorp.MatchBound(func(_ gorp.Context, b bindCtx, e *entry) (bool, error) {
		return e.ID > (b.threshold + v), nil
	})
}

func boundIDEQ(v int32) gorp.BoundFilter[bindCtx, int32, entry] {
	return gorp.MatchBound(func(_ gorp.Context, b bindCtx, e *entry) (bool, error) {
		return e.ID == (b.threshold + v), nil
	})
}

var _ = Describe("Filter Combinators", func() {
	var (
		entries []entry
		tx      gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		tx = db.OpenTx()
		entries = make([]entry, 10)
		for i := range 10 {
			entries[i] = entry{ID: int32(i), Data: "data"}
		}
		Expect(gorp.NewCreate[int32, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("MatchBound", func() {
		It("Should thread the bind value into the closure", func(ctx SpecContext) {
			f := gorp.MatchBound(func(_ gorp.Context, b bindCtx, e *entry) (bool, error) {
				return e.ID == b.threshold, nil
			})
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(f(bindCtx{threshold: 4})).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[4]}))
		})

		It("Should re-evaluate against a different bind on each call", func(ctx SpecContext) {
			f := gorp.MatchBound(func(_ gorp.Context, b bindCtx, e *entry) (bool, error) {
				return e.ID < b.threshold, nil
			})
			var lo, hi []entry
			Expect(gorp.NewRetrieve[int32, entry]().Entries(&lo).
				Where(f(bindCtx{threshold: 3})).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewRetrieve[int32, entry]().Entries(&hi).
				Where(f(bindCtx{threshold: 8})).Exec(ctx, tx)).To(Succeed())
			Expect(lo).To(HaveLen(3))
			Expect(hi).To(HaveLen(8))
		})

		It("Should propagate errors from the closure", func(ctx SpecContext) {
			f := gorp.MatchBound(func(_ gorp.Context, _ bindCtx, _ *entry) (bool, error) {
				return false, errFilter
			})
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().Entries(&res).
				Where(f(bindCtx{})).Exec(ctx, tx)).To(MatchError(errFilter))
		})
	})

	Describe("And", func() {
		It("Should match entries satisfying all filters", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.And(idGT(2), idLT(6))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[3], entries[4], entries[5]}))
		})

		It("Should return no results when filters are mutually exclusive", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.And(idGT(5), idLT(3))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(BeEmpty())
		})

		It("Should pass through with a single filter", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.And(idEQ(3))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[3]}))
		})

		It("Should match all entries with no filters", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.And[int32, entry]()).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(10))
		})

		It("Should short-circuit on error", func(ctx SpecContext) {
			var res []entry
			callCount := 0
			second := gorp.Match(func(_ gorp.Context, _ *entry) (bool, error) {
				callCount++
				return true, nil
			})
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.And(errFilterFn(), second)).
				Exec(ctx, tx)).To(MatchError(errFilter))
			Expect(callCount).To(Equal(0))
		})

		It("Should compose three or more filters", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.And(idGT(1), idLT(8), dataEQ("data"))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(6))
		})

		It("AndBound should bind every child to the same Retrieve", func(ctx SpecContext) {
			combined := gorp.AndBound(boundIDGT(0), boundIDLT(5))
			// threshold=2 → idGT(2) AND idLT(7)
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().Entries(&res).
				Where(combined(bindCtx{threshold: 2})).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[3], entries[4], entries[5], entries[6]}))
		})

		It("AndBound should re-thread the bind on each evaluation", func(ctx SpecContext) {
			combined := gorp.AndBound(boundIDGT(0), boundIDLT(3))
			// threshold=1 → idGT(1) AND idLT(4)
			var first []entry
			Expect(gorp.NewRetrieve[int32, entry]().Entries(&first).
				Where(combined(bindCtx{threshold: 1})).
				Exec(ctx, tx)).To(Succeed())
			Expect(first).To(Equal([]entry{entries[2], entries[3]}))
			// threshold=5 → idGT(5) AND idLT(8)
			var second []entry
			Expect(gorp.NewRetrieve[int32, entry]().Entries(&second).
				Where(combined(bindCtx{threshold: 5})).
				Exec(ctx, tx)).To(Succeed())
			Expect(second).To(Equal([]entry{entries[6], entries[7]}))
		})
	})

	Describe("Or", func() {
		It("Should match entries satisfying any filter", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(idEQ(1), idEQ(5), idEQ(9))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[5], entries[9]}))
		})

		It("Should return no results when no branch matches", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(idEQ(100), idEQ(200))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(BeEmpty())
		})

		It("Should pass through with a single filter", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(idEQ(7))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[7]}))
		})

		It("Should match no entries with no filters", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or[int32, entry]()).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(BeEmpty())
		})

		It("Should short-circuit on first match", func(ctx SpecContext) {
			callCount := 0
			second := gorp.Match(func(_ gorp.Context, _ *entry) (bool, error) {
				callCount++
				return true, nil
			})
			alwaysTrue := gorp.Match(func(_ gorp.Context, _ *entry) (bool, error) {
				return true, nil
			})
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(alwaysTrue, second)).
				Exec(ctx, tx)).To(Succeed())
			Expect(callCount).To(Equal(0))
		})

		It("Should short-circuit on error", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(errFilterFn(), idEQ(1))).
				Exec(ctx, tx)).To(MatchError(errFilter))
		})

		It("OrBound should bind every child to the same Retrieve", func(ctx SpecContext) {
			combined := gorp.OrBound(boundIDEQ(0), boundIDEQ(2), boundIDEQ(4))
			// threshold=1 → idEQ(1) OR idEQ(3) OR idEQ(5)
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().Entries(&res).
				Where(combined(bindCtx{threshold: 1})).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[3], entries[5]}))
		})
	})

	Describe("Not", func() {
		It("Should invert a filter", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Not(idGT(5))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(6))
			for _, e := range res {
				Expect(e.ID).To(BeNumerically("<=", 5))
			}
		})

		It("Should propagate errors", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Not(errFilterFn())).
				Exec(ctx, tx)).To(MatchError(errFilter))
		})

		It("Should double-negate to the original", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Not(gorp.Not(idLT(3)))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[0], entries[1], entries[2]}))
		})

		It("NotBound should invert a bound child after binding", func(ctx SpecContext) {
			inverted := gorp.NotBound(boundIDLT(3))
			// threshold=2 → NOT idLT(5) → idGE(5)
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().Entries(&res).
				Where(inverted(bindCtx{threshold: 2})).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[5], entries[6], entries[7], entries[8], entries[9]}))
		})
	})

	Describe("MatchRaw inside Or/Not", func() {
		alwaysTrueRaw := func(_, _ []byte) (bool, error) { return true, nil }
		alwaysFalseRaw := func(_, _ []byte) (bool, error) { return false, nil }
		containsDataRaw := func(_, value []byte) (bool, error) {
			return bytes.Contains(value, []byte("data")), nil
		}
		neverMatch := gorp.Match(func(_ gorp.Context, _ *entry) (bool, error) {
			return false, nil
		})

		It("Or(MatchRaw(true), neverMatch) should match every entry", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(gorp.MatchRaw[int32, entry](alwaysTrueRaw), neverMatch)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(10))
		})

		It("Or(MatchRaw(containsData), neverMatch) should match every entry", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(gorp.MatchRaw[int32, entry](containsDataRaw), neverMatch)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(10))
		})

		It("Not(MatchRaw(false)) should match every entry", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Not(gorp.MatchRaw[int32, entry](alwaysFalseRaw))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(10))
		})

		It("Not(MatchRaw(true)) should match no entries", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Not(gorp.MatchRaw[int32, entry](alwaysTrueRaw))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(BeEmpty())
		})

		It("Or(MatchRaw, Match) should compose correctly under WhereKeys", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				WhereKeys(1, 2, 3).
				Where(gorp.Or(gorp.MatchRaw[int32, entry](containsDataRaw), neverMatch)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(3))
		})

		It("Or of raw-only children should compose Raw to pre-screen", func() {
			or := gorp.Or(
				gorp.MatchRaw[int32, entry](alwaysFalseRaw),
				gorp.MatchRaw[int32, entry](alwaysTrueRaw),
			)
			Expect(or.Raw).NotTo(BeNil())
			Expect(or.Eval).To(BeNil())
		})

		It("Or with a non-raw child should not compose Raw", func() {
			or := gorp.Or(
				gorp.MatchRaw[int32, entry](alwaysTrueRaw),
				neverMatch,
			)
			Expect(or.Raw).To(BeNil())
			Expect(or.Eval).NotTo(BeNil())
		})

		It("Not of a raw-only child should compose Raw to pre-screen", func() {
			not := gorp.Not(gorp.MatchRaw[int32, entry](alwaysFalseRaw))
			Expect(not.Raw).NotTo(BeNil())
			Expect(not.Eval).To(BeNil())
		})

		It("Not of a non-raw child should not compose Raw", func() {
			not := gorp.Not(neverMatch)
			Expect(not.Raw).To(BeNil())
			Expect(not.Eval).NotTo(BeNil())
		})
	})

	Describe("Nested Composition", func() {
		It("Should support Or(And, And)", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(
					gorp.And(idGT(0), idLT(3)),
					gorp.And(idGT(6), idLT(9)),
				)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[2], entries[7], entries[8]}))
		})

		It("Should support And(Or, Not)", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.And(
					gorp.Or(idEQ(1), idEQ(2), idEQ(3), idEQ(4)),
					gorp.Not(idEQ(2)),
				)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[3], entries[4]}))
		})

		It("Should support Or(Not, Not)", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(
					gorp.Not(idGT(0)),
					gorp.Not(idLT(9)),
				)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[0], entries[9]}))
		})

		It("Should support three levels of nesting", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(
					gorp.And(idGT(0), gorp.Not(idGT(2))),
					gorp.And(idLT(9), gorp.Not(idLT(7))),
				)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[2], entries[7], entries[8]}))
		})
	})

	Describe("Combinators with WhereKeys", func() {
		It("Should apply filters after key lookup", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				WhereKeys(1, 2, 3, 4, 5).
				Where(idGT(3)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[4], entries[5]}))
		})

		It("Should apply Or filter after key lookup", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				WhereKeys(1, 2, 3, 4, 5).
				Where(gorp.Or(idEQ(1), idEQ(5))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[5]}))
		})

		It("Should apply Not filter after key lookup", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				WhereKeys(1, 2, 3, 4, 5).
				Where(gorp.Not(idEQ(3))).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[2], entries[4], entries[5]}))
		})
	})

	Describe("Combinators with Count", func() {
		It("Should count with And filter", func(ctx SpecContext) {
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.And(idGT(2), idLT(7))).
				Count(ctx, tx)).To(Equal(4))
		})

		It("Should count with Or filter", func(ctx SpecContext) {
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.Or(idEQ(0), idEQ(9))).
				Count(ctx, tx)).To(Equal(2))
		})

		It("Should count with nested filter", func(ctx SpecContext) {
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.And(
					gorp.Or(idEQ(1), idEQ(2), idEQ(3)),
					gorp.Not(idEQ(2)),
				)).
				Count(ctx, tx)).To(Equal(2))
		})
	})

	Describe("Combinators with Exists", func() {
		It("Should return true when Or filter has a match", func(ctx SpecContext) {
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.Or(idEQ(1), idEQ(999))).
				Exists(ctx, tx)).To(BeTrue())
		})

		It("Should return false when And filter excludes all", func(ctx SpecContext) {
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.And(idGT(100), idLT(200))).
				Exists(ctx, tx)).To(BeFalse())
		})
	})

	Describe("Combinators with Delete", func() {
		It("Should delete entries matching an Or filter", func(ctx SpecContext) {
			Expect(gorp.NewDelete[int32, entry]().
				Where(gorp.Or(idEQ(0), idEQ(9))).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(0).Exists(ctx, tx)).To(BeFalse())
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(9).Exists(ctx, tx)).To(BeFalse())
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(5).Exists(ctx, tx)).To(BeTrue())
		})

		It("Should delete entries matching a Not filter", func(ctx SpecContext) {
			Expect(gorp.NewDelete[int32, entry]().
				Where(gorp.Not(idLT(8))).
				Exec(ctx, tx)).To(Succeed())
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(8))
		})
	})

	Describe("Raw Filters and Combinators", func() {
		// matchRawDataContains returns a raw filter that matches when the encoded
		// bytes contain the provided byte sequence.
		matchRawDataContains := func(sub []byte) gorp.Filter[int32, entry] {
			return gorp.MatchRaw[int32, entry](func(_, data []byte) (bool, error) {
				return bytes.Contains(data, sub), nil
			})
		}
		// matchRawAlways returns a raw filter that always matches and records how
		// many times it was invoked.
		matchRawAlways := func(calls *int) gorp.Filter[int32, entry] {
			return gorp.MatchRaw[int32, entry](func(_, _ []byte) (bool, error) {
				*calls++
				return true, nil
			})
		}
		// matchRawErr returns a raw filter that always errors.
		matchRawErr := func() gorp.Filter[int32, entry] {
			return gorp.MatchRaw[int32, entry](func(_, _ []byte) (bool, error) {
				return false, fmt.Errorf("raw filter error")
			})
		}
		// matchEvalAndRaw returns a filter with both Eval and Raw populated, to
		// verify And composes the two stages independently.
		matchEvalAndRaw := func(idBelow int32, sub []byte) gorp.Filter[int32, entry] {
			return gorp.Filter[int32, entry]{
				Eval: func(_ gorp.Context, e *entry, _, _ []byte) (bool, error) {
					return e.ID < idBelow, nil
				},
				Raw: func(_, data []byte) (bool, error) {
					return bytes.Contains(data, sub), nil
				},
			}
		}

		Describe("MatchRaw", func() {
			It("Should accept a pure raw filter through Where", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(matchRawDataContains([]byte("data"))).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(10))
			})
			It("Should reject all entries when the raw predicate returns false", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(matchRawDataContains([]byte("nonexistent"))).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(BeEmpty())
			})
			It("Should propagate raw filter errors", func(ctx SpecContext) {
				var res []entry
				err := gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(matchRawErr()).
					Exec(ctx, tx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("raw filter error"))
			})
		})

		Describe("And with raw filters", func() {
			It("Should compose raw stages across multiple raw filters", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(gorp.And(
						matchRawDataContains([]byte("data")),
						matchRawDataContains([]byte("ata")),
					)).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(10))
			})
			It("Should short-circuit the raw stage on first false", func(ctx SpecContext) {
				calls := 0
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(gorp.And(
						matchRawDataContains([]byte("nonexistent")),
						matchRawAlways(&calls),
					)).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(BeEmpty())
				Expect(calls).To(Equal(0))
			})
			It("Should propagate raw stage errors", func(ctx SpecContext) {
				var res []entry
				err := gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(gorp.And(
						matchRawErr(),
						matchRawDataContains([]byte("data")),
					)).
					Exec(ctx, tx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("raw filter error"))
			})
			It("Should compose Eval and Raw stages independently for a filter carrying both", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(gorp.And(
						matchEvalAndRaw(5, []byte("data")),
						idGT(1),
					)).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(Equal([]entry{entries[2], entries[3], entries[4]}))
			})
			It("Should reject every entry when raw stage rejects and eval would pass", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(gorp.And(
						matchRawDataContains([]byte("nonexistent")),
						idGT(0),
					)).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(BeEmpty())
			})
		})

		Describe("Or with raw filters", func() {
			// Or composes raw children correctly: under mixed-kind composition
			// the raw is dispatched against the entry's raw bytes at decode
			// time; under all-raw composition the raw OR survives as a
			// pre-decode pre-screen so no decode is needed.
			It("Should match entries that pass the raw child OR the eval child", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(gorp.Or(
						matchRawDataContains([]byte("data")),
						idEQ(3),
					)).
					Exec(ctx, tx)).To(Succeed())
				// Every entry has Data == "data" so the raw branch matches
				// all 10; the eval branch only adds entry 3, which is already
				// covered.
				Expect(res).To(HaveLen(10))
			})
			It("Should match every entry when every child is a matching pure-raw", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(gorp.Or(
						matchRawDataContains([]byte("data")),
						matchRawDataContains([]byte("ata")),
					)).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(10))
			})
		})

		Describe("Not with raw filters", func() {
			// Not over a pure-raw child composes an inverted Raw so the
			// pre-decode pre-screen path survives Not.
			It("Should reject entries that pass the raw child", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Where(gorp.Not(matchRawDataContains([]byte("data")))).
					Exec(ctx, tx)).To(Succeed())
				// Every entry has "data" in its encoded bytes, so the
				// inverted raw rejects all of them.
				Expect(res).To(BeEmpty())
			})
		})
	})

	Describe("Multiple Where Calls", func() {
		It("Should AND three Where calls together", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(idGT(1)).
				Where(idLT(8)).
				Where(dataEQ("data")).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(6))
		})

		It("Should AND a Where call with an Or filter", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or(idEQ(1), idEQ(2), idEQ(3), idEQ(4))).
				Where(idGT(2)).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal([]entry{entries[3], entries[4]}))
		})
	})
})
