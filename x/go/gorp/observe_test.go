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
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Observe", func() {
	var (
		db         *gorp.DB
		entryTable *gorp.Table[int32, entry]
		grapeTable *gorp.Table[int32, grape]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		entryTable = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}))
		grapeTable = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, grape]{DB: db}))
	})
	It("Should correctly observe a change to the key value store", func(ctx SpecContext) {
		tx := db.OpenTx()
		Expect(gorp.NewCreate[int32, entry]().
			Entry(&entry{ID: 42, Data: "data"}).
			Exec(ctx, tx)).To(Succeed())
		called := false
		entryTable.Observe().OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
			for ch := range r {
				Expect(ch.Value).To(Equal(entry{ID: 42, Data: "data"}))
				Expect(ch.Variant).To(Equal(change.VariantSet))
				called = true
			}
		})
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(called).To(BeTrue())
	})

	It("Should not notify for a different type than the entries written", func(ctx SpecContext) {
		tx := db.OpenTx()
		Expect(gorp.NewCreate[int32, entry]().Entry(&entry{ID: 42, Data: "data"}).Exec(ctx, tx)).To(Succeed())
		called := false
		grapeTable.Observe().OnChange(func(ctx context.Context, r gorp.TxReader[int32, grape]) {
			called = true
		})
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(called).To(BeFalse())
	})

	It("Should notify each observer with only their matching entries in a mixed transaction", func(ctx SpecContext) {
		tx := db.OpenTx()
		Expect(gorp.NewCreate[int32, entry]().Entry(&entry{ID: 1, Data: "one"}).Exec(ctx, tx)).To(Succeed())
		Expect(gorp.NewCreate[int32, entry]().Entry(&entry{ID: 2, Data: "two"}).Exec(ctx, tx)).To(Succeed())
		Expect(gorp.NewCreate[int32, grape]().Entry(&grape{ID: 100, Data: "hundred"}).Exec(ctx, tx)).To(Succeed())

		var (
			entryChanges []change.Change[int32, entry]
			grapeChanges []change.Change[int32, grape]
		)

		entryTable.Observe().OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
			for ch := range r {
				entryChanges = append(entryChanges, ch)
			}
		})
		grapeTable.Observe().OnChange(func(ctx context.Context, r gorp.TxReader[int32, grape]) {
			for ch := range r {
				grapeChanges = append(grapeChanges, ch)
			}
		})

		Expect(tx.Commit(ctx)).To(Succeed())

		Expect(entryChanges).To(HaveLen(2))
		Expect(entryChanges[0].Value).To(Equal(entry{ID: 1, Data: "one"}))
		Expect(entryChanges[1].Value).To(Equal(entry{ID: 2, Data: "two"}))

		Expect(grapeChanges).To(HaveLen(1))
		Expect(grapeChanges[0].Value).To(Equal(grape{ID: 100, Data: "hundred"}))
	})

	It("Should correctly decode the key on delete notifications", func(ctx SpecContext) {
		Expect(gorp.NewCreate[int32, entry]().
			Entry(&entry{ID: 42, Data: "data"}).
			Exec(ctx, db)).To(Succeed())

		tx := db.OpenTx()
		Expect(gorp.NewDelete[int32, entry]().Where(gorp.MatchKeys[int32, entry](42)).Exec(ctx, tx)).To(Succeed())

		var deleteChanges []change.Change[int32, entry]
		entryTable.Observe().OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
			for ch := range r {
				deleteChanges = append(deleteChanges, ch)
			}
		})

		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(deleteChanges).To(HaveLen(1))
		Expect(deleteChanges[0].Variant).To(Equal(change.VariantDelete))
		Expect(deleteChanges[0].Key).To(Equal(int32(42)))
	})
})
