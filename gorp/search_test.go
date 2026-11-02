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
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func indexedText(e *indexedEntry) []string { return []string{e.Name, e.Category} }

func newTextIndex() *gorp.SearchIndex[int32, indexedEntry] {
	return gorp.NewSearchIndex("text", indexedText)
}

func createIndexed(ctx SpecContext, tx gorp.Tx, entries ...indexedEntry) {
	GinkgoHelper()
	Expect(gorp.NewCreate[int32, indexedEntry]().
		Entries(&entries).Exec(ctx, tx)).To(Succeed())
}

func retrieveIDs(
	ctx SpecContext,
	tx gorp.Tx,
	q gorp.Retrieve[int32, indexedEntry],
) []int32 {
	GinkgoHelper()
	var res []indexedEntry
	Expect(q.Entries(&res).Exec(ctx, tx)).To(Succeed())
	return idsOf(res)
}

// searchEntries index as follows: "pressure" matches 2 then 1, because 2 has fewer
// words; "sensor" matches 1 then 4; "analog" matches 3, then 1 and 4.
var searchEntries = []indexedEntry{
	{ID: 1, Name: "pressure sensor", Category: "analog"},
	{ID: 2, Name: "pressure", Category: "digital"},
	{ID: 3, Name: "valve", Category: "analog"},
	{ID: 4, Name: "temperature sensor", Category: "analog"},
}

