// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("Observe", Ordered, func() {
	var (
		db   *gorp.DB
		kvDB kv.DB
	)
	BeforeEach(func() {
		kvDB = memkv.New()
		db = gorp.Wrap(kvDB)
	})
	AfterEach(func() { Expect(kvDB.Close()).To(Succeed()) })
	It("Should correctly observe a change to the key value store", func() {
		tx := db.OpenTx()
		Expect(gorp.NewCreate[int32, entry]().Entry(&entry{ID: 42, Data: "data"}).Exec(ctx, tx)).To(Succeed())
		called := false
		gorp.Observe[int32, entry](db).OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
			ch, ok := r.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(ch.Value).To(Equal(entry{ID: 42, Data: "data"}))
			Expect(ch.Variant).To(Equal(change.Set))
			called = true
			ch, ok = r.Next(ctx)
			Expect(ok).To(BeFalse())
		})
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(called).To(BeTrue())
	})
	It("Should only notify for the type of the entries written", func() {
		tx := db.OpenTx()
		Expect(gorp.NewCreate[int32, entry]().Entry(&entry{ID: 42, Data: "data"}).Exec(ctx, tx)).To(Succeed())
		called := false
		gorp.Observe[int32, entryTwo](db).OnChange(func(ctx context.Context, r gorp.TxReader[int32, entryTwo]) {
			called = true
			_, ok := r.Next(ctx)
			Expect(ok).To(BeFalse())
		})
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(called).To(BeTrue())
	})
})
