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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Search gate breaker", func() {
	// 5831318722 stages 1,000 writes on a plain search index and searches them in
	// 1,753,839 to 1,756,163 bytes per op. The budget allows 2% over that.
	const budget = 1_791_000

	It("Should stage writes on an index with no secondary text in the bytes of base",
		func(ctx SpecContext) {
			db := DeferClose(OpenGorpMsgpackDB())
			StartRun(db.Run)
			idx := gorp.NewSearchIndex(
				"search",
				func(e *indexBenchEntry) []string { return []string{e.Name} },
			)
			table := DeferClose(MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{
					DB:      db,
					Indexes: []gorp.Index[int32, indexBenchEntry]{idx},
				},
			)))
			entries := makeIndexBenchEntries(1000)
			res := testing.Benchmark(func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					tx := db.OpenTx()
					Expect(table.NewCreate().Entries(&entries).Exec(ctx, tx)).
						To(Succeed())
					MustSucceed(idx.Search(
						ctx, tx, search.Query[int32]{Term: "name", Limit: 10},
					))
					Expect(tx.Close()).To(Succeed())
				}
			})
			Expect(res.AllocedBytesPerOp()).To(BeNumerically("<=", budget))
		})
})
