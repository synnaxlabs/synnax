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
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Weighted search breaker", func() {
	var (
		db    *gorp.DB
		idx   *gorp.SearchIndex[int32, indexedEntry]
		table *gorp.Table[int32, indexedEntry]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(OpenGorpMsgpackDB())
		StartRun(db.Run)
		createIndexed(ctx, db,
			indexedEntry{ID: 1, Name: "pt", Category: "pressure at the inlet"},
			indexedEntry{ID: 2, Name: "pressurizer", Category: "pressure loop heater"},
			indexedEntry{ID: 3, Name: "valve", Category: "analog"},
		)
		idx = gorp.NewWeightedSearchIndex(
			"text",
			func(e *indexedEntry) []string { return []string{e.Name} },
			func(e *indexedEntry) []string {
				if e.Category == "" {
					return nil
				}
				return []string{e.Category}
			},
		)
		table = DeferClose(openIndexedTable(ctx, db, idx))
	})
	find := func(ctx SpecContext, tx gorp.Tx, term string) []int32 {
		GinkgoHelper()
		return MustSucceed(idx.Search(ctx, tx, search.Query[int32]{Term: term})).Keys
	}
	put := func(ctx SpecContext, tx gorp.Tx, e indexedEntry) {
		GinkgoHelper()
		Expect(table.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
	}

	It("Should rank a name prefix match above a description prefix match",
		func(ctx SpecContext) {
			// 2 matches "press" by a prefix of its name, 1 only by its description.
			Expect(find(ctx, db, "press")).To(Equal([]int32{2, 1}))
		})

	It("Should drop the secondary tokens of an entry whose secondary text empties",
		func(ctx SpecContext) {
			put(ctx, db, indexedEntry{ID: 3, Name: "valve"})
			Expect(find(ctx, db, "analog")).To(BeEmpty())
		})

	It("Should show a staged secondary change only to its tx, and drop it on close",
		func(ctx SpecContext) {
			tx := db.OpenTx()
			put(ctx, tx, indexedEntry{ID: 3, Name: "valve", Category: "digital"})
			Expect(find(ctx, tx, "analog")).To(BeEmpty())
			Expect(find(ctx, tx, "digital")).To(Equal([]int32{3}))
			Expect(find(ctx, db, "analog")).To(Equal([]int32{3}))
			Expect(tx.Close()).To(Succeed())
			Expect(find(ctx, db, "analog")).To(Equal([]int32{3}))
			Expect(find(ctx, db, "digital")).To(BeEmpty())
		})

	It("Should apply a staged secondary change and delete on commit",
		func(ctx SpecContext) {
			tx := db.OpenTx()
			put(ctx, tx, indexedEntry{ID: 3, Name: "valve", Category: "digital"})
			Expect(table.NewDelete().
				Where(gorp.MatchKeys[int32, indexedEntry](1)).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			Expect(find(ctx, db, "analog")).To(BeEmpty())
			Expect(find(ctx, db, "digital")).To(Equal([]int32{3}))
			Expect(find(ctx, db, "inlet")).To(BeEmpty())
		})
})
