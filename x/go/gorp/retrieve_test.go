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
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Retrieve", func() {
	var (
		entries []entry
		tx      gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		tx = db.OpenTx()
		entries = make([]entry, 10)
		for i := range 10 {
			entries[i] = entry{ID: int32(i), Data: "data"}
		}
		Expect(gorp.NewCreate[int32, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("Query State", func() {

		Describe("HasLimit", func() {
			It("Should return false when no limit is set", func() {
				q := gorp.NewRetrieve[int32, entry]()
				Expect(q.HasLimit()).To(BeFalse())
			})
			It("Should return true when a limit is set", func() {
				q := gorp.NewRetrieve[int32, entry]().Limit(10)
				Expect(q.HasLimit()).To(BeTrue())
			})
		})

		Describe("HasOffset", func() {
			It("Should return false when no offset is set", func() {
				q := gorp.NewRetrieve[int32, entry]()
				Expect(q.HasOffset()).To(BeFalse())
			})
			It("Should return true when an offset is set", func() {
				q := gorp.NewRetrieve[int32, entry]().Offset(5)
				Expect(q.HasOffset()).To(BeTrue())
			})
		})

		Describe("HasWhereKeys", func() {
			It("Should return false when no keys are set", func() {
				q := gorp.NewRetrieve[int32, entry]()
				Expect(q.HasWhereKeys()).To(BeFalse())
			})
			It("Should return true when keys are set", func() {
				q := gorp.NewRetrieve[int32, entry]().WhereKeys(1, 2, 3)
				Expect(q.HasWhereKeys()).To(BeTrue())
			})
		})

		Describe("GetWhereKeys", func() {
			It("Should return nil when no keys are set", func() {
				q := gorp.NewRetrieve[int32, entry]()
				Expect(q.GetWhereKeys()).To(BeNil())
			})
			It("Should return the keys when set", func() {
				q := gorp.NewRetrieve[int32, entry]().WhereKeys(1, 2, 3)
				Expect(q.GetWhereKeys()).To(Equal([]int32{1, 2, 3}))
			})
			It("Should accumulate keys from multiple calls", func() {
				q := gorp.NewRetrieve[int32, entry]().WhereKeys(1).WhereKeys(2, 3)
				Expect(q.GetWhereKeys()).To(Equal([]int32{1, 2, 3}))
			})
		})

		Describe("HasFilters", func() {
			It("Should return false when no filters are set", func() {
				q := gorp.NewRetrieve[int32, entry]()
				Expect(q.HasFilters()).To(BeFalse())
			})
			It("Should return true when a filter is set", func() {
				q := gorp.NewRetrieve[int32, entry]().
					Where(gorp.Match[int32, entry](func(_ gorp.Context, _ *entry) (bool, error) { return true, nil }))
				Expect(q.HasFilters()).To(BeTrue())
			})
		})

		Describe("GetEntries", func() {
			It("Should return an empty entries when none are bound", func() {
				q := gorp.NewRetrieve[int32, entry]()
				Expect(q.GetEntries().All()).To(BeEmpty())
			})
			It("Should return bound entries", func() {
				var res []entry
				q := gorp.NewRetrieve[int32, entry]().Entries(&res)
				Expect(q.GetEntries()).ToNot(BeNil())
			})
		})
	})

	Describe("WhereKeys", func() {
		Context("Multiple Entries", func() {
			It("Should retrieve the entry by key", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(Equal([]entry{entries[0]}))
			})
			It("Should return a query.ErrNotFound error if ANY key is not found", func(ctx SpecContext) {
				var res []entry
				err := gorp.NewRetrieve[int32, entry]().
					WhereKeys(entries[0].GorpKey(), 444444).
					Entries(&res).
					Exec(ctx, tx)
				By("Returning the correct error")
				Expect(err).To(MatchError(query.ErrNotFound))
				By("Still retrieving as many entries as possible")
				Expect(res).To(HaveLen(1))
			})
			It("Should still retrieve all possible entries even if some are not found", func(ctx SpecContext) {
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					WhereKeys(44444, entries[0].GorpKey(), entries[1].GorpKey()).
					Entries(&res).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
				Expect(res).To(Equal(entries[:2]))
			})
			Describe("Exists", func() {
				It("Should return true if ALL keys have matching entries", func(ctx SpecContext) {
					Expect(gorp.NewRetrieve[int32, entry]().
						WhereKeys(entries[0].GorpKey(), entries[1].GorpKey()).
						Exists(ctx, tx)).To(BeTrue())
				})
				It("Should return false if ANY key has no matching entry", func(ctx SpecContext) {
					Expect(gorp.NewRetrieve[int32, entry]().
						WhereKeys(entries[0].GorpKey(), 444444).
						Exists(ctx, tx)).To(BeFalse())
				})
			})
		})

		Context("Single Entry", func() {
			It("Should retrieve the entry by key", func(ctx SpecContext) {
				res := &entry{}
				Expect(gorp.NewRetrieve[int32, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entry(res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(Equal(&entries[0]))
			})
			It("Should allow for a nil entry to be provided", func(ctx SpecContext) {
				var res *entry
				Expect(gorp.NewRetrieve[int32, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entry(res).
					Exec(ctx, tx)).To(Succeed())
			})
			It("Should return a query.ErrNotFound error if the key is not found", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					WhereKeys(444444).
					Entry(&entry{}).
					Exec(ctx, tx)).Error().To(MatchError(query.ErrNotFound))
			})
			It("Should return a query.ErrNotFound error if the where clause matches no entry", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 241241, nil })).
					Entry(&entry{}).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			})
			Describe("exists", func() {
				It("Should return true if the key has a matching entry", func(ctx SpecContext) {
					Expect(gorp.NewRetrieve[int32, entry]().
						WhereKeys(entries[0].GorpKey()).
						Exists(ctx, tx)).To(BeTrue())
				})
				It("Should return false if the key has no matching entry", func(ctx SpecContext) {
					Expect(gorp.NewRetrieve[int32, entry]().
						WhereKeys(444444).
						Exists(ctx, tx)).To(BeFalse())
				})
			})
		})
	})

	Describe("WherePrefix", func() {
		Context("With byte-slice keys", func() {
			It("Should retrieve a single entry by exact prefix", func(ctx SpecContext) {
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

			It("Should retrieve multiple entries matching a common prefix", func(ctx SpecContext) {
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

			It("Should return empty results when prefix doesn't match any entries", func(ctx SpecContext) {
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

			It("Should retrieve all entries with common base prefix", func(ctx SpecContext) {
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
	})

	Describe("Where", func() {
		It("Should retrieve the entry by a filter parameter", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == entries[1].ID, nil })).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1]}))
		})
		It("Should AND multiple Where calls together", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID < 5, nil })).
				Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID > 2, nil })).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[3], entries[4]}))
		})
		It("Should support Or combinator", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Or[int32, entry](
					gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == entries[1].ID, nil }),
					gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == entries[2].ID, nil }),
				)).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[2]}))
		})
		It("Should support Not combinator", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Not[int32, entry](
					gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID >= 5, nil }),
				)).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(HaveLen(5))
			for _, e := range res {
				Expect(e.ID).To(BeNumerically("<", 5))
			}
		})
		It("Should support nested And(Or(...), Or(...))", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.And[int32, entry](
					gorp.Or[int32, entry](
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 1, nil }),
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 2, nil }),
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 3, nil }),
					),
					gorp.Or[int32, entry](
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 2, nil }),
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 3, nil }),
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 4, nil }),
					),
				)).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[2], entries[3]}))
		})
		It("Should NOT return a query.NamesNotFound error if no entries are found", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 444444, nil })).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(BeEmpty())
		})
		Describe("exists", func() {
			It("Should return true if ANY entries exist", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.Or[int32, entry](
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == entries[1].ID, nil }),
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 44444, nil }),
					)).
					Exists(ctx, tx)).To(BeTrue())
			})
			It("Should return false if ALL entries do not exist", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.Or[int32, entry](
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 444444, nil }),
						gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID == 44444, nil }),
					)).
					Exists(ctx, tx)).To(BeFalse())
			})
		})
		Describe("Limit", func() {
			It("Should limit the number of entries returned", func(ctx SpecContext) {
				toCreate := 100
				entries := make([]entry, toCreate)
				for i := range toCreate {
					entries[i] = entry{ID: int32(i), Data: "data"}
				}
				Expect(gorp.NewCreate[int32, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Limit(10).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(10))
			})
		})
		Describe("Offset", func() {
			It("Should offset the entries returned", func(ctx SpecContext) {
				toCreate := 100
				entries := make([]entry, toCreate)
				for i := range toCreate {
					entries[i] = entry{ID: int32(i), Data: "data"}
				}
				Expect(gorp.NewCreate[int32, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Offset(10).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(90))
			})
		})
		Describe("Limit + Offset", func() {
			It("Should limit and offset the entries returned", func(ctx SpecContext) {
				toCreate := 100
				entries := make([]entry, toCreate)
				for i := range toCreate {
					entries[i] = entry{ID: int32(i), Data: "data"}
				}
				Expect(gorp.NewCreate[int32, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).
					Limit(10).
					Offset(10).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(10))
			})
		})
	})

	Describe("WhereRaw", func() {
		It("Should filter entries by their raw bytes before decoding", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				WhereRaw(func(data []byte) (bool, error) {
					return bytes.Contains(data, []byte("data")), nil
				}).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(HaveLen(10))
		})

		It("Should skip entries that do not match the raw filter", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				WhereRaw(func(data []byte) (bool, error) {
					return false, nil
				}).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(BeEmpty())
		})

		It("Should return an error if the filter encounters an error", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				WhereRaw(func(data []byte) (bool, error) {
					return true, errors.New("cat")
				}).
				Exec(ctx, tx),
			).To(MatchError(ContainSubstring("cat")))
			Expect(res).To(BeEmpty())
		})
	})

	Describe("No Parameters", func() {
		It("Should return all entries for the given type", func(ctx SpecContext) {
			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal(entries))
		})
	})
	Describe("GetWhereKeys", func() {
		It("Should return keys when WhereKeys has been called", func() {
			q := gorp.NewRetrieve[int32, entry]().WhereKeys(1, 2, 3)
			keys := q.GetWhereKeys()
			Expect(keys).To(Equal([]int32{1, 2, 3}))
		})
		It("Should return false when WhereKeys has not been called", func() {
			q := gorp.NewRetrieve[int32, entry]()
			Expect(q.HasWhereKeys()).To(BeFalse())
		})
		It("Should accumulate keys across multiple WhereKeys calls", func() {
			q := gorp.NewRetrieve[int32, entry]().WhereKeys(1, 2).WhereKeys(3, 4)
			keys := q.GetWhereKeys()
			Expect(keys).To(Equal([]int32{1, 2, 3, 4}))
		})
	})
	Describe("HasFilters", func() {
		It("Should return true when Where has been called", func() {
			q := gorp.NewRetrieve[int32, entry]().Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) {
				return e.ID == 1, nil
			}))
			Expect(q.HasFilters()).To(BeTrue())
		})
		It("Should return false when Where has not been called", func() {
			q := gorp.NewRetrieve[int32, entry]().WhereKeys(1, 2, 3)
			Expect(q.HasFilters()).To(BeFalse())
		})
		It("Should return false for a fresh query", func() {
			q := gorp.NewRetrieve[int32, entry]()
			Expect(q.HasFilters()).To(BeFalse())
		})
	})
	Describe("Count", func() {
		Context("WhereKeys", func() {
			It("Should return the count of existing keys", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					WhereKeys(entries[0].GorpKey(), entries[1].GorpKey()).
					Count(ctx, tx)).To(Equal(2))
			})

			It("Should handle non-existent keys", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					WhereKeys(entries[0].GorpKey(), 444444).
					Count(ctx, tx)).To(Equal(1))
			})
		})

		Context("Where", func() {
			It("Should count entries matching a filter", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID < 5, nil })).
					Count(ctx, tx)).To(Equal(5))
			})

			It("Should return zero for non-matching filters", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID > 100, nil })).
					Count(ctx, tx)).To(Equal(0))
			})

			It("Should AND multiple Where calls", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID < 5, nil })).
					Where(gorp.Match[int32, entry](func(_ gorp.Context, e *entry) (bool, error) { return e.ID > 2, nil })).
					Count(ctx, tx)).To(Equal(2))
			})

			It("Should pass the correct transaction in the Context of the Where clause", func(ctx SpecContext) {
				callCount := 0
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.Match[int32, entry](func(gCtx gorp.Context, _ *entry) (bool, error) {
						callCount++
						Expect(gCtx.Context).To(BeIdenticalTo(ctx))
						Expect(gCtx.Tx).To(BeIdenticalTo(tx))
						return false, nil
					})).Count(ctx, tx)).To(Equal(0))
				Expect(callCount).To(BeNumerically(">", 0))
			})
		})

		Context("WherePrefix", func() {
			var (
				r1 = prefixEntry{ID: 123, Data: "data"}
				r2 = prefixEntry{ID: 456, Data: "data"}
			)
			BeforeEach(func(ctx SpecContext) {
				Expect(gorp.NewCreate[[]byte, prefixEntry]().
					Entry(&r1).
					Exec(ctx, tx)).
					To(Succeed())
				Expect(gorp.NewCreate[[]byte, prefixEntry]().
					Entry(&r2).
					Exec(ctx, tx)).
					To(Succeed())
			})

			It("Should count entries matching a prefix", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[[]byte, prefixEntry]().
					WherePrefix([]byte("prefix-123")).
					Count(ctx, tx)).To(Equal(1))
			})

			It("Should return zero for non-matching prefix", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[[]byte, prefixEntry]().
					WherePrefix([]byte("nonexistent-prefix")).
					Count(ctx, tx)).To(Equal(0))
			})

			It("Should work in combination with WhereKeys", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[[]byte, prefixEntry]().
					WhereKeys(r1.GorpKey(), r2.GorpKey()).
					WherePrefix([]byte("prefix-123")).
					Count(ctx, tx)).To(Equal(1))
			})
		})

		Context("No Parameters", func() {
			It("Should count all entries", func(ctx SpecContext) {
				Expect(gorp.NewRetrieve[int32, entry]().Count(ctx, tx)).To(Equal(10))
			})
		})
	})
})
