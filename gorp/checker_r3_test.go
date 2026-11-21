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

var _ = Describe("Checker r3: a write through an ended transaction", func() {
	var (
		db     *gorp.DB
		items  *gorp.Table[int32, r2Item]
		lookup *gorp.LookupIndex[int32, r2Item, string]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		lookup = gorp.NewLookupIndex[int32, r2Item, string](
			"name",
			func(e *r2Item) string { return e.Name },
		)
		items = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, r2Item]{
			DB:      db,
			Name:    "r3item",
			Indexes: []gorp.Index[int32, r2Item]{lookup},
		}))
	})

	It(
		"Should not panic past a failed commit while another transaction stages",
		func(ctx SpecContext) {
			dead := db.OpenTx()
			Expect(items.NewCreate().Entry(&r2Item{Key: 1, Name: "a"}).Exec(ctx, dead)).
				To(Succeed())
			gorp.BeforeCommit(dead, func(context.Context) error { return errR2 })
			Expect(dead.Commit(ctx)).To(MatchError(errR2))
			live := db.OpenTx()
			Expect(items.NewCreate().Entry(&r2Item{Key: 2, Name: "b"}).Exec(ctx, live)).
				To(Succeed())
			var deadErr error
			Expect(func() {
				deadErr = items.NewCreate().
					Entry(&r2Item{Key: 3, Name: "c"}).
					Exec(ctx, dead)
			}).NotTo(Panic())
			GinkgoWriter.Printf("write through the ended transaction: %v\n", deadErr)
			Expect(live.Commit(ctx)).To(Succeed())
			Expect(dead.Close()).To(Succeed())
			Expect(lookup.Get(ctx, db, "b")).To(ConsistOf(int32(2)))
			Expect(lookup.Get(ctx, db, "c")).To(BeEmpty())
		},
	)
})
