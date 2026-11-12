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
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/gorp/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Observe", func() {
	var (
		db         *gorp.DB
		entryTable *gorp.Table[int32, entry]
		grapeTable *gorp.Table[int32, grape]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(OpenGorpMsgpackDB())
		entryTable = MustOpen(
			gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}),
		)
		grapeTable = MustOpen(
			gorp.OpenTable(ctx, gorp.TableConfig[int32, grape]{DB: db}),
		)
	})
	It(
		"Should correctly observe a change to the key value store",
		func(ctx SpecContext) {
			tx := db.OpenTx()
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 42, Data: "data"}).
				Exec(ctx, tx)).To(Succeed())
			called := false
			entryTable.Observe().
				OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
					for ch := range r {
						Expect(ch.Value).To(Equal(entry{ID: 42, Data: "data"}))
						Expect(ch.Kind).To(Equal(gorp.ChangeCreate))
						called = true
					}
				})
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(called).To(BeTrue())
		},
	)

	It(
		"Should not notify for a different type than the entries written",
		func(ctx SpecContext) {
			tx := db.OpenTx()
			Expect(
				gorp.NewCreate[int32, entry]().Entry(&entry{ID: 42, Data: "data"}).
					Exec(ctx, tx),
			).To(Succeed())
			called := false
			grapeTable.Observe().
				OnChange(func(ctx context.Context, r gorp.TxReader[int32, grape]) {
					called = true
				})
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(called).To(BeFalse())
		},
	)

	It(
		"Should notify each observer with only their matching entries in a mixed transaction",
		func(ctx SpecContext) {
			tx := db.OpenTx()
			Expect(
				gorp.NewCreate[int32, entry]().Entry(&entry{ID: 1, Data: "one"}).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(
				gorp.NewCreate[int32, entry]().Entry(&entry{ID: 2, Data: "two"}).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(
				gorp.NewCreate[int32, grape]().Entry(&grape{ID: 100, Data: "hundred"}).
					Exec(ctx, tx),
			).To(Succeed())

			var (
				entryChanges []gorp.EntryChange[int32, entry]
				grapeChanges []gorp.EntryChange[int32, grape]
			)

			entryTable.Observe().
				OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
					for ch := range r {
						entryChanges = append(entryChanges, ch)
					}
				})
			grapeTable.Observe().
				OnChange(func(ctx context.Context, r gorp.TxReader[int32, grape]) {
					for ch := range r {
						grapeChanges = append(grapeChanges, ch)
					}
				})

			Expect(tx.Commit(ctx)).To(Succeed())

			Expect(entryChanges).To(HaveLen(2))
			Expect(entryChanges[0].Value).To(Equal(entry{ID: 1, Data: "one"}))
			Expect(entryChanges[1].Value).To(Equal(entry{ID: 2, Data: "two"}))

			Expect(grapeChanges).To(HaveLen(1))
			Expect(grapeChanges[0].Value).To(Equal(grape{ID: 100, Data: "hundred"}))
		},
	)

	It(
		"Should correctly decode the key on delete notifications",
		func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 42, Data: "data"}).
				Exec(ctx, db)).To(Succeed())

			tx := db.OpenTx()
			Expect(
				gorp.NewDelete[int32, entry]().Where(gorp.MatchKeys[int32, entry](42)).
					Exec(ctx, tx),
			).To(Succeed())

			var deleteChanges []gorp.EntryChange[int32, entry]
			entryTable.Observe().
				OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
					for ch := range r {
						deleteChanges = append(deleteChanges, ch)
					}
				})

			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(deleteChanges).To(HaveLen(1))
			Expect(deleteChanges[0].Kind).To(Equal(gorp.ChangeDelete))
			Expect(deleteChanges[0].Key).To(Equal(int32(42)))
		},
	)
})

