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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", Ordered, func() {
	var (
		db      *gorp.DB
		kv      kv.DB
		entries []entry
		tx      gorp.Tx
	)
	BeforeAll(func() {
		kv = memkv.New()
		db = gorp.Wrap(kv)
		for i := 0; i < 10; i++ {
			entries = append(entries, entry{ID: i, Data: "data"})
		}

	})
	AfterAll(func() { Expect(kv.Close()).To(Succeed()) })
	BeforeEach(func() {
		tx = db.OpenTx()
		Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
	Describe("WhereKeys", func() {
		Context("Multiple Entries", func() {
			It("Should retrieve the entry by key", func() {
				var res []entry
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(Equal([]entry{entries[0]}))
			})
			It("Should return a query.NotFound error if ANY key is not found", func() {
				var res []entry
				err := gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey(), 444444).
					Entries(&res).
					Exec(ctx, tx)
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
					Exec(ctx, tx)).To(HaveOccurredAs(query.NotFound))
				Expect(res).To(Equal(entries[:2]))
			})
			Describe("Exists", func() {
				It("Should return true if ALL keys have matching entries", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey(), entries[1].GorpKey()).
						Exists(ctx, tx)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeTrue())
				})
				It("Should return false if ANY key has no matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey(), 444444).
						Exists(ctx, tx)
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
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(Equal(&entries[0]))
			})
			It("Should allow for a nil entry to be provided", func() {
				var res *entry
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entry(res).
					Exec(ctx, tx)).To(Succeed())
			})
			It("Should return a query.NotFound error if the key is not found", func() {
				err := gorp.NewRetrieve[int, entry]().
					WhereKeys(444444).
					Entry(&entry{}).
					Exec(ctx, tx)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, query.NotFound)).To(BeTrue())
			})
			It("Should return a query.NotFound error if the where clause matches no entry", func() {
				err := gorp.NewRetrieve[int, entry]().
					Where(func(e *entry) bool { return e.ID == 241241 }).
					Entry(&entry{}).
					Exec(ctx, tx)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, query.NotFound)).To(BeTrue())
			})
			Describe("exists", func() {
				It("Should return true if the key has a matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey()).
						Exists(ctx, tx)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeTrue())
				})
				It("Should return false if the key has no matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(444444).
						Exists(ctx, tx)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeFalse())
				})
			})
		})
	})
	Describe("WherePrefix", func() {
		It("Should retrieve the entry by a prefix", func() {
			r := prefixEntry{
				ID:   123,
				Data: "data",
			}
			r2 := prefixEntry{
				ID:   456,
				Data: "data",
			}
			Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r2).Exec(ctx, tx)).To(Succeed())
			var res []prefixEntry
			Expect(gorp.NewRetrieve[[]byte, prefixEntry]().
				WherePrefix([]byte("prefix-123")).
				Entries(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]prefixEntry{r}))
		})

	})
	Describe("Where", func() {
		It("Should retrieve the entry by a filter parameter", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == entries[1].ID }).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1]}))
		})
		It("Should support isMultiple filters", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == entries[1].ID }).
				Where(func(e *entry) bool { return e.ID == entries[2].ID }).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[2]}))
		})
		It("Should require a filter to match when gorp.Required()", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == entries[1].ID }, gorp.Required()).
				Where(func(e *entry) bool { return e.ID == entries[2].ID }).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1]}))
		})
		It("Should NOT return a query.NamesNotFound error if no entries are found", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == 444444 }).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(HaveLen(0))
		})
		Describe("exists", func() {
			It("Should return true if ANY entries exist", func() {
				exists, err := gorp.NewRetrieve[int, entry]().
					Where(func(e *entry) bool { return e.ID == entries[1].ID }).
					Where(func(e *entry) bool { return e.ID == 44444 }).
					Exists(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(exists).To(BeTrue())
			})
			It("Should return false if ALL entries do not exist", func() {
				exists, err := gorp.NewRetrieve[int, entry]().
					Where(func(e *entry) bool { return e.ID == 444444 }).
					Where(func(e *entry) bool { return e.ID == 44444 }).
					Exists(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(exists).To(BeFalse())
			})
		})
		Describe("Limit", func() {
			It("Should limit the number of entries returned", func() {
				toCreate := 100
				var entries []entry
				for i := 0; i < toCreate; i++ {
					entries = append(entries, entry{ID: i, Data: "data"})
				}
				Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
				var res []entry
				Expect(gorp.NewRetrieve[int, entry]().
					Entries(&res).
					Limit(10).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(10))
			})
		})
		Describe("Offset", func() {
			It("Should offset the entries returned", func() {
				toCreate := 100
				var entries []entry
				for i := 0; i < toCreate; i++ {
					entries = append(entries, entry{ID: i, Data: "data"})
				}
				Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
				var res []entry
				Expect(gorp.NewRetrieve[int, entry]().
					Entries(&res).
					Offset(10).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(90))
			})
		})
		Describe("Limit + Offset", func() {
			It("Should limit and offset the entries returned", func() {
				toCreate := 100
				var entries []entry
				for i := 0; i < toCreate; i++ {
					entries = append(entries, entry{ID: i, Data: "data"})
				}
				Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
				var res []entry
				Expect(gorp.NewRetrieve[int, entry]().
					Entries(&res).
					Limit(10).
					Offset(10).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(10))
			})
		})
	})
	Describe("No Parameters", func() {
		It("Should return all entries for the given type", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal(entries))
		})
	})
})
