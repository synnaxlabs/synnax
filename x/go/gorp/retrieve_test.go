// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("retrieveEntity", Ordered, func() {
	var (
		db      *gorp.DB
		kv      kv.DB
		entries []entry
		txn     gorp.ReadTxn
	)
	BeforeAll(func() {
		kv = memkv.New()
		db = gorp.Wrap(kv)
		for i := 0; i < 10; i++ {
			entries = append(entries, entry{ID: i, Data: "data"})
		}
		txn_ := db.BeginWrite(ctx)
		Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(txn_)).To(Succeed())
		txn = txn_
	})
	AfterAll(func() { Expect(kv.Close()).To(Succeed()) })
	Describe("WhereKeys", func() {
		Context("Multiple Entries", func() {
			It("Should retrieve the entry by key", func() {
				var res []entry
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entries(&res).
					Exec(txn)).To(Succeed())
				Expect(res).To(Equal([]entry{entries[0]}))
			})
			It("Should return a query.NotFound error if ANY key is not found", func() {
				var res []entry
				err := gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey(), 444444).
					Entries(&res).
					Exec(txn)
				By("Returning the correct error")
				Expect(err).To(HaveOccurredAs(query.NotFound))
				By("Still retrieving as many entries as possible")
				Expect(res).To(HaveLen(1))
			})
			It("Should still retrieve all possible entries even if some are not found", func() {
				var res []entry
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(44444, entries[0].GorpKey(), entries[1].GorpKey()).
					Entries(&res).
					Exec(txn)).To(HaveOccurredAs(query.NotFound))
				Expect(res).To(Equal(entries[:2]))
			})
			Describe("Exists", func() {
				It("Should return true if ALL keys have matching entries", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey(), entries[1].GorpKey()).
						Exists(txn)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeTrue())
				})
				It("Should return false if ANY key has no matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey(), 444444).
						Exists(txn)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeFalse())
				})
			})
		})
		Context("Single Entry", func() {
			It("Should retrieve the entry by key", func() {
				res := &entry{}
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entry(res).
					Exec(txn)).To(Succeed())
				Expect(res).To(Equal(&entries[0]))
			})
			It("Should allow for a nil entry to be provided", func() {
				var res *entry
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entry(res).
					Exec(txn)).To(Succeed())
			})
			It("Should return a query.NotFound error if the key is not found", func() {
				err := gorp.NewRetrieve[int, entry]().
					WhereKeys(444444).
					Entry(&entry{}).
					Exec(txn)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, query.NotFound)).To(BeTrue())
			})
			Describe("exists", func() {
				It("Should return true if the key has a matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey()).
						Exists(txn)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeTrue())
				})
				It("Should return false if the key has no matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(444444).
						Exists(txn)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeFalse())
				})
			})
		})
	})
	Describe("Where", func() {
		It("Should retrieve the entry by a filter parameter", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == entries[1].ID }).
				Exec(txn),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1]}))
		})
		It("Should support multiple filters", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == entries[1].ID }).
				Where(func(e *entry) bool { return e.ID == entries[2].ID }).
				Exec(txn),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[2]}))
		})
		It("Should NOT return a query.NotFound error if no entries are found", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == 444444 }).
				Exec(txn),
			).To(Succeed())
			Expect(res).To(HaveLen(0))
		})
		Describe("exists", func() {
			It("Should return true if ANY entries exist", func() {
				exists, err := gorp.NewRetrieve[int, entry]().
					Where(func(e *entry) bool { return e.ID == entries[1].ID }).
					Where(func(e *entry) bool { return e.ID == 44444 }).
					Exists(txn)
				Expect(err).To(Not(HaveOccurred()))
				Expect(exists).To(BeTrue())
			})
			It("Should return false if ALL entries do not exist", func() {
				exists, err := gorp.NewRetrieve[int, entry]().
					Where(func(e *entry) bool { return e.ID == 444444 }).
					Where(func(e *entry) bool { return e.ID == 44444 }).
					Exists(txn)
				Expect(err).To(Not(HaveOccurred()))
				Expect(exists).To(BeFalse())
			})
		})
	})
})
