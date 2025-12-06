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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

type entry struct {
	ID   int32
	Data string
}

func (m entry) GorpKey() int32 { return m.ID }

func (m entry) SetOptions() []any { return nil }

type prefixEntry struct {
	ID   int32
	Data string
}

func (m prefixEntry) GorpKey() []byte { return []byte("prefix-" + strconv.Itoa(int(m.ID))) }

func (m prefixEntry) SetOptions() []any { return nil }

type entryTwo struct {
	ID   int32
	Data string
}

func (m entryTwo) GorpKey() int32 { return m.ID }

func (m entryTwo) SetOptions() []any { return nil }

var _ = Describe("Create", Ordered, func() {
	var (
		ctx context.Context
		db  *gorp.DB
		tx  gorp.Tx
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	BeforeEach(func() {
		ctx = context.Background()
		tx = db.OpenTx()
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
	Context("Single entry", func() {
		It("Should create the entry in the db", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			Expect(gorp.NewCreate[int32, entry]().Entry(e).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(42).Exists(ctx, tx)).To(BeTrue())
		})
	})

	Context("Multiple entries", func() {
		It("Should create the entries in the db", func() {
			e := make([]entry, 10)
			for i := range 10 {
				e[i] = entry{ID: int32(i), Data: "data"}
			}
			Expect(gorp.NewCreate[int32, entry]().Entries(&e).Exec(ctx, tx)).To(Succeed())
			keys := make([]int32, 10)
			for i, e := range e {
				keys[i] = e.ID
			}
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(keys...).
				Exists(ctx, tx)).To(BeTrue())
		})
	})

	Describe("Guard", func() {
		It("Should prevent the accidental override of existing entries", func() {
			e := &entry{
				ID:   int32(42),
				Data: "The answer to life, the universe, and everything",
			}
			Expect(gorp.NewCreate[int32, entry]().Entry(e).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewCreate[int32, entry]().Entry(e).MergeExisting(func(_ gorp.Context, c, e entry) (entry, error) {
				Expect(e.GorpKey()).To(Equal(int32(42)))
				return entry{}, validate.Error
			}).Exec(ctx, tx)).To(HaveOccurredAs(validate.Error))
		})
		It("Should not call the filter if no entry with a matching GorpKey is found", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			c := 0
			Expect(gorp.NewCreate[int32, entry]().Entry(e).MergeExisting(func(_ gorp.Context, creating, _ entry) (entry, error) {
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
			Expect(gorp.NewCreate[int32, entry]().Entry(e).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewCreate[int32, entry]().Entry(e).MergeExisting(func(_ gorp.Context, _, e entry) (entry, error) {
				Expect(e.GorpKey()).To(Equal(int32(42)))
				return entry{ID: e.ID, Data: e.Data + "!"}, nil
			}).Exec(ctx, tx)).To(Succeed())
			var e2 entry
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(int32(42)).Entry(&e2).Exec(ctx, tx)).To(Succeed())
			Expect(e2.Data).To(Equal("The answer to life, the universe, and everything!"))
		})
	})

	Describe("Writer", func() {
		It("Should execute operations within a transaction", func() {
			entries := make([]entry, 10)
			keys := make([]int32, 10)
			for i := range 10 {
				entries[i] = entry{ID: int32(i), Data: "data"}
				keys[i] = int32(i)
			}
			Expect(gorp.NewCreate[int32, entry]().Entries(&entries).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(keys...).Exists(ctx, db)).To(BeFalse())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(gorp.NewRetrieve[int32, entry]().WhereKeys(keys...).Exists(ctx, tx)).To(BeTrue())
		})
	})
})