var _ = Describe("Changes", func() {
	var (
		db         *gorp.DB
		entryTable *gorp.Table[int32, entry]
		commits    []gorp.Commit
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(OpenGorpMsgpackDB())
		entryTable = MustOpen(
			gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}),
		)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, grape]{DB: db}))
		commits = nil
		DeferCleanup(db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
			commits = append(commits, c)
		}))
	})

	It(
		"Should decode the sets and deletes of the table in the order of the commit",
		func(ctx SpecContext) {
			Expect(entryTable.NewCreate().
				Entry(&entry{ID: 1, Data: "one"}).
				Exec(ctx, db)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := entryTable.NewCreate().
					Entry(&entry{ID: 2, Data: "two"}).
					Exec(ctx, tx); err != nil {
					return err
				}
				if err := gorp.NewCreate[int32, grape]().
					Entry(&grape{ID: 3, Data: "three"}).
					Exec(ctx, tx); err != nil {
					return err
				}
				return entryTable.NewDelete().
					Where(gorp.MatchKeys[int32, entry](1)).
					Exec(ctx, tx)
			})).To(Succeed())
			Expect(commits).To(HaveLen(2))
			Expect(slices.Collect(entryTable.Changes(ctx, commits[1]))).To(Equal(
				[]gorp.EntryChange[int32, entry]{
					{
						Key:   2,
						Value: entry{ID: 2, Data: "two"},
						Kind:  gorp.ChangeCreate,
					},
					{Key: 1, Kind: gorp.ChangeDelete},
				},
			))
		},
	)

	It("Should mark a change to an existing entry as an update", func(ctx SpecContext) {
		Expect(entryTable.NewCreate().
			Entry(&entry{ID: 1, Data: "one"}).
			Exec(ctx, db)).To(Succeed())
		Expect(entryTable.NewUpdate().
			Where(gorp.MatchKeys[int32, entry](1)).
			Change(func(_ gorp.Context, e entry) entry {
				e.Data = "uno"
				return e
			}).
			Exec(ctx, db)).To(Succeed())
		Expect(commits).To(HaveLen(2))
		Expect(slices.Collect(entryTable.Changes(ctx, commits[1]))).To(Equal(
			[]gorp.EntryChange[int32, entry]{
				{Key: 1, Value: entry{ID: 1, Data: "uno"}, Kind: gorp.ChangeUpdate},
			},
		))
	})

	It("Should mark a cascaded delete as derived", func(ctx SpecContext) {
		racks := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
			DB:   db,
			Name: "rack",
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
			DB:   db,
			Name: "label",
		}))
		devices := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
			DB:         db,
			Name:       "device",
			References: newDeviceRefs().all(),
		}))
		Expect(racks.NewCreate().Entry(&refRack{Key: 1}).Exec(ctx, db)).To(Succeed())
		Expect(devices.NewCreate().Entries(&[]refDevice{
			{Key: "a", Rack: 1},
			{Key: "b", Rack: 1, Parent: "a"},
		}).Exec(ctx, db)).To(Succeed())
		Expect(devices.NewDelete().
			Where(gorp.MatchKeys[string, refDevice]("a")).
			Exec(ctx, db)).To(Succeed())
		Expect(commits).To(HaveLen(3))
		Expect(slices.Collect(devices.Changes(ctx, commits[2]))).To(ConsistOf(
			gorp.EntryChange[string, refDevice]{Key: "a", Kind: gorp.ChangeDelete},
			gorp.EntryChange[string, refDevice]{
				Key:     "b",
				Kind:    gorp.ChangeDelete,
				Derived: true,
			},
		))
	})

	It("Should return no changes for a commit to another table", func(ctx SpecContext) {
		Expect(gorp.NewCreate[int32, grape]().
			Entry(&grape{ID: 3, Data: "three"}).
			Exec(ctx, db)).To(Succeed())
		Expect(commits).To(HaveLen(1))
		Expect(slices.Collect(entryTable.Changes(ctx, commits[0]))).To(BeEmpty())
	})
})
