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
)

type entry struct {
	ID   int
	Data string
}

func (m entry) GorpKey() int { return m.ID }

func (m entry) SetOptions() []interface{} { return nil }

var _ = Describe("Create", Ordered, func() {
	var (
		db   *gorp.DB
		kvDB kv.DB
		txn  gorp.WriteTxn
	)
	BeforeAll(func() {
		kvDB = memkv.New()
		db = gorp.Wrap(kvDB)
	})
	AfterAll(func() { Expect(kvDB.Close()).To(Succeed()) })
	BeforeEach(func() { txn = db.BeginWrite(ctx) })
	AfterEach(func() { Expect(txn.Close()).To(Succeed()) })
	Context("Single entry", func() {
		It("Should create the entry in the db", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			Expect(gorp.NewCreate[int, entry]().Entry(e).Exec(txn)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(42).Exists(txn)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})
	Describe("Multiple entries", func() {
		It("Should create the entries in the db", func() {
			var entries []entry
			for i := 0; i < 10; i++ {
				entries = append(entries, entry{ID: i, Data: "data"})
			}
			Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(txn)).To(Succeed())
			var keys []int
			for _, e := range entries {
				keys = append(keys, e.ID)
			}
			exists, err := gorp.NewRetrieve[int, entry]().
				WhereKeys(keys...).Exists(txn)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
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
			Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(txn)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(keys...).Exists(txn)
			Expect(err).To(BeNil())
			Expect(exists).To(BeFalse())
			Expect(txn.Commit()).To(Succeed())
			exists, err = gorp.NewRetrieve[int, entry]().WhereKeys(keys...).Exists(txn)
			Expect(err).To(BeNil())
			Expect(exists).To(BeTrue())
		})
	})
})
