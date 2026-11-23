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
	"github.com/synnaxlabs/x/errors"
)

// openDeletePlanBench opens n targets that dependentsBenchRefs weak references point
// at, with 3 holders each.
func openDeletePlanBench(
	b *testing.B,
	n int,
) (*gorp.DB, *gorp.Table[int32, benchEntry]) {
	b.Helper()
	db := openBenchDB(b)
	ctx := b.Context()
	targets, err := gorp.OpenTable(ctx, gorp.TableConfig[int32, benchEntry]{
		DB:     db,
		Name:   "target",
		Issued: func(int32) bool { return true },
	})
	if err != nil {
		b.Fatal(err)
	}
	holderTable, err := gorp.OpenTable(
		ctx,
		gorp.TableConfig[int32, dependentsBenchHolder]{
			DB:         db,
			Name:       "holder",
			References: newDependentsBenchRefs(),
		},
	)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := errors.Join(holderTable.Close(), targets.Close()); err != nil {
			b.Error(err)
		}
	})
	entries := make([]benchEntry, n)
	for i := range entries {
		entries[i] = benchEntry{ID: int32(i), Data: "target"}
	}
	if err = targets.NewCreate().Entries(&entries).Exec(ctx, db); err != nil {
		b.Fatal(err)
	}
	held := make([]dependentsBenchHolder, 3*n)
	for i := range held {
		held[i] = dependentsBenchHolder{Key: int32(i), Targets: []int32{int32(i % n)}}
	}
	if err = holderTable.NewCreate().Entries(&held).Exec(ctx, db); err != nil {
		b.Fatal(err)
	}
	return db, targets
}

// BenchmarkDeletePlan measures PlanDelete against the delete that it plans, for 1 and
// 100 targets that dependentsBenchRefs weak references point at, with 3 holders each.
// Each run uses a transaction that it then discards. The plan target is at most 1.10 x
// the delete.
func BenchmarkDeletePlan(b *testing.B) {
	const targetCount = 100
	db, targets := openDeletePlanBench(b, targetCount)
	ctx := b.Context()
	run := func(b *testing.B, op func(tx gorp.Tx) error) {
		b.Helper()
		b.ReportAllocs()
		for b.Loop() {
			tx := db.OpenTx()
			if err := errors.Join(op(tx), tx.Close()); err != nil {
				b.Fatal(err)
			}
		}
	}
	for _, n := range []int{1, targetCount} {
		keys := make([]int32, n)
		for i := range keys {
			keys[i] = int32(i)
		}
		b.Run(fmt.Sprintf("delete/n=%d", n), func(b *testing.B) {
			run(b, func(tx gorp.Tx) error {
				return targets.NewDelete().
					Where(gorp.MatchKeys[int32, benchEntry](keys...)).
					Exec(ctx, tx)
			})
		})
		b.Run(fmt.Sprintf("plan/n=%d", n), func(b *testing.B) {
			run(b, func(tx gorp.Tx) error {
				_, err := targets.PlanDelete(ctx, tx, keys)
				return err
			})
		})
	}
}
