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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"strconv"
)

type entry struct {
	ID   int
	Data string
}

func (m entry) GorpKey() int { return m.ID }

func (m entry) SetOptions() []interface{} { return nil }

type prefixEntry struct {
	ID   int
	Data string
}

func (m prefixEntry) GorpKey() []byte { return []byte("prefix-" + strconv.Itoa(m.ID)) }

func (m prefixEntry) SetOptions() []interface{} { return nil }

type entryTwo struct {
	ID   int
	Data string
}

func (m entryTwo) GorpKey() int { return m.ID }

func (m entryTwo) SetOptions() []interface{} { return nil }

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
			for i := 0; i < 10; i++ {
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
			Expect(gorp.NewCreate[int, entry]().Entry(e).GuardExisting(func(e entry) error {
				Expect(e.GorpKey()).To(Equal(42))
				return validate.Error
			}).Exec(ctx, tx)).To(HaveOccurredAs(validate.Error))
		})
		It("Should not call the filter if no entry with a matching GorpKey is found", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			c := 0
			Expect(gorp.NewCreate[int, entry]().Entry(e).GuardExisting(func(e entry) error {
				c++
				return validate.Error
			}).Exec(ctx, tx)).To(Succeed())
			Expect(c).To(Equal(0))
		})
	})

	Describe("Writer", func() {
		It("Should execute operations within a transaction", func() {
			var (
				entries []entry
				keys    []int
			)
			for i := 0; i < 10; i++ {
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
