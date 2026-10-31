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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/gorp/testutil"
	. "github.com/synnaxlabs/x/testutil"
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

func (m prefixEntry) GorpKey() string { return "prefix-" + strconv.Itoa(int(m.ID)) }

func (m prefixEntry) SetOptions() []any { return nil }

type grape struct {
	ID   int32
	Data string
}

func (m grape) GorpKey() int32 { return m.ID }

func (m grape) SetOptions() []any { return nil }

var _ = Describe("Create", Ordered, func() {
	var (
		db *gorp.DB
		tx gorp.Tx
	)
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		db = DeferClose(OpenGorpMsgpackDB())
	})
	BeforeEach(func() {
		tx = DeferClose(db.OpenTx())
	})
	Context("Single entry", func() {
		It("Should create the entry in the db", func(ctx SpecContext) {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			Expect(gorp.NewCreate[int32, entry]().Entry(e).Exec(ctx, tx)).To(Succeed())
			Expect(
				gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](42)).
					Exists(ctx, tx),
			).To(BeTrue())
		})
	})

	Context("Multiple entries", func() {
		It("Should create the entries in the db", func(ctx SpecContext) {
			e := make([]entry, 10)
			for i := range 10 {
				e[i] = entry{ID: int32(i), Data: "data"}
			}
			Expect(
				gorp.NewCreate[int32, entry]().Entries(&e).Exec(ctx, tx),
			).To(Succeed())
			keys := make([]int32, 10)
			for i, e := range e {
				keys[i] = e.ID
			}
			Expect(
				gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](keys...)).
					Exists(ctx, tx),
			).To(BeTrue())
		})
	})

	Describe("Writer", func() {
		It("Should execute operations within a transaction", func(ctx SpecContext) {
			entries := make([]entry, 10)
			keys := make([]int32, 10)
			for i := range 10 {
				entries[i] = entry{ID: int32(i), Data: "data"}
				keys[i] = int32(i)
			}
			Expect(
				gorp.NewCreate[int32, entry]().Entries(&entries).Exec(ctx, tx),
			).To(Succeed())
			Expect(
				gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](keys...)).
					Exists(ctx, db),
			).To(BeFalse())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(
				gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](keys...)).
					Exists(ctx, tx),
			).To(BeTrue())
		})
	})
})
