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
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

type entry struct {
	ID   int32
	Data string
}

func (m entry) GorpKey() int32 { return m.ID }

type prefixEntry struct {
	ID   int32
	Data string
}

func (m prefixEntry) GorpKey() string { return "prefix-" + strconv.Itoa(int(m.ID)) }

type grape struct {
	ID   int32
	Data string
}

func (m grape) GorpKey() int32 { return m.ID }

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

	Describe("Exclusive", func() {
		create := func(ctx SpecContext, data string) error {
			return gorp.NewCreate[int32, entry]().
				Exclusive().
				Entry(&entry{ID: 7, Data: data}).
				Exec(ctx, tx)
		}
		stored := func(ctx SpecContext) entry {
			GinkgoHelper()
			var e entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](7)).
				Entry(&e).
				Exec(ctx, tx)).To(Succeed())
			return e
		}
		exists := SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("%s 7 already exists", types.Name[entry]())),
		)
		It("Should create an entry whose key does not exist", func(ctx SpecContext) {
			Expect(create(ctx, "first")).To(Succeed())
			Expect(stored(ctx).Data).To(Equal("first"))
		})
		It("Should refuse an entry that a committed write created", func(
			ctx SpecContext,
		) {
			Expect(create(ctx, "first")).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			next := DeferClose(db.OpenTx())
			Expect(gorp.NewCreate[int32, entry]().
				Exclusive().
				Entry(&entry{ID: 7, Data: "second"}).
				Exec(ctx, next)).To(exists)
			Expect(next.Close()).To(Succeed())
			Expect(gorp.NewDelete[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](7)).
				Exec(ctx, db)).To(Succeed())
		})
		It("Should refuse an entry that an earlier write in the tx created", func(
			ctx SpecContext,
		) {
			Expect(create(ctx, "first")).To(Succeed())
			Expect(create(ctx, "second")).To(exists)
			Expect(stored(ctx).Data).To(Equal("first"))
		})
		It(
			"Should create an entry that the transaction deleted",
			func(ctx SpecContext) {
				Expect(create(ctx, "first")).To(Succeed())
				Expect(gorp.NewDelete[int32, entry]().
					Where(gorp.MatchKeys[int32, entry](7)).
					Exec(ctx, tx)).To(Succeed())
				Expect(create(ctx, "second")).To(Succeed())
				Expect(stored(ctx).Data).To(Equal("second"))
			},
		)
		It("Should overwrite an entry without the option", func(ctx SpecContext) {
			Expect(create(ctx, "first")).To(Succeed())
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 7, Data: "second"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(stored(ctx).Data).To(Equal("second"))
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