var _ = Describe("SearchIndex", func() {
	var (
		db    *gorp.DB
		idx   *gorp.SearchIndex[int32, indexedEntry]
		table *gorp.Table[int32, indexedEntry]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(OpenGorpMsgpackDB())
		createIndexed(ctx, db, searchEntries...)
		idx = newTextIndex()
		table = DeferClose(openIndexedTable(ctx, db, idx))
	})

	stage := func(ctx SpecContext, tx gorp.Tx, entry indexedEntry) {
		GinkgoHelper()
		Expect(table.NewCreate().Entry(&entry).Exec(ctx, tx)).To(Succeed())
	}

	search := func(ctx SpecContext, tx gorp.Tx, term string) []int32 {
		GinkgoHelper()
		return MustSucceed(idx.Search(ctx, tx, term, 0)).Keys
	}

	Describe("Name", func() {
		It("Should return the name of the index", func() {
			Expect(idx.Name()).To(Equal("text"))
		})
	})

	Describe("Search", func() {
		DescribeTable("Should rank the matches of every indexed text",
			func(ctx SpecContext, term string, expected []int32) {
				Expect(search(ctx, db, term)).To(Equal(expected))
			},
			Entry("an exact word", "pressure", []int32{2, 1}),
			Entry("a word of a second text", "analog", []int32{3, 1, 4}),
			Entry("every word of a term", "pressure analog", []int32{1}),
			Entry("a prefix", "temp", []int32{4}),
			Entry("a word within one edit", "valv", []int32{3}),
			Entry("no match", "flow", []int32{}),
		)

		DescribeTable("Should keep the closest matches up to the limit",
			func(ctx SpecContext, limit int, expected []int32) {
				Expect(MustSucceed(idx.Search(ctx, db, "analog", limit)).Keys).
					To(Equal(expected))
			},
			Entry("fewer than every match", 2, []int32{3, 1}),
			Entry("more than every match", 10, []int32{3, 1, 4}),
			Entry("every match", 0, []int32{3, 1, 4}),
		)

		It("Should index entries that a committed tx writes", func(ctx SpecContext) {
			createIndexed(ctx, db, indexedEntry{ID: 5, Name: "flow meter"})
			Expect(search(ctx, db, "flow")).To(Equal([]int32{5}))
			createIndexed(ctx, db, indexedEntry{ID: 5, Name: "valve"})
			Expect(search(ctx, db, "flow")).To(BeEmpty())
			Expect(table.NewDelete().
				Where(gorp.MatchKeys[int32, indexedEntry](3)).
				Exec(ctx, db)).To(Succeed())
			Expect(search(ctx, db, "valve")).To(Equal([]int32{5}))
		})

		It("Should panic when tx is nil", func(ctx SpecContext) {
			Expect(func() { MustSucceed(idx.Search(ctx, nil, "valve", 0)) }).
				To(PanicWith(ContainSubstring("nil transaction")))
		})

		Describe("Tx", func() {
			var tx gorp.Tx
			BeforeEach(func() { tx = DeferClose(db.OpenTx()) })

			It("Should show writes staged in tx only to tx", func(ctx SpecContext) {
				stage(ctx, tx, indexedEntry{ID: 5, Name: "pressure"})
				Expect(table.NewDelete().
					Where(gorp.MatchKeys[int32, indexedEntry](2)).
					Exec(ctx, tx)).To(Succeed())
				Expect(search(ctx, tx, "pressure")).To(Equal([]int32{5, 1}))
				Expect(search(ctx, db, "pressure")).To(Equal([]int32{2, 1}))
			})

			It("Should apply staged writes when tx commits", func(ctx SpecContext) {
				stage(ctx, tx, indexedEntry{ID: 5, Name: "flow"})
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(search(ctx, db, "flow")).To(Equal([]int32{5}))
			})

			It("Should drop staged writes when tx closes", func(ctx SpecContext) {
				stage(ctx, tx, indexedEntry{ID: 5, Name: "flow"})
				Expect(tx.Close()).To(Succeed())
				Expect(search(ctx, db, "flow")).To(BeEmpty())
			})
		})
	})

	Describe("Filter", func() {
		It("Should retrieve the matches closest match first", func(ctx SpecContext) {
			Expect(retrieveIDs(ctx, db, table.NewRetrieve().
				Where(idx.Filter("analog")))).To(Equal([]int32{3, 1, 4}))
		})

		DescribeTable("Should page through the matches by relevance",
			func(ctx SpecContext, offset, limit int, expected []int32) {
				Expect(retrieveIDs(ctx, db, table.NewRetrieve().
					Where(idx.Filter("analog")).
					Offset(offset).
					Limit(limit))).To(Equal(expected))
			},
			Entry("the first page", 0, 2, []int32{3, 1}),
			Entry("a later page", 1, 2, []int32{1, 4}),
			Entry("past the end", 3, 2, []int32{}),
		)

		DescribeTable("Should keep the rank when keys narrow the matches",
			func(ctx SpecContext, limit int, expected []int32) {
				Expect(retrieveIDs(ctx, db, table.NewRetrieve().
					Where(gorp.MatchKeys[int32, indexedEntry](4, 3, 2, 1)).
					Where(idx.Filter("sensor")).
					Limit(limit))).To(Equal(expected))
			},
			Entry("every match", 0, []int32{1, 4}),
			Entry("the closest match", 1, []int32{1}),
		)

		It("Should count the matches", func(ctx SpecContext) {
			Expect(table.NewRetrieve().
				Where(idx.Filter("sensor")).
				Count(ctx, db)).To(Equal(2))
		})

		It("Should see writes staged in the tx", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			stage(ctx, tx, indexedEntry{ID: 5, Name: "valve"})
			Expect(retrieveIDs(ctx, tx, table.NewRetrieve().
				Where(idx.Filter("valve")))).To(ConsistOf(int32(3), int32(5)))
		})

		DescribeTable("Should compose with other filters",
			func(ctx SpecContext, filter func() gorp.Filter[int32, indexedEntry],
				expected []int32,
			) {
				Expect(retrieveIDs(ctx, db, table.NewRetrieve().
					Where(filter()))).To(ConsistOf(expected))
			},
			Entry("And with a predicate", func() gorp.Filter[int32, indexedEntry] {
				return gorp.And(idx.Filter("analog"), matchName("valve"))
			}, []int32{3}),
			Entry("Or with keys", func() gorp.Filter[int32, indexedEntry] {
				return gorp.Or(
					idx.Filter("valve"),
					gorp.MatchKeys[int32, indexedEntry](2),
				)
			}, []int32{2, 3}),
			Entry("Not", func() gorp.Filter[int32, indexedEntry] {
				return gorp.Not(idx.Filter("analog"))
			}, []int32{2}),
			Entry("Or with a predicate inside And",
				func() gorp.Filter[int32, indexedEntry] {
					return gorp.And(
						gorp.Or(idx.Filter("valve"), matchName("pressure")),
						gorp.MatchKeys[int32, indexedEntry](1, 2, 3, 4),
					)
				}, []int32{2, 3}),
		)
	})

	Describe("Populate failure", func() {
		var (
			failing    *gorp.DB
			failed     *gorp.SearchIndex[int32, indexedEntry]
			categories *gorp.LookupIndex[int32, indexedEntry, string]
		)
		BeforeEach(func(ctx SpecContext) {
			failing = gorp.Wrap(
				&failOnceDB{DB: DeferClose(memkv.New()), err: errors.New("boom")},
				gorp.WithCodec(msgpack.Codec),
			)
			createIndexed(ctx, failing, searchEntries...)
			failed = newTextIndex()
			categories = gorp.NewLookupIndex(
				"category", func(e *indexedEntry) string { return e.Category },
			)
			DeferClose(openIndexedTable(ctx, failing, categories, failed))
			Expect(categories.Get(nil, "x")).Error().
				To(MatchError(gorp.ErrIndexInvalid))
		})

		It("Should search by scanning every entry", func(ctx SpecContext) {
			Expect(MustSucceed(failed.Search(ctx, failing, "analog", 0)).Keys).
				To(Equal([]int32{3, 1, 4}))
			Expect(MustSucceed(failed.Search(ctx, failing, "analog", 2)).Keys).
				To(Equal([]int32{3, 1}))
		})

		It("Should retrieve the matches closest match first", func(ctx SpecContext) {
			Expect(retrieveIDs(ctx, failing, gorp.NewRetrieve[int32, indexedEntry]().
				Where(failed.Filter("analog")))).To(Equal([]int32{3, 1, 4}))
		})

		It("Should return the error of an entry the scan cannot decode",
			func(ctx SpecContext) {
				corrupt[int32, indexedEntry](ctx, failing, 1)
				Expect(failed.Search(ctx, failing, "analog", 0)).Error().
					To(MatchError(ContainSubstring("failed to decode")))
			})

		It("Should compose with a sibling index that falls back to a scan",
			func(ctx SpecContext) {
				Expect(
					retrieveIDs(ctx, failing, gorp.NewRetrieve[int32, indexedEntry]().
						Where(gorp.And(failed.Filter("sensor"), categories.Filter("analog"))),
					),
				).To(Equal([]int32{1, 4}))
				Expect(
					retrieveIDs(ctx, failing, gorp.NewRetrieve[int32, indexedEntry]().
						Where(gorp.Or(failed.Filter("valve"), categories.Filter("digital"))),
					),
				).To(ConsistOf(int32(2), int32(3)))
			})
	})
})

func matchName(name string) gorp.Filter[int32, indexedEntry] {
	return gorp.Match(func(_ gorp.Context, e *indexedEntry) (bool, error) {
		return e.Name == name, nil
	})
}
