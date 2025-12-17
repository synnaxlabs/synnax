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
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Delete", func() {
	var (
		ctx context.Context
		tx  gorp.Tx
	)
	BeforeEach(func() {
		ctx = context.Background()
		tx = db.OpenTx()
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("WhereKeys", func() {
		It("Should delete an entry by key in the db", func() {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 1, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int32, entry]().WhereKeys(1).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(1).Exists(ctx, tx)).To(BeFalse())
		})
		It("Should NOT return an error if the entry does not exist", func() {
			Expect(gorp.NewDelete[int32, entry]().WhereKeys(1).Exec(ctx, tx)).To(Succeed())
		})
	})

	Describe("Where", func() {
		It("Should delete an entry by predicate in the db", func() {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 1, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int32, entry]().Where(func(_ gorp.Context, e *entry) (bool, error) {
				return e.Data == "Synnax", nil
			}).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(1).Exists(ctx, tx)).To(BeFalse())
		})

		It("Should not return an error if the entry does not exist", func() {
			Expect(gorp.NewDelete[int32, entry]().Where(func(_ gorp.Context, e *entry) (bool, error) {
				return e.Data == "Synnax", nil
			}).Exec(ctx, tx)).To(Succeed())
		})
	})

	Describe("Guard", func() {
		It("Should prevent deletion if any of the guard functions fail", func() {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 1, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int32, entry]().
				WhereKeys(1).
				Guard(func(_ gorp.Context, e entry) error {
					return validate.Error
				}).Exec(ctx, tx)).To(HaveOccurredAs(validate.Error))
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(1).Exists(ctx, tx)).To(BeTrue())
		})

		It("Should pass the correct transaction to the gorp context of the guard clause", func() {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 22, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int32, entry]().
				WhereKeys(22).
				Guard(func(gCtx gorp.Context, _ entry) error {
					Expect(gCtx.Tx).To(BeIdenticalTo(tx))
					Expect(gCtx.Context).To(BeIdenticalTo(ctx))
					return validate.Error
				}).Exec(ctx, tx)).To(HaveOccurredAs(validate.Error))
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(22).Exists(ctx, tx)).To(BeTrue())
		})

	})
})
