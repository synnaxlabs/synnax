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
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("Observe", Ordered, func() {
	var (
		db  *gorp.DB
		ctx context.Context
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	BeforeEach(func() { ctx = context.Background() })
	It("Should correctly observe a change to the key value store", func() {
		tx := db.OpenTx()
		Expect(gorp.NewCreate[int, entry]().Entry(&entry{ID: 42, Data: "data"}).Exec(ctx, tx)).To(Succeed())
		called := false
		gorp.Observe[int, entry](db).OnChange(func(ctx context.Context, r gorp.TxReader[int, entry]) {
			for ch := range r {
				Expect(ch.Value).To(Equal(entry{ID: 42, Data: "data"}))
				Expect(ch.Variant).To(Equal(change.Set))
				called = true
			}
		})
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(called).To(BeTrue())
	})
	It("Should only notify for the type of the entries written", func() {
		tx := db.OpenTx()
		Expect(gorp.NewCreate[int, entry]().Entry(&entry{ID: 42, Data: "data"}).Exec(ctx, tx)).To(Succeed())
		called := false
		gorp.Observe[int, entryTwo](db).OnChange(func(ctx context.Context, r gorp.TxReader[int, entryTwo]) {
			called = true
			count := 0
			for range r {
				count++
			}
			Expect(count).To(Equal(0))
		})
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(called).To(BeTrue())
	})
})
