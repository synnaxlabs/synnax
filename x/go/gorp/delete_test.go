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
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Delete", Ordered, func() {
	var (
		db   *gorp.DB
		kvDB kv.DB
		tx   gorp.Tx
	)
	BeforeAll(func() {
		kvDB = memkv.New()
		db = gorp.Wrap(kvDB)
	})
	AfterAll(func() { Expect(kvDB.Close()).To(Succeed()) })
	BeforeEach(func() { tx = db.OpenTx() })
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("WhereKeys", func() {
		It("Should delete an entry by key in the db", func() {
			Expect(gorp.NewCreate[int, entry]().
				Entry(&entry{ID: 1, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int, entry]().WhereKeys(1).Exec(ctx, tx)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(1).Exists(ctx, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
		It("Should NOT return an error if the entry does not exist", func() {
			Expect(gorp.NewDelete[int, entry]().WhereKeys(1).Exec(ctx, tx)).To(Succeed())
		})
	})

	Describe("Where", func() {
		It("Should delete an entry by predicate in the db", func() {
			Expect(gorp.NewCreate[int, entry]().
				Entry(&entry{ID: 1, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int, entry]().Where(func(ctx gorp.Context, e *entry) (bool, error) {
				return e.Data == "Synnax", nil
			}).Exec(ctx, tx)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(1).Exists(ctx, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("Should not return an error if the entry does not exist", func() {
			Expect(gorp.NewDelete[int, entry]().Where(func(ctx gorp.Context, e *entry) (bool, error) {
				return e.Data == "Synnax", nil
			}).Exec(ctx, tx)).To(Succeed())
		})
	})

	Describe("Guard", func() {
		It("Should prevent deletion if any of the guard functions fail", func() {
			Expect(gorp.NewCreate[int, entry]().
				Entry(&entry{ID: 1, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int, entry]().
				WhereKeys(1).
				Guard(func(_ gorp.Context, e entry) error {
					return validate.Error
				}).Exec(ctx, tx)).To(HaveOccurredAs(validate.Error))
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(1).Exists(ctx, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("Should pass the correct transaction to the gorp context of the guard clause", func() {
			Expect(gorp.NewCreate[int, entry]().
				Entry(&entry{ID: 22, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int, entry]().
				WhereKeys(22).
				Guard(func(gCtx gorp.Context, e entry) error {
					Expect(gCtx.Tx).To(BeIdenticalTo(tx))
					Expect(gCtx.Context).To(BeIdenticalTo(ctx))
					return validate.Error
				}).Exec(ctx, tx)).To(HaveOccurredAs(validate.Error))
			exists := MustSucceed(gorp.NewRetrieve[int, entry]().WhereKeys(22).Exists(ctx, tx))
			Expect(exists).To(BeTrue())
		})

	})
})
