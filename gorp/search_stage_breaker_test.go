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
	"context"
	"testing"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/search"
)

// BenchmarkBreakerSearchStage stages 1,000 writes in a tx on a plain search index and
// searches them in the tx.
func BenchmarkBreakerSearchStage(b *testing.B) {
	db := openBenchDB(b)
	ctx := context.Background()
	idx := gorp.NewSearchIndex(
		"search",
		func(e *indexBenchEntry) []string { return []string{e.Name} },
	)
	table, err := gorp.OpenTable(ctx, gorp.TableConfig[int32, indexBenchEntry]{
		DB:      db,
		Indexes: []gorp.Index[int32, indexBenchEntry]{idx},
	})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = table.Close() })
	entries := makeIndexBenchEntries(1000)
	b.ReportAllocs()
	for b.Loop() {
		tx := db.OpenTx()
		if err := table.NewCreate().Entries(&entries).Exec(ctx, tx); err != nil {
			b.Fatal(err)
		}
		if _, err := idx.Search(
			ctx, tx, search.Query[int32]{Term: "name", Limit: 10},
		); err != nil {
			b.Fatal(err)
		}
		_ = tx.Close()
	}
}
