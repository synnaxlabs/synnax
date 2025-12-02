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
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("update", func() {
	var (
		ctx     context.Context
		entries []entry
		tx      gorp.Tx
	)
	BeforeEach(func() {
		ctx = context.Background()
		tx = db.OpenTx()
		entries = make([]entry, 10)
		for i := range 10 {
			entries[i] = entry{ID: i, Data: "data"}
		}
		Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	It("Should correctly update set of entries", func() {
		Expect(gorp.NewUpdate[int, entry]().
			WhereKeys(entries[0].GorpKey()).
			Change(func(_ gorp.Context, e entry) entry {
				e.Data = "new data"
				return e
			}).Exec(ctx, tx)).To(Succeed())
		var res entry
		Expect(gorp.NewRetrieve[int, entry]().
			WhereKeys(entries[0].GorpKey()).
			Entry(&res).
			Exec(ctx, tx)).To(Succeed())
		Expect(res).To(Equal(entry{ID: 0, Data: "new data"}))
	})

	It("Should return an error if no change function was specified", func() {
		Expect(gorp.NewUpdate[int, entry]().
			WhereKeys(entries[0].GorpKey()).
			Exec(ctx, tx)).To(HaveOccurredAs(query.InvalidParameters))
	})

	It("Should return an error if the the key cannot be found", func() {
		Expect(gorp.NewUpdate[int, entry]().
			WhereKeys(999).
			Change(func(_ gorp.Context, e entry) entry {
				e.Data = "new data"
				return e
			}).Exec(ctx, tx)).To(HaveOccurredAs(query.NotFound))
	})

	It("Should pass the correct transaction into the gorp.Context in the where function", func() {
		count := 0
		Expect(gorp.NewUpdate[int, entry]().
			WhereKeys(entries[0].GorpKey()).
			Change(func(gCtx gorp.Context, e entry) entry {
				e.Data = "new data"
				Expect(gCtx.Context).To(BeIdenticalTo(ctx))
				Expect(gCtx.Tx).To(BeIdenticalTo(tx))
				count++
				return e
			}).Exec(ctx, tx)).To(Succeed())
		Expect(count).To(Equal(1))
	})

	Describe("Where", func() {
		It("Should correctly update a set of entries based on a where filter function", func() {
			Expect(gorp.NewUpdate[int, entry]().
				Where(func(_ gorp.Context, e *entry) (bool, error) { return e.ID < 5, nil }).
				Change(func(_ gorp.Context, e entry) entry {
					e.Data = "new data"
					return e
				}).Exec(ctx, tx)).To(Succeed())
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Where(func(_ gorp.Context, e *entry) (bool, error) { return e.ID < 5, nil }).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			for i := range res {
				Expect(res[i]).To(Equal(entry{ID: i, Data: "new data"}))
			}
		})

		It("Should pass the correct transaction to the gorp.Context in the where function", func() {
			count := 0
			Expect(gorp.NewUpdate[int, entry]().
				WhereKeys(entries[0].GorpKey()).
				Where(func(gCtx gorp.Context, e *entry) (bool, error) {
					count++
					Expect(gCtx.Context).To(BeIdenticalTo(ctx))
					Expect(gCtx.Tx).To(BeIdenticalTo(tx))
					Expect(e).NotTo(BeNil())
					return true, nil
				}).Change(func(_ gorp.Context, e entry) entry { return e }).
				Exec(ctx, tx)).To(Succeed())
			Expect(count).To(Equal(1))
		})
	})
})
