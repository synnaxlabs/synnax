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
	"slices"
	"strconv"
	"testing"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
)

// refBenchHolder references target entries by key.
type refBenchHolder struct {
	Key     int32
	Note    string
	Targets []int32
}

func (h refBenchHolder) GorpKey() int32 { return h.Key }

func newRefBenchReference() *gorp.KeyReference[int32, refBenchHolder, int32] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[int32, refBenchHolder, int32]{
		Name:     "targets",
		Table:    "ref_target",
		OnDelete: gorp.Detach,
		Kind:     gorp.Use,
		Targets:  func(h *refBenchHolder) []int32 { return h.Targets },
		Detach: func(h *refBenchHolder, t int32) {
			h.Targets = slices.DeleteFunc(slices.Clone(h.Targets), func(k int32) bool {
				return k == t
			})
		},
	})
}

// openRefBench opens a target table of n+1 entries and a holder table, and creates a
// holder that references the first n targets.
func openRefBench(
	b *testing.B,
	n int,
) (*gorp.DB, *gorp.Table[int32, refBenchHolder], refBenchHolder) {
	b.Helper()
	db := openBenchDB(b)
	ctx := b.Context()
	targets, err := gorp.OpenTable(ctx, gorp.TableConfig[int32, benchEntry]{
		DB:   db,
		Name: "ref_target",
	})
	if err != nil {
		b.Fatal(err)
	}
	entries := make([]benchEntry, n+1)
	for i := range entries {
		entries[i] = benchEntry{ID: int32(i)}
	}
	if err = targets.NewCreate().Entries(&entries).Exec(ctx, db); err != nil {
		b.Fatal(err)
	}
	holders, err := gorp.OpenTable(ctx, gorp.TableConfig[int32, refBenchHolder]{
		DB:         db,
		Name:       "ref_holder",
		References: []gorp.Reference[int32, refBenchHolder]{newRefBenchReference()},
	})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := errors.Join(holders.Close(), targets.Close()); err != nil {
			b.Error(err)
		}
	})
	h := refBenchHolder{Key: 1, Targets: make([]int32, n)}
	for i := range h.Targets {
		h.Targets[i] = int32(i)
	}
	if err = holders.NewCreate().Entry(&h).Exec(ctx, db); err != nil {
		b.Fatal(err)
	}
	return db, holders, h
}

// BenchmarkReferenceWrite measures a write of a holder that references n targets: an
// edit that keeps every target, and one that moves its last target.
func BenchmarkReferenceWrite(b *testing.B) {
	for _, n := range []int{1, 10, 300} {
		for _, edit := range []string{"keep", "move"} {
			b.Run(fmt.Sprintf("%s/n=%d", edit, n), func(b *testing.B) {
				db, holders, h := openRefBench(b, n)
				ctx := b.Context()
				b.ReportAllocs()
				b.ResetTimer()
				for i := range b.N {
					h.Note = strconv.Itoa(i)
					if edit == "move" {
						h.Targets[n-1] = int32(n - 1 + i%2)
					}
					if err := holders.NewCreate().Entry(&h).Exec(ctx, db); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
