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
