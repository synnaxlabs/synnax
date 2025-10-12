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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

type entry struct {
	ID   int
	Data string
}

func (m entry) GorpKey() int { return m.ID }

func (m entry) SetOptions() []any { return nil }

type prefixEntry struct {
	ID   int
	Data string
}

func (m prefixEntry) GorpKey() []byte { return []byte("prefix-" + strconv.Itoa(m.ID)) }

func (m prefixEntry) SetOptions() []any { return nil }

type entryTwo struct {
	ID   int
	Data string
}

func (m entryTwo) GorpKey() int { return m.ID }

func (m entryTwo) SetOptions() []any { return nil }

var _ = Describe("Create", Ordered, func() {
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
	Context("Single entry", func() {
		It("Should create the entry in the db", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			Expect(gorp.NewCreate[int, entry]().Entry(e).Exec(ctx, tx)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(42).Exists(ctx, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})

	Context("Multiple entries", func() {
		It("Should create the entries in the db", func() {
			var e []entry
			for i := range 10 {
				e = append(e, entry{ID: i, Data: "data"})
			}
			Expect(gorp.NewCreate[int, entry]().Entries(&e).Exec(ctx, tx)).To(Succeed())
			var keys []int
			for _, e := range e {
				keys = append(keys, e.ID)
			}
			exists, err := gorp.NewRetrieve[int, entry]().
				WhereKeys(keys...).Exists(ctx, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})

	Describe("Guard", func() {
		It("Should prevent the accidental override of existing entries", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			Expect(gorp.NewCreate[int, entry]().Entry(e).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewCreate[int, entry]().Entry(e).MergeExisting(func(_ gorp.Context, c, e entry) (entry, error) {
				Expect(e.GorpKey()).To(Equal(42))
				return entry{}, validate.Error
			}).Exec(ctx, tx)).To(HaveOccurredAs(validate.Error))
		})
		It("Should not call the filter if no entry with a matching GorpKey is found", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			c := 0
			Expect(gorp.NewCreate[int, entry]().Entry(e).MergeExisting(func(_ gorp.Context, creating, _ entry) (entry, error) {
				c++
				return creating, validate.Error
			}).Exec(ctx, tx)).To(Succeed())
			Expect(c).To(Equal(0))
		})
		It("Should merge an existing entry with the new entry", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			Expect(gorp.NewCreate[int, entry]().Entry(e).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewCreate[int, entry]().Entry(e).MergeExisting(func(_ gorp.Context, c entry, e entry) (entry, error) {
				Expect(e.GorpKey()).To(Equal(42))
				return entry{ID: e.ID, Data: e.Data + "!"}, nil
			}).Exec(ctx, tx)).To(Succeed())
			var e2 entry
			Expect(gorp.NewRetrieve[int, entry]().WhereKeys(42).Entry(&e2).Exec(ctx, tx)).To(Succeed())
			Expect(e2.Data).To(Equal("The answer to life, the universe, and everything!"))
		})
	})

	Describe("Writer", func() {
		It("Should execute operations within a transaction", func() {
			var (
				entries []entry
				keys    []int
			)
			for i := range 10 {
				entries = append(entries, entry{ID: i, Data: "data"})
				keys = append(keys, i)
			}
			Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(keys...).Exists(ctx, db)
			Expect(err).To(BeNil())
			Expect(exists).To(BeFalse())
			Expect(tx.Commit(ctx)).To(Succeed())
			exists, err = gorp.NewRetrieve[int, entry]().WhereKeys(keys...).Exists(ctx, tx)
			Expect(err).To(BeNil())
			Expect(exists).To(BeTrue())
		})
	})
})
