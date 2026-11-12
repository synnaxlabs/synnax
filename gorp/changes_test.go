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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type unstoredEntry struct{ Key string }

func (e unstoredEntry) GorpKey() string { return e.Key }

type write = func(SpecContext, gorp.Tx) error

func rackLink(k uint32) gorp.Link {
	return gorp.Link{Target: gorp.NewTarget("rack", k), Kind: gorp.Place}
}

func parentLink(k string) gorp.Link {
	return gorp.Link{Target: gorp.NewTarget("device", k), Kind: gorp.Place}
}

func labelLink(k string) gorp.Link {
	return gorp.Link{Target: gorp.NewTarget("label", k), Kind: gorp.Use}
}

var errUndecodable = errors.New("undecodable")

// deviceRefusingCodec fails to decode a refDevice, as a store does with a record that a
// migration missed.
type deviceRefusingCodec struct{ encoding.Codec }

func (c deviceRefusingCodec) Decode(data []byte, value any) error {
	if _, ok := value.(*refDevice); ok {
		return errUndecodable
	}
	return c.Codec.Decode(data, value)
}

var _ = Describe("Changes", func() {
	var (
		db      *gorp.DB
		records *gorp.Table[int32, entry]
		record  = gorp.NewTarget("record", int32(1))
		encode  = func(data string) []byte {
			GinkgoHelper()
			return MustSucceed(
				msgpack.Codec.Encode(entry{ID: 1, Data: data}),
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

	It(
		"Should name a write through an unnamed table by the named table of its type",
		func(ctx SpecContext) {
			unnamed := MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}),
			)
			tx := DeferClose(db.OpenTx())
			Expect(unnamed.NewCreate().Entry(&entry{ID: 1, Data: "a"}).Exec(ctx, tx)).
				To(Succeed())
			Expect(
				gorp.ChangesOf(tx),
			).To(HaveExactElements(HaveField("Record", record)))
			Expect(remove(ctx, tx)).To(Succeed())
			Expect(gorp.ChangesOf(tx)).To(BeEmpty())
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
			device  = refDevice{Key: "d", Rack: 1, Labels: []string{"l1", "l2"}}
			created = []gorp.Link{rackLink(1), labelLink("l1"), labelLink("l2")}
			unlink  = func(ctx SpecContext, tx gorp.Tx) error {
				return devices.NewDelete().
					Where(gorp.MatchKeys[string, refDevice]("d")).
					Exec(ctx, tx)
			}
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
			Expect(racks.NewCreate().
				Entries(&[]refRack{{Key: 1}, {Key: 2}}).
				Exec(ctx, db)).To(Succeed())
			Expect(labels.NewCreate().
				Entries(&[]refLabel{{Key: "l1"}, {Key: "l2"}, {Key: "l3"}}).
				Exec(ctx, db)).To(Succeed())
		})

		DescribeTable(
			"Should add every link of a created record",
			func(ctx SpecContext, w write) {
				tx := DeferClose(db.OpenTx())
				Expect(w(ctx, tx)).To(Succeed())
				Expect(gorp.ChangesOf(tx)).To(HaveExactElements(SatisfyAll(
					HaveField("Kind", gorp.ChangeCreate),
					HaveField("Added", created),
					HaveField("Removed", BeNil()),
					HaveField("Links", created),
				)))
			},
			Entry("through a table", func(ctx SpecContext, tx gorp.Tx) error {
				return devices.NewCreate().Entry(&device).Exec(ctx, tx)
			}),
			Entry("through WrapWriter", func(ctx SpecContext, tx gorp.Tx) error {
				return gorp.WrapWriter[string, refDevice](tx).Set(ctx, device)
			}),
			Entry(
				"through a Create without a table",
				func(ctx SpecContext, tx gorp.Tx) error {
					return gorp.NewCreate[string, refDevice]().Entry(&device).
						Exec(ctx, tx)
				},
			),
		)

		It(
			"Should list a link once when a record holds its target twice",
			func(ctx SpecContext) {
				twice := refDevice{Key: "d", Rack: 1, Labels: []string{"l1", "l1"}}
				tx := DeferClose(db.OpenTx())
				Expect(devices.NewCreate().Entry(&twice).Exec(ctx, tx)).To(Succeed())
				once := []gorp.Link{rackLink(1), labelLink("l1")}
				Expect(gorp.ChangesOf(tx)).To(HaveExactElements(SatisfyAll(
					HaveField("Added", once),
					HaveField("Links", once),
				)))
			},
		)

		DescribeTable(
			"Should record the links that an update adds, removes, and keeps",
			func(
				ctx SpecContext,
				change func(refDevice) refDevice,
				added, removed, links []gorp.Link,
			) {
				Expect(devices.NewCreate().Entry(&device).Exec(ctx, db)).To(Succeed())
				tx := DeferClose(db.OpenTx())
				Expect(devices.NewUpdate().
					Where(gorp.MatchKeys[string, refDevice]("d")).
					Change(func(_ gorp.Context, d refDevice) refDevice {
						return change(d)
					}).
					Exec(ctx, tx)).To(Succeed())
				Expect(gorp.ChangesOf(tx)).To(HaveExactElements(SatisfyAll(
					HaveField("Kind", gorp.ChangeUpdate),
					HaveField("Added", added),
					HaveField("Removed", removed),
					HaveField("Links", links),
				)))
			},
			Entry(
				"a key reference that moves",
				func(d refDevice) refDevice {
					d.Rack = 2
					return d
				},
				[]gorp.Link{rackLink(2)},
				[]gorp.Link{rackLink(1)},
				[]gorp.Link{rackLink(2), labelLink("l1"), labelLink("l2")},
			),
			Entry(
				"an array reference that changes",
				func(d refDevice) refDevice {
					d.Labels = []string{"l2", "l3"}
					return d
				},
				[]gorp.Link{labelLink("l3")},
				[]gorp.Link{labelLink("l1")},
				[]gorp.Link{rackLink(1), labelLink("l2"), labelLink("l3")},
			),
			Entry(
				"no reference",
				func(d refDevice) refDevice { return d },
				[]gorp.Link(nil),
				[]gorp.Link(nil),
				created,
			),
		)

		DescribeTable(
			"Should remove every link of a deleted record",
			func(ctx SpecContext, writes []write) {
				Expect(devices.NewCreate().Entry(&device).Exec(ctx, db)).To(Succeed())
				tx := DeferClose(db.OpenTx())
				for _, w := range writes {
					Expect(w(ctx, tx)).To(Succeed())
				}
				Expect(gorp.ChangesOf(tx)).To(HaveExactElements(SatisfyAll(
					HaveField("Kind", gorp.ChangeDelete),
					HaveField("Added", BeNil()),
					HaveField("Removed", created),
					HaveField("Links", created),
				)))
			},
			Entry("a delete through a table", []write{unlink}),
			Entry("a delete without a table", []write{
				func(ctx SpecContext, tx gorp.Tx) error {
					return gorp.NewDelete[string, refDevice]().
						Where(gorp.MatchKeys[string, refDevice]("d")).
						Exec(ctx, tx)
				},
			}),
			Entry("an update then a delete", []write{
				func(ctx SpecContext, tx gorp.Tx) error {
					moved := refDevice{Key: "d", Rack: 2}
					return devices.NewCreate().Entry(&moved).Exec(ctx, tx)
				},
				unlink,
			}),
		)

		It("Should take the kind of a link from its reference", func(ctx SpecContext) {
			statuses := MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[string, refStatus]{
					DB:         db,
					Name:       "status",
					References: []gorp.Reference[string, refStatus]{newOwnerRef()},
				}),
			)
			tx := DeferClose(db.OpenTx())
			Expect(statuses.NewCreate().
				Entry(&refStatus{Key: "s", Owner: &refOwner{Table: "rack", Key: "1"}}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.ChangesOf(tx)).To(HaveExactElements(
				HaveField("Added", []gorp.Link{rackLink(1)}),
			))
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
					Record:  gorp.NewTarget("device", "a"),
					Kind:    gorp.ChangeDelete,
					Removed: []gorp.Link{rackLink(1)},
					Links:   []gorp.Link{rackLink(1)},
				},
				gorp.Change{
					Record:  gorp.NewTarget("device", "b"),
					Kind:    gorp.ChangeDelete,
					Derived: true,
					Removed: []gorp.Link{rackLink(1), parentLink("a")},
					Links:   []gorp.Link{rackLink(1), parentLink("a")},
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
					Record:  gorp.NewTarget("device", "b"),
					Kind:    gorp.ChangeDelete,
					Removed: []gorp.Link{rackLink(1), parentLink("a")},
					Links:   []gorp.Link{rackLink(1), parentLink("a")},
				}))
			},
		)

		It("Should mark a detach as derived", func(ctx SpecContext) {
			Expect(devices.NewCreate().Entry(&device).Exec(ctx, db)).To(Succeed())
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
					Value:   MustSucceed(msgpack.Codec.Encode(detached)),
					Removed: []gorp.Link{labelLink("l1")},
					Links:   []gorp.Link{rackLink(1), labelLink("l2")},
				},
			))
		})
	})

	DescribeTable(
		"Should read the links before a write from the index or the stored record",
		func(ctx SpecContext, loaded bool) {
			kvs := DeferClose(memkv.New())
			pre := gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
			Expect(gorp.NewCreate[uint32, refRack]().
				Entries(&[]refRack{{Key: 1}, {Key: 2}}).
				Exec(ctx, pre)).To(Succeed())
			Expect(gorp.NewCreate[string, refLabel]().
				Entries(&[]refLabel{{Key: "l1"}, {Key: "l2"}, {Key: "l3"}}).
				Exec(ctx, pre)).To(Succeed())
			Expect(gorp.NewCreate[string, refDevice]().
				Entry(&refDevice{Key: "d", Rack: 1, Labels: []string{"l1", "l2"}}).
				Exec(ctx, pre)).To(Succeed())
			store := kvs
			if !loaded {
				store = &failOnceDB{DB: kvs, err: errors.New("populate failed")}
			}
			linked := gorp.Wrap(store, gorp.WithCodec(msgpack.Codec))
			refs := newDeviceRefs()
			devices := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         linked,
				Name:       "device",
				References: refs.all(),
			}))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   linked,
				Name: "rack",
			}))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
				DB:   linked,
				Name: "label",
			}))
			if loaded {
				Expect(refs.rack.Get(ctx, nil, 1)).To(ConsistOf("d"))
			} else {
				Expect(refs.rack.Get(ctx, nil, 1)).Error().
					To(MatchError(gorp.ErrIndexInvalid))
			}
			tx := DeferClose(linked.OpenTx())
			Expect(devices.NewUpdate().
				Where(gorp.MatchKeys[string, refDevice]("d")).
				Change(func(_ gorp.Context, d refDevice) refDevice {
					d.Rack = 2
					d.Labels = []string{"l2", "l3"}
					return d
				}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.ChangesOf(tx)).To(HaveExactElements(SatisfyAll(
				HaveField("Added", []gorp.Link{rackLink(2), labelLink("l3")}),
				HaveField("Removed", []gorp.Link{rackLink(1), labelLink("l1")}),
			)))
		},
		Entry("when the index has loaded", true),
		Entry("when the index has not loaded", false),
	)

	It(
		"Should fail a write whose stored record cannot be decoded, naming the record",
		func(ctx SpecContext) {
			kvs := DeferClose(memkv.New())
			pre := gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
			Expect(gorp.NewCreate[uint32, refRack]().
				Entry(&refRack{Key: 1}).
				Exec(ctx, pre)).To(Succeed())
			Expect(gorp.NewCreate[string, refDevice]().
				Entry(&refDevice{Key: "d", Rack: 1}).
				Exec(ctx, pre)).To(Succeed())
			linked := gorp.Wrap(
				kvs,
				gorp.WithCodec(deviceRefusingCodec{Codec: msgpack.Codec}),
			)
			refs := newDeviceRefs()
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         linked,
				Name:       "device",
				References: refs.all(),
			}))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   linked,
				Name: "rack",
			}))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
				DB:   linked,
				Name: "label",
			}))
			Expect(refs.rack.Get(ctx, nil, 1)).Error().
				To(MatchError(gorp.ErrIndexInvalid))
			tx := DeferClose(linked.OpenTx())
			Expect(
				gorp.WrapWriter[string, refDevice](tx).Delete(ctx, "d"),
			).To(SatisfyAll(
				MatchError(errUndecodable),
				MatchError("[gorp] - read the links of device d: undecodable"),
			))
			Expect(gorp.ChangesOf(tx)).To(BeEmpty())
		},
	)
})
