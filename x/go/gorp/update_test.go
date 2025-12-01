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
	"github.com/synnaxlabs/x/testutil"
)

var _ = Describe("update", Ordered, func() {
	var (
		db      *gorp.DB
		kv      xkv.DB
		entries []entry
		tx      gorp.Tx
	)
	BeforeAll(func() {
		kv = memkv.New()
		db = gorp.Wrap(kv)
		for i := range 10 {
			entries = append(entries, entry{ID: i, Data: "data"})
		}
	})
	AfterAll(func() { Expect(kv.Close()).To(Succeed()) })
	BeforeEach(func() {
		tx = db.OpenTx()
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
			Exec(ctx, tx)).To(testutil.HaveOccurredAs(query.InvalidParameters))
	})

	It("Should return an error if the the key cannot be found", func() {
		Expect(gorp.NewUpdate[int, entry]().
			WhereKeys(999).
			Change(func(_ gorp.Context, e entry) entry {
				e.Data = "new data"
				return e
			}).Exec(ctx, tx)).To(testutil.HaveOccurredAs(query.NotFound))
	})

	It("Should pass the correct transaction into the gorp.Context in the where function", func() {
		count := 0
		Expect(gorp.NewUpdate[int, entry]().
			WhereKeys(entries[0].GorpKey()).
			Change(func(uCtx gorp.Context, e entry) entry {
				e.Data = "new data"
				Expect(uCtx.Context).To(BeIdenticalTo(ctx))
				Expect(uCtx.Tx).To(BeIdenticalTo(tx))
				count++
				return e
			}).Exec(ctx, tx)).To(Succeed())
		Expect(count).To(Equal(1))
	})

})
