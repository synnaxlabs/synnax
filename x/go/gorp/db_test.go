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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB", Ordered, func() {
	var (
		kv xkv.DB
		db *gorp.DB
	)
	BeforeAll(func() {
		kv = memkv.New()
		db = gorp.Wrap(kv)
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })

	Describe("WithTx", func() {
		It("Should commit the transaction if the callback returns nil", func() {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewCreate[int32, entry]().Entry(&entry{ID: 1, Data: "One"}).Exec(ctx, tx)
			})).To(Succeed())
			var res entry
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(1).Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(entry{ID: 1, Data: "One"}))
		})
		It("Should not commit the transaction if the callback returns an error", func() {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewCreate[int32, entry]().Entry(&entry{ID: 1, Data: "One"}).Exec(ctx, tx)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_ = gorp.NewCreate[int32, entry]().Entry(&entry{ID: 2, Data: "Two"}).Exec(ctx, tx)
				return query.NotFound
			})).ToNot(Succeed())
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(2).Exec(ctx, db)).To(HaveOccurredAs(query.NotFound))
		})
	})

	Describe("OverrideTx", func() {
		It("Should return the override transaction if it is not nil", func() {
			tx := db.OpenTx()
			Expect(gorp.OverrideTx(db, tx)).To(Equal(tx))
			Expect(tx.Close()).To(Succeed())
		})
		It("Should return the base transaction if the override transaction is nil", func() {
			Expect(gorp.OverrideTx(db, nil)).To(Equal(db))
		})
	})

	Describe("KV", func() {
		It("Should return the underlying key-value store for the DB", func() {
			Expect(db.KV()).To(BeIdenticalTo(kv))
		})
	})
})
