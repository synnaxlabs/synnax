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
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == 241241, nil }).
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
		Context("With byte-slice keys", func() {
			It("Should retrieve a single entry by exact prefix", func() {
				r := prefixEntry{ID: 123, Data: "data"}
				r2 := prefixEntry{ID: 456, Data: "data"}
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

			It("Should retrieve multiple entries matching a common prefix", func() {
				r1 := prefixEntry{ID: 100, Data: "first"}
				r2 := prefixEntry{ID: 101, Data: "second"}
				r3 := prefixEntry{ID: 200, Data: "third"}
				Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r1).Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r2).Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r3).Exec(ctx, tx)).To(Succeed())
				var res []prefixEntry
				Expect(gorp.NewRetrieve[[]byte, prefixEntry]().
					WherePrefix([]byte("prefix-10")).
					Entries(&res).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(2))
			})

			It("Should return empty results when prefix doesn't match any entries", func() {
				r := prefixEntry{ID: 123, Data: "data"}
				Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r).Exec(ctx, tx)).To(Succeed())
				var res []prefixEntry
				Expect(gorp.NewRetrieve[[]byte, prefixEntry]().
					WherePrefix([]byte("nonexistent-")).
					Entries(&res).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(BeEmpty())
			})

			It("Should retrieve all entries with common base prefix", func() {
				r1 := prefixEntry{ID: 1, Data: "one"}
				r2 := prefixEntry{ID: 2, Data: "two"}
				Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r1).Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r2).Exec(ctx, tx)).To(Succeed())
				var res []prefixEntry
				Expect(gorp.NewRetrieve[[]byte, prefixEntry]().
					WherePrefix([]byte("prefix-")).
					Entries(&res).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(2))
			})
		})

		Context("With string keys", func() {
			It("Should retrieve entries with matching string key prefix", func() {
				e1 := stringEntry{Key: "device-001", Data: "first"}
				e2 := stringEntry{Key: "device-002", Data: "second"}
				e3 := stringEntry{Key: "sensor-001", Data: "third"}
				Expect(gorp.NewCreate[string, stringEntry]().Entry(&e1).Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewCreate[string, stringEntry]().Entry(&e2).Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewCreate[string, stringEntry]().Entry(&e3).Exec(ctx, tx)).To(Succeed())
				var res []stringEntry
				Expect(gorp.NewRetrieve[string, stringEntry]().
					WherePrefix([]byte("device-")).
					Entries(&res).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(2))
				for _, r := range res {
					Expect(r.Key).To(HavePrefix("device-"))
				}
			})

			It("Should retrieve single entry with exact string prefix match", func() {
				e1 := stringEntry{Key: "alpha-123", Data: "first"}
				e2 := stringEntry{Key: "alpha-456", Data: "second"}
				Expect(gorp.NewCreate[string, stringEntry]().Entry(&e1).Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewCreate[string, stringEntry]().Entry(&e2).Exec(ctx, tx)).To(Succeed())
				var res []stringEntry
				Expect(gorp.NewRetrieve[string, stringEntry]().
					WherePrefix([]byte("alpha-123")).
					Entries(&res).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].Key).To(Equal("alpha-123"))
			})

			It("Should return empty results when string prefix doesn't match", func() {
				e := stringEntry{Key: "existing-key", Data: "data"}
				Expect(gorp.NewCreate[string, stringEntry]().Entry(&e).Exec(ctx, tx)).To(Succeed())
				var res []stringEntry
				Expect(gorp.NewRetrieve[string, stringEntry]().
					WherePrefix([]byte("nonexistent-")).
					Entries(&res).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(BeEmpty())
			})
		})
	})
	Describe("Where", func() {
		It("Should retrieve the entry by a filter parameter", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == entries[1].ID, nil }).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1]}))
		})
		It("Should support isMultiple filters", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == entries[1].ID, nil }).
				Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == entries[2].ID, nil }).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[2]}))
		})
		It("Should require a filter to match when gorp.Required()", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == entries[1].ID, nil }, gorp.Required()).
				Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == entries[2].ID, nil }).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1]}))
		})
		It("Should NOT return a query.NamesNotFound error if no entries are found", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == 444444, nil }).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(HaveLen(0))
		})
		Describe("exists", func() {
			It("Should return true if ANY entries exist", func() {
				exists, err := gorp.NewRetrieve[int, entry]().
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == entries[1].ID, nil }).
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == 44444, nil }).
					Exists(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(exists).To(BeTrue())
			})
			It("Should return false if ALL entries do not exist", func() {
				exists, err := gorp.NewRetrieve[int, entry]().
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == 444444, nil }).
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID == 44444, nil }).
					Exists(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(exists).To(BeFalse())
			})
		})
		Describe("Limit", func() {
			It("Should limit the number of entries returned", func() {
				toCreate := 100
				var entries []entry
				for i := range toCreate {
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
				for i := range toCreate {
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
				for i := range toCreate {
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
	Describe("Count", func() {
		Context("WhereKeys", func() {
			It("Should return the count of existing keys", func() {
				count, err := gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey(), entries[1].GorpKey()).
					Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(2))
			})

			It("Should handle non-existent keys", func() {
				count, err := gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey(), 444444).
					Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(1))
			})
		})

		Context("Where", func() {
			It("Should count entries matching a filter", func() {
				count, err := gorp.NewRetrieve[int, entry]().
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID < 5, nil }).
					Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(5))
			})

			It("Should return zero for non-matching filters", func() {
				count, err := gorp.NewRetrieve[int, entry]().
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID > 100, nil }).
					Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(0))
			})

			It("Should handle multiple filters", func() {
				count, err := gorp.NewRetrieve[int, entry]().
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID < 5, nil }, gorp.Required()).
					Where(func(ctx gorp.Context, e *entry) (bool, error) { return e.ID > 2, nil }, gorp.Required()).
					Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(2)) // Should count entries with ID 3 and 4
			})

			It("Should pass the correct transaction in the Context of the Where clause", func() {
				callCount := 0
				count, err := gorp.NewRetrieve[int, entry]().
					Where(func(wCtx gorp.Context, e *entry) (bool, error) {
						callCount++
						Expect(wCtx.Context).To(BeIdenticalTo(ctx))
						Expect(wCtx.Tx).To(BeIdenticalTo(tx))
						return false, nil
					}).Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(0))
				Expect(callCount).To(BeNumerically(">", 0))
			})
		})

		Context("WherePrefix", func() {
			BeforeEach(func() {
				r1 := prefixEntry{ID: 123, Data: "data"}
				r2 := prefixEntry{ID: 456, Data: "data"}
				Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r1).Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewCreate[[]byte, prefixEntry]().Entry(&r2).Exec(ctx, tx)).To(Succeed())
			})

			It("Should count entries matching a prefix", func() {
				count, err := gorp.NewRetrieve[[]byte, prefixEntry]().
					WherePrefix([]byte("prefix-123")).
					Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(1))
			})

			It("Should return zero for non-matching prefix", func() {
				count, err := gorp.NewRetrieve[[]byte, prefixEntry]().
					WherePrefix([]byte("nonexistent-prefix")).
					Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(0))
			})
		})

		Context("No Parameters", func() {
			It("Should count all entries", func() {
				count, err := gorp.NewRetrieve[int, entry]().
					Count(ctx, tx)
				Expect(err).To(Not(HaveOccurred()))
				Expect(count).To(Equal(10)) // Based on the 10 entries created in BeforeEach
			})
		})
	})
})
