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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

// failOnceDB wraps a kv.DB and forces the first OpenIterator call routed
// directly to the DB to fail, allowing subsequent calls through to the
// underlying DB. Calls routed through OpenTx().OpenIterator(...) are not
// intercepted because tx OpenIterator dispatches off the underlying tx
// rather than the wrapper. This is exactly the failure shape needed:
// migrations open iterators via tx (so they succeed), populate opens
// iterators via DB-as-Tx (so this trips them), and the post-failure
// sequential-scan fallback in Retrieve also goes via DB-as-Tx and lands
// on the second-call branch (success).
type failOnceDB struct {
	kv.DB
	calls atomic.Int32
	err   error
}

func (f *failOnceDB) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	if f.calls.Add(1) == 1 {
		return nil, f.err
	}
	return f.DB.OpenIterator(opts)
}

var _ = Describe("Async populate", func() {
	Describe("WaitForIndexes", func() {
		var idxDB *gorp.DB
		BeforeEach(func() {
			mem := DeferClose(memkv.New())
			idxDB = gorp.Wrap(mem)
		})

		It("Should return nil for a table with no indexes", func(ctx SpecContext) {
			table := MustSucceed(gorp.OpenTable[int32, indexedEntry](
				ctx, gorp.TableConfig[int32, indexedEntry]{DB: idxDB},
			))
			DeferClose(table)
			Expect(table.WaitForIndexes(ctx)).To(Succeed())
		})

		It("Should return nil after a successful populate", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entry(&indexedEntry{ID: 1, Name: "alpha"}).
				Exec(ctx, idxDB)).To(Succeed())
			nameIdx := gorp.NewLookupIndex[int32, indexedEntry, string](
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table := MustSucceed(gorp.OpenTable[int32, indexedEntry](
				ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				},
			))
			DeferClose(table)
			Expect(table.WaitForIndexes(ctx)).To(Succeed())
		})

		It("Should return ctx.Err when ctx cancels before populate completes",
			func(ctx SpecContext) {
				nameIdx := gorp.NewLookupIndex[int32, indexedEntry, string](
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table := MustSucceed(gorp.OpenTable[int32, indexedEntry](
					ctx, gorp.TableConfig[int32, indexedEntry]{
						DB:      idxDB,
						Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
					},
				))
				DeferClose(table)
				canceledCtx, cancel := context.WithCancel(ctx)
				cancel()
				err := table.WaitForIndexes(canceledCtx)
				if err != nil {
					Expect(err).To(MatchError(context.Canceled))
				}
				// Tolerate the race where populate finishes before our
				// cancel takes effect — both outcomes are correct.
			})
	})

	Describe("Populate failure", func() {
		var (
			db      *gorp.DB
			nameIdx *gorp.LookupIndex[int32, indexedEntry, string]
		)
		BeforeEach(func(ctx SpecContext) {
			mem := DeferClose(memkv.New())
			db = gorp.Wrap(&failOnceDB{DB: mem, err: errors.New("populate boom")})
			seed := []indexedEntry{
				{ID: 1, Name: "alpha"},
				{ID: 2, Name: "beta"},
				{ID: 3, Name: "alpha"},
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&seed).Exec(ctx, db)).To(Succeed())
			nameIdx = gorp.NewLookupIndex[int32, indexedEntry, string](
				"name", func(e *indexedEntry) string { return e.Name },
			)
		})

		It("Should surface the populate error from WaitForIndexes",
			func(ctx SpecContext) {
				table := MustSucceed(gorp.OpenTable[int32, indexedEntry](
					ctx, gorp.TableConfig[int32, indexedEntry]{
						DB:      db,
						Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
					},
				))
				DeferClose(table)
				Expect(table.WaitForIndexes(ctx)).
					To(MatchError(ContainSubstring("populate boom")))
			})

		It("Should return ErrIndexInvalid from direct Get", func(ctx SpecContext) {
			table := MustSucceed(gorp.OpenTable[int32, indexedEntry](
				ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      db,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				},
			))
			DeferClose(table)
			Expect(table.WaitForIndexes(ctx)).ToNot(Succeed())
			Expect(nameIdx.Get("alpha")).Error().
				To(MatchError(gorp.ErrIndexInvalid))
		})

		It("Should fall back to a sequential scan from Retrieve via Filter",
			func(ctx SpecContext) {
				table := MustSucceed(gorp.OpenTable[int32, indexedEntry](
					ctx, gorp.TableConfig[int32, indexedEntry]{
						DB:      db,
						Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
					},
				))
				DeferClose(table)
				Expect(table.WaitForIndexes(ctx)).ToNot(Succeed())
				var got []indexedEntry
				Expect(gorp.NewRetrieve[int32, indexedEntry]().
					Where(nameIdx.Filter("alpha")).
					Entries(&got).Exec(ctx, db)).To(Succeed())
				Expect(got).To(HaveLen(2))
				ids := []int32{got[0].ID, got[1].ID}
				Expect(ids).To(ConsistOf(int32(1), int32(3)))
			})
	})
})
