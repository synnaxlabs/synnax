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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("DB", func() {
	Describe("WithTx", func() {
		It(
			"Should commit the transaction if the callback returns nil",
			func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return gorp.NewCreate[int32, entry]().Entry(&entry{ID: 1, Data: "One"}).
						Exec(ctx, tx)
				})).To(Succeed())
				var res entry
				Expect(
					gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](1)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res).To(Equal(entry{ID: 1, Data: "One"}))
			},
		)
		It(
			"Should not commit the transaction if the callback returns an error",
			func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return gorp.NewCreate[int32, entry]().Entry(&entry{ID: 1, Data: "One"}).
						Exec(ctx, tx)
				})).To(Succeed())
				Expect(
					db.WithTx(ctx, func(_ gorp.Tx) error { return query.ErrNotFound }),
				).
					To(MatchError(query.ErrNotFound))
				Expect(
					gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](2)).
						Exec(ctx, db),
				).To(MatchError(query.ErrNotFound))
			},
		)
	})

	Describe("OnCommit", func() {
		It("Should run the hook after the transaction commits", func(ctx SpecContext) {
			tx := db.OpenTx()
			ran := false
			gorp.OnCommit(tx, func() { ran = true })
			Expect(ran).To(BeFalse())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(ran).To(BeTrue())
			Expect(tx.Close()).To(Succeed())
		})

		It("Should not run the hook when the transaction closes uncommitted", func() {
			tx := db.OpenTx()
			ran := false
			gorp.OnCommit(tx, func() { ran = true })
			Expect(tx.Close()).To(Succeed())
			Expect(ran).To(BeFalse())
		})

		It("Should run the hook once when the transaction closes after a commit",
			func(ctx SpecContext) {
				tx := db.OpenTx()
				runs := 0
				gorp.OnCommit(tx, func() { runs++ })
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())
				Expect(runs).To(Equal(1))
			})

		It("Should run the hook at once for a DB used directly", func() {
			ran := false
			gorp.OnCommit(db, func() { ran = true })
			Expect(ran).To(BeTrue())
		})

		It("Should not run the hook when WithTx fails", func(ctx SpecContext) {
			ran := false
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				gorp.OnCommit(tx, func() { ran = true })
				return query.ErrNotFound
			})).To(MatchError(query.ErrNotFound))
			Expect(ran).To(BeFalse())
		})
	})

	Describe("OverrideTx", func() {
		It("Should return the override transaction if it is not nil", func() {
			tx := db.OpenTx()
			Expect(gorp.OverrideTx(db, tx)).To(Equal(tx))
			Expect(tx.Close()).To(Succeed())
		})
		It(
			"Should return the base transaction if the override transaction is nil",
			func() {
				Expect(gorp.OverrideTx(db, nil)).To(Equal(db))
			},
		)
	})

	Describe("KV", func() {
		It("Should return the underlying key-value store for the DB", func() {
			Expect(db.KV()).To(BeIdenticalTo(kvDB))
		})
	})
})
