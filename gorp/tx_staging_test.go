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
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Transactions that stage in turn", func() {
	var (
		db     *gorp.DB
		items  *gorp.Table[int32, txItem]
		lookup *gorp.LookupIndex[int32, txItem, string]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		lookup = gorp.NewLookupIndex[int32, txItem, string](
			"name",
			func(e *txItem) string { return e.Name },
		)
		items = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, txItem]{
			DB:      db,
			Name:    "txitem",
			Indexes: []gorp.Index[int32, txItem]{lookup},
		}))
	})

	It(
		"Should let an observer of a commit write in a transaction of its own",
		func(ctx SpecContext) {
			wrote := false
			written := make(chan error, 1)
			DeferCleanup(
				db.Observe().OnChange(func(ctx context.Context, _ gorp.Commit) {
					if wrote {
						return
					}
					wrote = true
					// A commit waits for the readers of the commit before it, so the
					// reader writes on a goroutine of its own.
					go func() {
						written <- items.NewCreate().Entry(&txItem{Key: 9, Name: "observer"}).
							Exec(ctx, db)
					}()
				}),
			)
			Expect(items.NewCreate().Entry(&txItem{Key: 1, Name: "a"}).Exec(ctx, db)).
				To(Succeed())
			Eventually(written).Should(Receive(BeNil()))
			Expect(lookup.Get(ctx, db, "observer")).To(ConsistOf(int32(9)))
		},
	)

	It(
		"Should refuse a write past a failed commit and let later ones write",
		func(ctx SpecContext) {
			tx := db.OpenTx()
			Expect(items.NewCreate().Entry(&txItem{Key: 1, Name: "a"}).Exec(ctx, tx)).
				To(Succeed())
			gorp.BeforeCommit(tx, func(context.Context) error { return errRefused })
			Expect(tx.Commit(ctx)).To(MatchError(errRefused))
			Expect(items.NewCreate().Entry(&txItem{Key: 2, Name: "b"}).Exec(ctx, tx)).
				To(MatchError(gorp.ErrTxEnded))
			Expect(tx.Close()).To(Succeed())
			c := &txItem{Key: 3, Name: "c"}
			Expect(func() {
				Expect(items.NewCreate().Entry(c).Exec(ctx, db)).To(Succeed())
			}).NotTo(Panic())
			Expect(lookup.Get(ctx, db, "c")).To(ConsistOf(int32(3)))
			Expect(lookup.Get(ctx, db, "b")).To(BeEmpty())
		},
	)
})

var errRefused = errors.New("commit refused")

type txItem struct {
	Key    int32
	Name   string
	Labels []string
}

func (i txItem) GorpKey() int32 { return i.Key }

type txLabel struct{ Key string }

func (l txLabel) GorpKey() string { return l.Key }
