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
	"fmt"
	"testing"

	"github.com/synnaxlabs/gorp"
)

// refBenchEntry holds keys of a target table through one reference.
type refBenchEntry struct {
	ID      int32
	Targets []int32
}

func (e refBenchEntry) GorpKey() int32 { return e.ID }

// refBenchTargetPool is how many distinct targets the entries of the bench share.
const refBenchTargetPool = 1000

// BenchmarkReferenceSetup measures a table open that fills one reference from 100,000
// stored entries, each holding perKey targets, as a Core start does.
func BenchmarkReferenceSetup(b *testing.B) {
	const n = 100_000
	for _, perKey := range []int{0, 1, 20} {
		b.Run(fmt.Sprintf("targets=%d", perKey), func(b *testing.B) {
			db := openBenchDB(b)
			ctx := b.Context()
			entries := make([]refBenchEntry, n)
			for i := range entries {
				entries[i] = refBenchEntry{ID: int32(i)}
				for j := range perKey {
					entries[i].Targets = append(
						entries[i].Targets,
						int32((i+j)%refBenchTargetPool),
					)
				}
			}
			if err := gorp.NewCreate[int32, refBenchEntry]().
				Entries(&entries).
				Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				ref := gorp.NewKeyReference(gorp.KeyReferenceConfig[
					int32,
					refBenchEntry,
					int32,
				]{
					Name:     "targets",
					Table:    "bench_target",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets:  func(e *refBenchEntry) []int32 { return e.Targets },
				})
				table, err := gorp.OpenTable(
					ctx,
					gorp.TableConfig[int32, refBenchEntry]{
						DB:      db,
						Indexes: []gorp.Index[int32, refBenchEntry]{ref},
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				// A query on the reference waits for it to fill.
				if _, err = gorp.NewRetrieve[int32, refBenchEntry]().
					Where(ref.Filter(0)).
					Exists(ctx, db); err != nil {
					b.Fatal(err)
				}
				if err = table.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
