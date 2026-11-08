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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type unstoredEntry struct{ Key string }

func (e unstoredEntry) GorpKey() string   { return e.Key }
func (e unstoredEntry) SetOptions() []any { return nil }

type write = func(SpecContext, gorp.Tx) error

var _ = Describe("Changes", func() {
	var (
		db      *gorp.DB
		records *gorp.Table[int32, entry]
		record  = gorp.NewTarget("record", int32(1))
		encode  = func(data string) []byte {
			return MustSucceed(
				msgpack.Codec.Encode(context.Background(), entry{ID: 1, Data: data}),
			)
		}
		create = func(data string) write {
			return func(ctx SpecContext, tx gorp.Tx) error {
				return records.NewCreate().
					Entry(&entry{ID: 1, Data: data}).
					Exec(ctx, tx)
			}
		}
		update = func(data string) write {
			return func(ctx SpecContext, tx gorp.Tx) error {
				return records.NewUpdate().
					Where(gorp.MatchKeys[int32, entry](1)).
					Change(func(_ gorp.Context, e entry) entry {
						e.Data = data
						return e
					}).
					Exec(ctx, tx)
			}
		}
		remove = func(ctx SpecContext, tx gorp.Tx) error {
			return records.NewDelete().
				Where(gorp.MatchKeys[int32, entry](1)).
				Exec(ctx, tx)
		}
		wrapped = func(data string) write {
			return func(ctx SpecContext, tx gorp.Tx) error {
				w := gorp.WrapWriter[int32, entry](tx)
				return w.Set(ctx, entry{ID: 1, Data: data})
			}
		}
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		records = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
			DB:   db,
			Name: "record",
		}))
	})

	DescribeTable(
		"Should merge the writes to a record into one change",
		func(ctx SpecContext, existing bool, writes []write, want gorp.Changes) {
			if existing {
				Expect(create("a")(ctx, db)).To(Succeed())
			}
			tx := DeferClose(db.OpenTx())
			for _, w := range writes {
				Expect(w(ctx, tx)).To(Succeed())
			}
			Expect(gorp.ChangesOf(tx)).To(Equal(want))
		},
		Entry("a create", false, []write{create("b")}, gorp.Changes{
			{Record: record, Kind: gorp.ChangeCreate, Value: encode("b")},
		}),
		Entry(
			"a create over an existing record",
			true,
			[]write{create("b")},
			gorp.Changes{
				{Record: record, Kind: gorp.ChangeUpdate, Value: encode("b")},
			},
		),
		Entry("an update", true, []write{update("b")}, gorp.Changes{
			{Record: record, Kind: gorp.ChangeUpdate, Value: encode("b")},
		}),
		Entry("a delete", true, []write{remove}, gorp.Changes{
			{Record: record, Kind: gorp.ChangeDelete},
		}),
		Entry(
			"a create then a delete",
			false,
			[]write{create("b"), remove},
			gorp.Changes(nil),
		),
		Entry(
			"a delete then a create",
			true,
			[]write{remove, create("b")},
			gorp.Changes{
				{Record: record, Kind: gorp.ChangeUpdate, Value: encode("b")},
			},
		),
		Entry("a set through WrapWriter", false, []write{wrapped("b")}, gorp.Changes{
			{Record: record, Kind: gorp.ChangeCreate, Value: encode("b")},
		}),
	)

	It("Should list records in the order of their first write", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		Expect(records.NewCreate().
			Entries(&[]entry{{ID: 2, Data: "a"}, {ID: 1, Data: "a"}}).
			Exec(ctx, tx)).To(Succeed())
		Expect(records.NewCreate().Entry(&entry{ID: 2, Data: "b"}).Exec(ctx, tx)).
			To(Succeed())
		Expect(gorp.ChangesOf(tx)).To(HaveExactElements(
			HaveField("Record", gorp.NewTarget("record", int32(2))),
			HaveField("Record", gorp.NewTarget("record", int32(1))),
		))
	})

	It(
		"Should name the table of an entry type that no table stores by the type",
		func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			Expect(gorp.WrapWriter[string, unstoredEntry](tx).
				Set(ctx, unstoredEntry{Key: "u"})).To(Succeed())
			Expect(gorp.ChangesOf(tx)).To(HaveExactElements(
				HaveField("Record", gorp.NewTarget("unstoredEntry", "u")),
			))
		},
	)

	It("Should not record a raw key-value write", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		Expect(tx.Set(ctx, []byte("raw"), []byte("value"))).To(Succeed())
		Expect(gorp.ChangesOf(tx)).To(BeEmpty())
	})

	It("Should panic for a DB", func() {
		Expect(func() { gorp.ChangesOf(db) }).
			To(PanicWith("[gorp] - ChangesOf needs a transaction, not a DB"))
	})

	Describe("References", func() {
		var (
			labels  *gorp.Table[string, refLabel]
			devices *gorp.Table[string, refDevice]
		)

		BeforeEach(func(ctx SpecContext) {
			racks := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "rack",
			}))
			labels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
				DB:   db,
				Name: "label",
			}))
			devices = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         db,
				Name:       "device",
				References: newDeviceRefs().all(),
			}))
			Expect(racks.NewCreate().Entry(&refRack{Key: 1}).Exec(ctx, db)).
				To(Succeed())
			Expect(labels.NewCreate().
				Entries(&[]refLabel{{Key: "l1"}, {Key: "l2"}}).
				Exec(ctx, db)).To(Succeed())
		})

		It("Should mark a cascaded delete as derived", func(ctx SpecContext) {
			Expect(devices.NewCreate().Entries(&[]refDevice{
				{Key: "a", Rack: 1},
				{Key: "b", Rack: 1, Parent: "a"},
			}).Exec(ctx, db)).To(Succeed())
			tx := DeferClose(db.OpenTx())
			Expect(devices.NewDelete().
				Where(gorp.MatchKeys[string, refDevice]("a")).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.ChangesOf(tx)).To(ConsistOf(
				gorp.Change{
					Record: gorp.NewTarget("device", "a"),
					Kind:   gorp.ChangeDelete,
				},
				gorp.Change{
					Record:  gorp.NewTarget("device", "b"),
					Kind:    gorp.ChangeDelete,
					Derived: true,
				},
			))
		})

		It(
			"Should not mark a record that the caller also writes as derived",
			func(ctx SpecContext) {
				Expect(devices.NewCreate().Entries(&[]refDevice{
					{Key: "a", Rack: 1},
					{Key: "b", Rack: 1, Parent: "a"},
				}).Exec(ctx, db)).To(Succeed())
				tx := DeferClose(db.OpenTx())
				Expect(devices.NewCreate().
					Entry(&refDevice{Key: "b", Rack: 1, Parent: "a"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(devices.NewDelete().
					Where(gorp.MatchKeys[string, refDevice]("a")).
					Exec(ctx, tx)).To(Succeed())
				Expect(gorp.ChangesOf(tx)).To(ContainElement(gorp.Change{
					Record: gorp.NewTarget("device", "b"),
					Kind:   gorp.ChangeDelete,
				}))
			},
		)

		It("Should mark a detach as derived", func(ctx SpecContext) {
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "d", Rack: 1, Labels: []string{"l1", "l2"}}).
				Exec(ctx, db)).To(Succeed())
			tx := DeferClose(db.OpenTx())
			Expect(labels.NewDelete().
				Where(gorp.MatchKeys[string, refLabel]("l1")).
				Exec(ctx, tx)).To(Succeed())
			detached := refDevice{Key: "d", Rack: 1, Labels: []string{"l2"}}
			Expect(gorp.ChangesOf(tx)).To(ConsistOf(
				gorp.Change{
					Record: gorp.NewTarget("label", "l1"),
					Kind:   gorp.ChangeDelete,
				},
				gorp.Change{
					Record:  gorp.NewTarget("device", "d"),
					Kind:    gorp.ChangeUpdate,
					Derived: true,
					Value:   MustSucceed(msgpack.Codec.Encode(ctx, detached)),
				},
			))
		})
	})
})
