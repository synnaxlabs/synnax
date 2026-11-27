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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// txEnded is the error of a use of a transaction that has ended.
const txEnded = "[gorp] - the transaction has ended"

var _ = Describe("An ended transaction", func() {
	var (
		db     *gorp.DB
		items  *gorp.Table[int32, txItem]
		labels *gorp.Table[string, txLabel]
		lookup *gorp.LookupIndex[int32, txItem, string]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		labels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, txLabel]{
			DB:   db,
			Name: "txlabel",
		}))
		lookup = gorp.NewLookupIndex[int32, txItem, string](
			"name",
			func(e *txItem) string { return e.Name },
		)
		items = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, txItem]{
			DB:      db,
			Name:    "txitem",
			Indexes: []gorp.Index[int32, txItem]{lookup},
			References: []gorp.Reference[int32, txItem]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[int32, txItem, string]{
					Name:     "labels",
					Table:    "txlabel",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets:  func(e *txItem) []string { return e.Labels },
				}),
			},
		}))
		Expect(labels.NewCreate().Entry(&txLabel{Key: "l"}).Exec(ctx, db)).To(Succeed())
		Expect(items.NewCreate().Entry(&txItem{Key: 1, Name: "a"}).Exec(ctx, db)).
			To(Succeed())
	})

	failCommit := func(ctx context.Context) gorp.Tx {
		tx := db.OpenTx()
		Expect(items.NewCreate().Entry(&txItem{Key: 9, Name: "z"}).Exec(ctx, tx)).
			To(Succeed())
		gorp.BeforeCommit(tx, func(context.Context) error { return errRefused })
		Expect(tx.Commit(ctx)).To(MatchError(errRefused))
		return tx
	}

	commit := func(ctx context.Context) gorp.Tx {
		tx := db.OpenTx()
		Expect(items.NewCreate().Entry(&txItem{Key: 9, Name: "z"}).Exec(ctx, tx)).
			To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		return tx
	}

	closeTx := func(context.Context) gorp.Tx {
		tx := db.OpenTx()
		Expect(tx.Close()).To(Succeed())
		return tx
	}

	DescribeTableSubtree("Should refuse every use but Close",
		func(end func(context.Context) gorp.Tx) {
			It("Should refuse a write", func(ctx SpecContext) {
				tx := end(ctx)
				Expect(items.NewCreate().
					Entry(&txItem{Key: 2, Name: "b", Labels: []string{"l"}}).
					Exec(ctx, tx)).To(MatchError(txEnded))
				Expect(lookup.Get(ctx, db, "b")).To(BeEmpty())
			})

			It("Should refuse a delete", func(ctx SpecContext) {
				tx := end(ctx)
				Expect(labels.NewDelete().
					Where(gorp.MatchKeys[string, txLabel]("l")).
					Exec(ctx, tx)).To(MatchError(txEnded))
				Expect(labels.NewRetrieve().
					Where(gorp.MatchKeys[string, txLabel]("l")).
					Exists(ctx, db)).To(BeTrue())
			})

			It("Should refuse a read by key", func(ctx SpecContext) {
				tx := end(ctx)
				var got []txItem
				Expect(items.NewRetrieve().
					Where(gorp.MatchKeys[int32, txItem](1)).
					Entries(&got).
					Exec(ctx, tx)).To(MatchError(txEnded))
			})

			It("Should refuse a read through an index filter", func(ctx SpecContext) {
				tx := end(ctx)
				var got []txItem
				Expect(items.NewRetrieve().
					Where(lookup.Filter("a")).
					Entries(&got).
					Exec(ctx, tx)).To(MatchError(txEnded))
			})

			It("Should refuse a read of the index", func(ctx SpecContext) {
				tx := end(ctx)
				Expect(lookup.Get(ctx, tx, "a")).Error().To(MatchError(txEnded))
			})

			It("Should refuse a commit", func(ctx SpecContext) {
				tx := end(ctx)
				Expect(tx.Commit(ctx)).To(MatchError(txEnded))
			})

			It("Should let Close pass and do nothing", func(ctx SpecContext) {
				tx := end(ctx)
				Expect(tx.Close()).To(Succeed())
				Expect(tx.Close()).To(Succeed())
			})
		},
		Entry("past a failed commit", failCommit),
		Entry("past a commit", commit),
		Entry("past a close", closeTx),
	)

	It(
		"Should refuse a write past a failed commit while another transaction stages",
		func(ctx SpecContext) {
			dead := failCommit(ctx)
			live := db.OpenTx()
			Expect(items.NewCreate().Entry(&txItem{Key: 2, Name: "b"}).Exec(ctx, live)).
				To(Succeed())
			var deadErr error
			Expect(func() {
				deadErr = items.NewCreate().
					Entry(&txItem{Key: 3, Name: "c"}).
					Exec(ctx, dead)
			}).NotTo(Panic())
			Expect(deadErr).To(MatchError(txEnded))
			Expect(lookup.Get(ctx, dead, "b")).Error().To(MatchError(txEnded))
			Expect(live.Commit(ctx)).To(Succeed())
			Expect(lookup.Get(ctx, db, "b")).To(ConsistOf(int32(2)))
			Expect(lookup.Get(ctx, db, "c")).To(BeEmpty())
		},
	)
})
