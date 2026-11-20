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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

type refRack struct{ Key uint32 }

func (r refRack) GorpKey() uint32 { return r.Key }

type refLabel struct{ Key string }

func (l refLabel) GorpKey() string { return l.Key }

type refDevice struct {
	Key    string
	Rack   uint32
	Parent string
	Labels []string
}

func (d refDevice) GorpKey() string { return d.Key }

// refDeviceName reads the table of refDevice as another entry type.
type refDeviceName struct{ Key string }

func (d refDeviceName) GorpKey() string      { return d.Key }
func (refDeviceName) CustomTypeName() string { return types.Name[refDevice]() }

type refOwner struct {
	Table string
	Key   string
}

type refStatus struct {
	Key   string
	Owner *refOwner
}

func (s refStatus) GorpKey() string { return s.Key }

type refProject struct{ Key string }

func (p refProject) GorpKey() string { return p.Key }

type refGroup struct {
	Key     string
	Project string
}

func (g refGroup) GorpKey() string { return g.Key }

type refSwatch struct {
	Key   string
	Group string
	Value string
}

func (s refSwatch) GorpKey() string { return s.Key }

// refFill holds a swatch by key, or a value once the swatch is gone.
type refFill struct {
	Key    string
	Group  string
	Swatch string
	Value  string
}

func (f refFill) GorpKey() string { return f.Key }

type refSchematic struct {
	Key     string
	Project string
	Group   string
}

func (s refSchematic) GorpKey() string { return s.Key }

// deviceRefs are the references of a refDevice, in the shape that Oracle generates.
type deviceRefs struct {
	rack   *gorp.KeyReference[string, refDevice, uint32]
	parent *gorp.KeyReference[string, refDevice, string]
	labels *gorp.KeyReference[string, refDevice, string]
}

func newDeviceRefs() deviceRefs {
	return deviceRefs{
		rack: gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refDevice, uint32]{
			Name:     "rack",
			Table:    "rack",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Place,
			Targets:  func(d *refDevice) []uint32 { return []uint32{d.Rack} },
		}),
		parent: gorp.NewKeyReference(
			gorp.KeyReferenceConfig[string, refDevice, string]{
				Name:     "parent",
				Table:    "device",
				OnDelete: gorp.Cascade,
				Kind:     gorp.Place,
				Acyclic:  true,
				Targets: func(d *refDevice) []string {
					if d.Parent == "" {
						return nil
					}
					return []string{d.Parent}
				},
			},
		),
		labels: gorp.NewKeyReference(
			gorp.KeyReferenceConfig[string, refDevice, string]{
				Name:     "labels",
				Table:    "label",
				OnDelete: gorp.Detach,
				Kind:     gorp.Use,
				Targets:  func(d *refDevice) []string { return d.Labels },
				Detach: func(d *refDevice, l string) {
					d.Labels = slices.DeleteFunc(
						slices.Clone(d.Labels),
						func(v string) bool {
							return v == l
						},
					)
				},
			},
		),
	}
}

func (r deviceRefs) all() []gorp.Reference[string, refDevice] {
	return []gorp.Reference[string, refDevice]{r.rack, r.parent, r.labels}
}

func newOwnerRef() *gorp.TargetReference[string, refStatus] {
	return gorp.NewTargetReference(gorp.TargetReferenceConfig[string, refStatus]{
		Name:     "owner",
		Tables:   []string{"rack", "device"},
		OnDelete: gorp.Cascade,
		Kind:     gorp.Place,
		Unique:   true,
		Targets: func(s *refStatus) []gorp.Target {
			if s.Owner == nil {
				return nil
			}
			return []gorp.Target{{Table: s.Owner.Table, Key: s.Owner.Key}}
		},
	})
}

var _ = Describe("Reference", func() {
	var (
		db        *gorp.DB
		racks     *gorp.Table[uint32, refRack]
		labels    *gorp.Table[string, refLabel]
		devices   *gorp.Table[string, refDevice]
		statuses  *gorp.Table[string, refStatus]
		deviceRef deviceRefs
		ownerRef  *gorp.TargetReference[string, refStatus]
		// onDetach is the detach hook of the device table. Nil runs no hook.
		onDetach func(ctx context.Context, tx gorp.Tx, d refDevice) error
		// onTargetSet is the target hook of the device table. Nil runs no hook.
		onTargetSet func(ctx context.Context, tx gorp.Tx, d refDevice) error
		// watched names the tables whose writes onTargetSet watches.
		watched []string
		open    = func(ctx SpecContext, opts ...gorp.Option) {
			GinkgoHelper()
			db = DeferClose(
				gorp.Wrap(memkv.New(), append(opts, gorp.WithCodec(msgpack.Codec))...),
			)
			StartRun(db.Run)
			racks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "rack",
			}))
			labels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
				DB:   db,
				Name: "label",
			}))
			deviceRef = newDeviceRefs()
			onDetach = nil
			onTargetSet = nil
			hooks := make(map[string]func(context.Context, gorp.Tx, refDevice) error)
			for _, table := range watched {
				hooks[table] = func(ctx context.Context, tx gorp.Tx, d refDevice) error {
					if onTargetSet == nil {
						return nil
					}
					return onTargetSet(ctx, tx, d)
				}
			}
			devices = MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
					DB:         db,
					Name:       "device",
					References: deviceRef.all(),
					OnDetach: func(ctx context.Context, tx gorp.Tx, d refDevice) error {
						if onDetach == nil {
							return nil
						}
						return onDetach(ctx, tx, d)
					},
					OnTargetSet: hooks,
				}),
			)
			ownerRef = newOwnerRef()
			statuses = MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, refStatus]{
					DB:         db,
					Name:       "status",
					References: []gorp.Reference[string, refStatus]{ownerRef},
				},
			))
		}
		createRack = func(ctx SpecContext, key uint32) {
			GinkgoHelper()
			Expect(
				racks.NewCreate().Entry(&refRack{Key: key}).Exec(ctx, db),
			).To(Succeed())
		}
		createLabels = func(ctx SpecContext, keys ...string) {
			GinkgoHelper()
			for _, k := range keys {
				Expect(labels.NewCreate().Entry(&refLabel{Key: k}).Exec(ctx, db)).
					To(Succeed())
			}
		}
		createDevice = func(ctx SpecContext, d refDevice) error {
			return devices.NewCreate().Entry(&d).Exec(ctx, db)
		}
		deviceExists = func(ctx SpecContext, key string) bool {
			GinkgoHelper()
			return MustSucceed(devices.NewRetrieve().
				Where(gorp.MatchKeys[string, refDevice](key)).
				Exists(ctx, db))
		}
		rackExists = func(ctx SpecContext, key uint32) bool {
			GinkgoHelper()
			return MustSucceed(racks.NewRetrieve().
				Where(gorp.MatchKeys[uint32, refRack](key)).
				Exists(ctx, db))
		}
	)

	Describe("Writes", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It("Should reject a reference to a missing entry", func(ctx SpecContext) {
			Expect(createDevice(ctx, refDevice{Key: "d1", Rack: 1})).To(And(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("rack: rack 1 does not exist")),
			))
		})

		It("Should accept a reference to an existing entry", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(createDevice(ctx, refDevice{Key: "d1", Rack: 1})).To(Succeed())
		})

		It(
			"Should accept a target created earlier in the same transaction",
			func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := racks.NewCreate().
						Entry(&refRack{Key: 1}).
						Exec(ctx, tx); err != nil {
						return err
					}
					return devices.NewCreate().
						Entry(&refDevice{Key: "d1", Rack: 1}).
						Exec(ctx, tx)
				})).To(Succeed())
			},
		)

		It(
			"Should reject a target in a table the reference does not allow",
			func(ctx SpecContext) {
				Expect(statuses.NewCreate().Entry(&refStatus{
					Key:   "s1",
					Owner: &refOwner{Table: "label", Key: "l1"},
				}).Exec(ctx, db)).To(MatchError(ContainSubstring(
					"owner: status cannot reference label l1",
				)))
			},
		)

		DescribeTable("Should reject a key that is not in its text form",
			func(ctx SpecContext, k string) {
				createRack(ctx, 7)
				Expect(statuses.NewCreate().Entry(&refStatus{
					Key:   "s1",
					Owner: &refOwner{Table: "rack", Key: k},
				}).Exec(ctx, db)).To(And(
					BeAssignableToTypeOf(validate.PathError{}),
					MatchError(ContainSubstring(
						"owner: \""+k+"\" is not a key of rack",
					)),
				))
			},
			Entry("leading zero", "07"),
			Entry("sign", "+7"),
			Entry("not a number", "seven"),
		)

		It("Should resolve a target by its table name", func(ctx SpecContext) {
			createRack(ctx, 7)
			Expect(statuses.NewCreate().Entry(&refStatus{
				Key:   "s1",
				Owner: &refOwner{Table: "rack", Key: "7"},
			}).Exec(ctx, db)).To(Succeed())
		})

		It(
			"Should reject a second entry holding a unique target",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				owner := &refOwner{Table: "rack", Key: "1"}
				Expect(statuses.NewCreate().
					Entry(&refStatus{Key: "s1", Owner: owner}).
					Exec(ctx, db)).To(Succeed())
				Expect(statuses.NewCreate().
					Entry(&refStatus{Key: "s2", Owner: owner}).
					Exec(ctx, db)).To(MatchError(ContainSubstring(
					"owner: status s1 already references rack 1",
				)))
			},
		)

		It(
			"Should let the holder of a unique target rewrite itself",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				s := refStatus{Key: "s1", Owner: &refOwner{Table: "rack", Key: "1"}}
				Expect(statuses.NewCreate().Entry(&s).Exec(ctx, db)).To(Succeed())
				Expect(statuses.NewCreate().Entry(&s).Exec(ctx, db)).To(Succeed())
			},
		)

		It("Should reject a write that forms a cycle", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
			Expect(createDevice(ctx, refDevice{Key: "b", Rack: 1, Parent: "a"})).
				To(Succeed())
			Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1, Parent: "b"})).
				To(MatchError(ContainSubstring("parent: device a would form a cycle")))
			Expect(createDevice(ctx, refDevice{Key: "c", Rack: 1})).To(Succeed())
			Expect(createDevice(ctx, refDevice{Key: "c", Rack: 1, Parent: "c"})).
				To(MatchError(ContainSubstring("parent: device c would form a cycle")))
		})

		It("Should check the references of an update", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(createDevice(ctx, refDevice{Key: "d1", Rack: 1})).To(Succeed())
			Expect(devices.NewUpdate().
				Where(gorp.MatchKeys[string, refDevice]("d1")).
				Change(func(_ gorp.Context, d refDevice) refDevice {
					d.Rack = 2
					return d
				}).
				Exec(ctx, db)).To(MatchError(ContainSubstring(
				"rack: rack 2 does not exist",
			)))
		})

		It(
			"Should let many entries that hold no unique target exist",
			func(ctx SpecContext) {
				for _, k := range []string{"s1", "s2"} {
					Expect(statuses.NewCreate().
						Entry(&refStatus{Key: k}).
						Exec(ctx, db)).To(Succeed())
				}
			},
		)

		It(
			"Should reject a target in a table that is not open",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(racks.Close()).To(Succeed())
				Expect(createDevice(ctx, refDevice{Key: "d1", Rack: 1})).
					To(MatchError(ContainSubstring(`no open table is named "rack"`)))
			},
		)

		It(
			"Should leave writes through package-level queries unchecked",
			func(ctx SpecContext) {
				Expect(gorp.NewCreate[string, refDevice]().
					Entry(&refDevice{Key: "d1", Rack: 99}).
					Exec(ctx, db)).To(Succeed())
			},
		)
	})

	Describe("Concurrent writes", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It(
			"Should keep one of two concurrent first writes of a unique target",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				owner := &refOwner{Table: "rack", Key: "1"}
				first := DeferClose(db.OpenTx())
				Expect(statuses.NewCreate().
					Entry(&refStatus{Key: "s1", Owner: owner}).
					Exec(ctx, first)).To(Succeed())
				second := make(chan error, 1)
				go func() {
					second <- statuses.NewCreate().
						Entry(&refStatus{Key: "s2", Owner: owner}).
						Exec(ctx, db)
				}()
				Consistently(second, 50*time.Millisecond).ShouldNot(Receive())
				Expect(first.Commit(ctx)).To(Succeed())
				Eventually(second).Should(Receive(MatchError(ContainSubstring(
					"owner: status s1 already references rack 1",
				))))
				var all []refStatus
				Expect(statuses.NewRetrieve().Entries(&all).Exec(ctx, db)).
					To(Succeed())
				Expect(all).To(ConsistOf(refStatus{Key: "s1", Owner: owner}))
			},
		)

		It(
			"Should not commit both a restrict delete and a create of a dependent",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				first := DeferClose(db.OpenTx())
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, first)).To(Succeed())
				second := make(chan error, 1)
				go func() {
					second <- createDevice(ctx, refDevice{Key: "d1", Rack: 1})
				}()
				Consistently(second, 50*time.Millisecond).ShouldNot(Receive())
				Expect(first.Commit(ctx)).To(Succeed())
				Eventually(second).Should(Receive(MatchError(ContainSubstring(
					"rack: rack 1 does not exist",
				))))
				Expect(rackExists(ctx, 1)).To(BeFalse())
				Expect(deviceExists(ctx, "d1")).To(BeFalse())
			},
		)
	})

	Describe("NewUncheckedWriter", func() {
		BeforeEach(func(ctx SpecContext) {
			open(ctx)
		})

		It("Should keep the indexes current without checks", func(ctx SpecContext) {
			Expect(devices.NewUncheckedWriter(db).
				Set(ctx, refDevice{Key: "a", Rack: 9})).To(Succeed())
			Expect(gorp.WrapWriter[string, refDevice](db).
				Set(ctx, refDevice{Key: "b", Rack: 9})).To(Succeed())
			Expect(deviceRef.rack.Get(ctx, db, 9)).
				To(ConsistOf("a", "b"))
		})

		It("Should only check targets that a write changes", func(ctx SpecContext) {
			Expect(devices.NewUncheckedWriter(db).
				Set(ctx, refDevice{Key: "a", Rack: 9})).To(Succeed())
			createLabels(ctx, "l1")
			labeled := refDevice{Key: "a", Rack: 9, Labels: []string{"l1"}}
			Expect(createDevice(ctx, labeled)).To(Succeed())
			Expect(createDevice(ctx, refDevice{Key: "a", Rack: 8})).
				To(MatchError(ContainSubstring("rack: rack 8 does not exist")))
		})

		DescribeTable("Should not check a target that a write keeps, at any position",
			func(ctx SpecContext, labels []string) {
				Expect(devices.NewUncheckedWriter(db).Set(ctx, refDevice{
					Key:    "a",
					Labels: []string{"gone", "l1"},
				})).To(Succeed())
				createLabels(ctx, "l1", "l2", "l3")
				Expect(createDevice(ctx, refDevice{Key: "a", Labels: labels})).
					To(Succeed())
			},
			Entry("at its position", []string{"gone", "l1", "l2"}),
			Entry("at a moved position", []string{"l2", "gone", "l1"}),
			Entry("two positions away", []string{"l2", "l3", "gone", "l1"}),
		)

		It("Should index the last of two target changes in one transaction",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				createRack(ctx, 2)
				createRack(ctx, 3)
				Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					for _, rack := range []uint32{2, 3} {
						d := refDevice{Key: "a", Rack: rack}
						if err := devices.NewCreate().
							Entry(&d).
							Exec(ctx, tx); err != nil {
							return err
						}
					}
					return nil
				})).To(Succeed())
				Expect(deviceRef.rack.Get(ctx, db, 3)).To(ConsistOf("a"))
				Expect(deviceRef.rack.Get(ctx, db, 2)).To(BeEmpty())
			},
		)
	})

	Describe("Get", func() {
		BeforeEach(func(ctx SpecContext) {
			open(ctx)
		})

		It(
			"Should find an entry from an observer of its commit",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				var found []string
				DeferCleanup(devices.Observe().OnChange(func(
					context.Context,
					gorp.TxReader[string, refDevice],
				) {
					found = MustSucceed(deviceRef.rack.Get(ctx, nil, 1))
				}))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return devices.NewCreate().
						Entry(&refDevice{Key: "a", Rack: 1}).
						Exec(ctx, tx)
				})).To(Succeed())
				Expect(found).To(Equal([]string{"a"}))
			},
		)
	})

	Describe("Filter", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It("Should find the entries holding a target", func(ctx SpecContext) {
			createRack(ctx, 1)
			createLabels(ctx, "l1", "l2")
			Expect(
				createDevice(ctx, refDevice{Key: "a", Rack: 1, Labels: []string{"l1"}}),
			).
				To(Succeed())
			Expect(createDevice(ctx, refDevice{
				Key:    "b",
				Rack:   1,
				Labels: []string{"l1", "l2"},
			})).To(Succeed())
			var res []refDevice
			Expect(devices.NewRetrieve().
				Where(deviceRef.labels.Filter("l2")).
				Entries(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal("b"))
		})

		It(
			"Should page the entries holding a target in storage order",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				keys := []string{"e", "b", "d", "a", "c"}
				for _, k := range keys {
					Expect(createDevice(ctx, refDevice{Key: k, Rack: 1})).To(Succeed())
				}
				page := func(after string) []string {
					q := devices.NewRetrieve().Where(deviceRef.rack.Filter(1)).Limit(2)
					if after != "" {
						q = q.After(after)
					}
					var res []refDevice
					Expect(q.Entries(&res).Exec(ctx, db)).To(Succeed())
					return lo.Map(res, func(d refDevice, _ int) string { return d.Key })
				}
				first := page("")
				second := page(first[1])
				Expect(slices.Concat(first, second, page(second[1]))).
					To(Equal([]string{"a", "b", "c", "d", "e"}))
			},
		)

		It("Should see writes staged in the same transaction", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := devices.NewCreate().
					Entry(&refDevice{Key: "a", Rack: 1}).
					Exec(ctx, tx); err != nil {
					return err
				}
				Expect(deviceRef.rack.Get(ctx, tx, 1)).
					To(ConsistOf("a"))
				return nil
			})).To(Succeed())
		})
	})

	Describe("Referenced", func() {
		all := func(string) bool { return true }
		BeforeEach(func(ctx SpecContext) {
			open(ctx)
			createRack(ctx, 1)
			createRack(ctx, 2)
			createRack(ctx, 3)
			Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
			Expect(createDevice(ctx, refDevice{Key: "b", Rack: 2})).To(Succeed())
		})

		It("Should return the targets that an entry holds", func(ctx SpecContext) {
			Expect(deviceRef.rack.Referenced(ctx, db, all, 1, 2, 3)).
				To(Equal(set.New[uint32](1, 2)))
		})

		It("Should count only the entries that in accepts", func(ctx SpecContext) {
			Expect(createDevice(ctx, refDevice{Key: "c", Rack: 2})).To(Succeed())
			c := func(k string) bool { return k == "c" }
			Expect(deviceRef.rack.Referenced(ctx, db, c, 1, 2, 3)).
				To(Equal(set.New[uint32](2)))
		})

		It("Should see writes staged in the same transaction", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "a", Rack: 3}).
				Exec(ctx, tx)).To(Succeed())
			Expect(deviceRef.rack.Referenced(ctx, tx, all, 1, 2, 3)).
				To(Equal(set.New[uint32](2, 3)))
			Expect(deviceRef.rack.Referenced(ctx, db, all, 1, 2, 3)).
				To(Equal(set.New[uint32](1, 2)))
		})

		It("Should drop a target once no entry holds it", func(ctx SpecContext) {
			Expect(devices.NewDelete().
				Where(gorp.MatchKeys[string, refDevice]("b")).
				Exec(ctx, db)).To(Succeed())
			Expect(deviceRef.rack.Referenced(ctx, db, all, 1, 2)).
				To(Equal(set.New[uint32](1)))
		})
	})

	Describe("FilterHolding", func() {
		BeforeEach(func(ctx SpecContext) {
			open(ctx)
			createRack(ctx, 1)
			createLabels(ctx, "l")
			Expect(
				createDevice(ctx, refDevice{Key: "a", Rack: 1, Labels: []string{"l"}}),
			).
				To(Succeed())
			Expect(createDevice(ctx, refDevice{Key: "b", Rack: 1})).To(Succeed())
		})
		keysOf := func(ctx SpecContext, tx gorp.Tx, holding bool) []string {
			GinkgoHelper()
			var res []refDevice
			Expect(devices.NewRetrieve().
				Where(deviceRef.labels.FilterHolding(holding)).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			return lo.Map(res, func(d refDevice, _ int) string { return d.Key })
		}

		It("Should find the entries that hold a target or none", func(ctx SpecContext) {
			Expect(keysOf(ctx, db, true)).To(ConsistOf("a"))
			Expect(keysOf(ctx, db, false)).To(ConsistOf("b"))
		})

		It("Should see writes staged in the same transaction", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "a", Rack: 1}).
				Exec(ctx, tx)).To(Succeed())
			Expect(keysOf(ctx, tx, false)).To(ConsistOf("a", "b"))
			Expect(keysOf(ctx, tx, true)).To(BeEmpty())
		})

		It("Should drop the entries a delete removes", func(ctx SpecContext) {
			Expect(devices.NewDelete().
				Where(gorp.MatchKeys[string, refDevice]("b")).
				Exec(ctx, db)).To(Succeed())
			Expect(keysOf(ctx, db, false)).To(BeEmpty())
		})
	})

	Describe("FilterAs", func() {
		BeforeEach(func(ctx SpecContext) {
			open(ctx)
			createRack(ctx, 1)
			createRack(ctx, 2)
			Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
			Expect(createDevice(ctx, refDevice{Key: "b", Rack: 2})).To(Succeed())
		})

		It(
			"Should find the entries holding a target as another entry type",
			func(ctx SpecContext) {
				var res []refDeviceName
				Expect(gorp.NewRetrieve[string, refDeviceName]().
					Where(deviceRef.rack.FilterAs[refDeviceName](2)).
					Entries(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res).To(Equal([]refDeviceName{{Key: "b"}}))
			},
		)

		It("Should see writes staged in the same transaction", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "c", Rack: 2}).
				Exec(ctx, tx)).To(Succeed())
			var res []refDeviceName
			Expect(gorp.NewRetrieve[string, refDeviceName]().
				Where(deviceRef.rack.FilterAs[refDeviceName](2)).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(ConsistOf(refDeviceName{Key: "b"}, refDeviceName{Key: "c"}))
		})
	})

	Describe("Dependents", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It(
			"Should list the entries of every table that hold a target",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
				Expect(statuses.NewCreate().
					Entry(&refStatus{Key: "s", Owner: &refOwner{Table: "rack", Key: "1"}}).
					Exec(ctx, db)).To(Succeed())
				Expect(racks.Dependents(ctx, db, 1)).To(ConsistOf(
					gorp.Target{Table: "device", Key: "a"},
					gorp.Target{Table: "status", Key: "s"},
				))
			},
		)

		It("Should return nothing for a target no entry holds", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(racks.Dependents(ctx, db, 1)).To(BeEmpty())
		})

		It("Should list the entries that hold a target through a snapshot", func(
			ctx SpecContext,
		) {
			createRack(ctx, 1)
			Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
			var deps []gorp.Target
			Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
				deps, err = racks.Dependents(ctx, snap, 1)
				return err
			})).To(Succeed())
			Expect(deps).To(Equal([]gorp.Target{{Table: "device", Key: "a"}}))
		})

		It("Should see writes staged in the same transaction", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
			tx := DeferClose(db.OpenTx())
			Expect(devices.NewDelete().
				Where(gorp.MatchKeys[string, refDevice]("a")).
				Exec(ctx, tx)).To(Succeed())
			Expect(racks.Dependents(ctx, tx, 1)).To(BeEmpty())
			Expect(racks.Dependents(ctx, db, 1)).To(HaveLen(1))
		})
	})

	Describe("Delete", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It(
			"Should refuse to delete a target that an entry restricts",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(createDevice(ctx, refDevice{Key: "d1", Rack: 1})).To(Succeed())
				err := racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, db)
				Expect(err).To(MatchError(validate.ErrValidation))
				Expect(err).To(MatchError(
					"cannot delete rack 1 because device d1 references it through rack: " +
						"validation error",
				))
				Expect(
					MustBeOk(errors.AsType[gorp.RestrictError](err)),
				).To(Equal(gorp.RestrictError{
					Target:    gorp.NewTarget("rack", uint32(1)),
					Holder:    gorp.NewTarget("device", "d1"),
					Reference: "rack",
				}))
				Expect(rackExists(ctx, 1)).To(BeTrue())
			},
		)

		It(
			"Should stop following the references of a closed table",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(createDevice(ctx, refDevice{Key: "d1", Rack: 1})).To(Succeed())
				Expect(devices.Close()).To(Succeed())
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, db)).To(Succeed())
			},
		)
		It(
			"Should delete a restricted target once nothing references it",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(createDevice(ctx, refDevice{Key: "d1", Rack: 1})).To(Succeed())
				Expect(devices.NewDelete().
					Where(gorp.MatchKeys[string, refDevice]("d1")).
					Exec(ctx, db)).To(Succeed())
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, db)).To(Succeed())
			},
		)

		It(
			"Should see a dependent created earlier in the same transaction",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := devices.NewCreate().
						Entry(&refDevice{Key: "d1", Rack: 1}).
						Exec(ctx, tx); err != nil {
						return err
					}
					return racks.NewDelete().
						Where(gorp.MatchKeys[uint32, refRack](1)).
						Exec(ctx, tx)
				})).To(MatchError(ContainSubstring("references it through rack")))
			},
		)

		It(
			"Should cascade through self and polymorphic references",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
				Expect(createDevice(ctx, refDevice{Key: "b", Rack: 1, Parent: "a"})).
					To(Succeed())
				Expect(createDevice(ctx, refDevice{Key: "c", Rack: 1, Parent: "b"})).
					To(Succeed())
				Expect(statuses.NewCreate().Entry(&refStatus{
					Key:   "s",
					Owner: &refOwner{Table: "device", Key: "c"},
				}).Exec(ctx, db)).To(Succeed())
				Expect(devices.NewDelete().
					Where(gorp.MatchKeys[string, refDevice]("a")).
					Exec(ctx, db)).To(Succeed())
				Expect(deviceExists(ctx, "a")).To(BeFalse())
				Expect(deviceExists(ctx, "b")).To(BeFalse())
				Expect(deviceExists(ctx, "c")).To(BeFalse())
				Expect(statuses.NewRetrieve().
					Where(gorp.MatchKeys[string, refStatus]("s")).
					Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should run guards only on the entries the query matches",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
				Expect(createDevice(ctx, refDevice{Key: "b", Rack: 1, Parent: "a"})).
					To(Succeed())
				Expect(devices.NewDelete().
					Where(gorp.MatchKeys[string, refDevice]("a")).
					Guard(func(_ gorp.Context, d refDevice) error {
						if d.Parent != "" {
							return validate.ErrValidation
						}
						return nil
					}).
					Exec(ctx, db)).To(Succeed())
				Expect(deviceExists(ctx, "b")).To(BeFalse())
			},
		)

		It(
			"Should detach a deleted target from the entries it keeps",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				createLabels(ctx, "l1", "l2")
				Expect(createDevice(ctx, refDevice{
					Key:    "d1",
					Rack:   1,
					Labels: []string{"l1", "l2"},
				})).To(Succeed())
				Expect(labels.NewDelete().
					Where(gorp.MatchKeys[string, refLabel]("l1")).
					Exec(ctx, db)).To(Succeed())
				var d refDevice
				Expect(devices.NewRetrieve().
					Where(gorp.MatchKeys[string, refDevice]("d1")).
					Entry(&d).
					Exec(ctx, db)).To(Succeed())
				Expect(d.Labels).To(Equal([]string{"l2"}))
				keys := MustSucceed(deviceRef.labels.Get(ctx, nil, "l1"))
				Expect(keys).To(BeEmpty())
			},
		)

		Describe("OnDetach", func() {
			var detached []refDevice
			BeforeEach(func(ctx SpecContext) {
				detached = nil
				onDetach = func(ctx context.Context, tx gorp.Tx, d refDevice) error {
					detached = append(detached, d)
					return racks.NewCreate().Entry(&refRack{Key: 7}).Exec(ctx, tx)
				}
				createRack(ctx, 1)
				createLabels(ctx, "l1", "l2")
				Expect(createDevice(ctx, refDevice{
					Key:    "d1",
					Rack:   1,
					Labels: []string{"l1", "l2"},
				})).To(Succeed())
				Expect(createDevice(ctx, refDevice{
					Key:    "d2",
					Rack:   1,
					Labels: []string{"l2"},
				})).To(Succeed())
			})

			It(
				"Should run the hook in the delete's transaction with the entry as written",
				func(ctx SpecContext) {
					tx := DeferClose(db.OpenTx())
					Expect(labels.NewDelete().
						Where(gorp.MatchKeys[string, refLabel]("l1")).
						Exec(ctx, tx)).To(Succeed())
					Expect(detached).To(HaveLen(1))
					Expect(detached[0].Key).To(Equal("d1"))
					Expect(detached[0].Labels).To(Equal([]string{"l2"}))
					Expect(rackExists(ctx, 7)).To(BeFalse())
					Expect(tx.Commit(ctx)).To(Succeed())
					Expect(rackExists(ctx, 7)).To(BeTrue())
				},
			)

			It("Should fail the delete when the hook fails", func(ctx SpecContext) {
				hookErr := errors.New("hook failed")
				onDetach = func(context.Context, gorp.Tx, refDevice) error {
					return hookErr
				}
				Expect(labels.NewDelete().
					Where(gorp.MatchKeys[string, refLabel]("l1")).
					Exec(ctx, db)).To(MatchError(hookErr))
				var d refDevice
				Expect(devices.NewRetrieve().
					Where(gorp.MatchKeys[string, refDevice]("d1")).
					Entry(&d).
					Exec(ctx, db)).To(Succeed())
				Expect(d.Labels).To(Equal([]string{"l1", "l2"}))
			})
		})
	})

	Describe("OnTargetSet", func() {
		var set []string
		BeforeEach(func() {
			watched = []string{"label", "rack"}
			DeferCleanup(func() { watched = nil })
		})
		JustBeforeEach(func(ctx SpecContext) {
			open(ctx)
			createRack(ctx, 1)
			createLabels(ctx, "l1", "l2")
			Expect(createDevice(ctx, refDevice{
				Key:    "d1",
				Rack:   1,
				Labels: []string{"l1", "l2"},
			})).To(Succeed())
			Expect(createDevice(ctx, refDevice{
				Key:    "d2",
				Rack:   1,
				Labels: []string{"l2"},
			})).To(Succeed())
			set = nil
			onTargetSet = func(ctx context.Context, tx gorp.Tx, d refDevice) error {
				set = append(set, d.Key)
				return racks.NewCreate().Entry(&refRack{Key: 7}).Exec(ctx, tx)
			}
		})

		It(
			"Should run the hook in the write's transaction for each entry that uses "+
				"the target",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				Expect(labels.NewCreate().Entry(&refLabel{Key: "l2"}).Exec(ctx, tx)).
					To(Succeed())
				Expect(set).To(ConsistOf("d1", "d2"))
				Expect(rackExists(ctx, 7)).To(BeFalse())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(rackExists(ctx, 7)).To(BeTrue())
			},
		)

		It("Should not run the hook for a target that no entry uses", func(
			ctx SpecContext,
		) {
			createLabels(ctx, "l3")
			Expect(set).To(BeEmpty())
		})

		It("Should not run the hook for a Place reference", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(set).To(BeEmpty())
		})

		It("Should fail the write when the hook fails", func(ctx SpecContext) {
			hookErr := errors.New("hook failed")
			onTargetSet = func(context.Context, gorp.Tx, refDevice) error {
				return hookErr
			}
			Expect(labels.NewCreate().Entry(&refLabel{Key: "l1"}).Exec(ctx, db)).
				To(MatchError(hookErr))
		})

		Context("When the table does not watch the target's table", func() {
			BeforeEach(func() { watched = []string{"rack"} })

			It("Should not run the hook", func(ctx SpecContext) {
				createLabels(ctx, "l1")
				Expect(set).To(BeEmpty())
			})
		})
	})

	Describe("Delete size", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx, gorp.WithMaxDeleteSize(2)) })

		It(
			"Should refuse a delete that cascades past the limit",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(createDevice(ctx, refDevice{Key: "a", Rack: 1})).To(Succeed())
				Expect(createDevice(ctx, refDevice{Key: "b", Rack: 1, Parent: "a"})).
					To(Succeed())
				Expect(createDevice(ctx, refDevice{Key: "c", Rack: 1, Parent: "b"})).
					To(Succeed())
				Expect(devices.NewDelete().
					Where(gorp.MatchKeys[string, refDevice]("a")).
					Exec(ctx, db)).To(MatchError(ContainSubstring(
					"deleting device a would change more than 2 entries",
				)))
				Expect(deviceExists(ctx, "a")).To(BeTrue())
				Expect(deviceExists(ctx, "c")).To(BeTrue())
			},
		)

		It("Should count detached dependents toward the limit", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(labels.NewCreate().Entry(&refLabel{Key: "l"}).Exec(ctx, db)).
				To(Succeed())
			for _, k := range []string{"a", "b"} {
				Expect(createDevice(ctx, refDevice{
					Key:    k,
					Rack:   1,
					Labels: []string{"l"},
				})).To(Succeed())
			}
			Expect(labels.NewDelete().
				Where(gorp.MatchKeys[string, refLabel]("l")).
				Exec(ctx, db)).To(MatchError(ContainSubstring(
				"deleting label l would change more than 2 entries",
			)))
		})

		It(
			"Should refuse a delete that more dependents restrict than the limit",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				for _, k := range []string{"a", "b", "c"} {
					Expect(createDevice(ctx, refDevice{Key: k, Rack: 1})).To(Succeed())
				}
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, db)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("cannot delete rack 1 because device")),
				))
				Expect(rackExists(ctx, 1)).To(BeTrue())
			},
		)

		It(
			"Should count the entries the query names toward the limit",
			func(ctx SpecContext) {
				for _, k := range []uint32{1, 2, 3} {
					createRack(ctx, k)
				}
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1, 2, 3)).
					Exec(ctx, db)).To(MatchError(ContainSubstring(
					"would change more than 2 entries",
				)))
			},
		)
	})

	Describe("Populate", func() {
		It(
			"Should index entries written before the table opened",
			func(ctx SpecContext) {
				kvs := memkv.New()
				pre := gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
				Expect(gorp.NewCreate[uint32, refRack]().
					Entry(&refRack{Key: 1}).
					Exec(ctx, pre)).To(Succeed())
				Expect(gorp.NewCreate[string, refDevice]().
					Entry(&refDevice{Key: "d1", Rack: 1}).
					Exec(ctx, pre)).To(Succeed())
				db = DeferClose(gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec)))
				StartRun(db.Run)
				racks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
					DB:   db,
					Name: "rack",
				}))
				MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
					DB:         db,
					Name:       "device",
					References: newDeviceRefs().all(),
				}))
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, db)).To(MatchError(ContainSubstring("through rack")))
			},
		)

		It(
			"Should end a probe that waits for the index when its context ends",
			func(ctx SpecContext) {
				kvs := &blockingDB{DB: memkv.New(), release: make(chan struct{})}
				db = DeferClose(gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec)))
				StartRun(db.Run)
				racks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
					DB:   db,
					Name: "rack",
				}))
				deviceRef = newDeviceRefs()
				MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
					DB:         db,
					Name:       "device",
					References: deviceRef.all(),
				}))
				DeferCleanup(func() { close(kvs.release) })
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				Expect(deviceRef.rack.Get(canceled, nil, 1)).Error().
					To(MatchError(context.Canceled))
				Expect(racks.Dependents(canceled, db, 1)).Error().
					To(MatchError(context.Canceled))
			},
		)

		Describe("Failure", func() {
			BeforeEach(func(ctx SpecContext) {
				kvs := DeferClose(memkv.New())
				pre := gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
				Expect(gorp.NewCreate[uint32, refRack]().
					Entry(&refRack{Key: 1}).
					Exec(ctx, pre)).To(Succeed())
				Expect(gorp.NewCreate[string, refDevice]().
					Entry(&refDevice{Key: "d1", Rack: 1}).
					Exec(ctx, pre)).To(Succeed())
				db = gorp.Wrap(
					&failOnceDB{DB: kvs, err: errors.New("populate boom")},
					gorp.WithCodec(msgpack.Codec),
				)
				StartRun(db.Run)
				deviceRef = newDeviceRefs()
				devices = MustOpen(
					gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
						DB:         db,
						Name:       "device",
						References: deviceRef.all(),
					}),
				)
				racks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
					DB:   db,
					Name: "rack",
				}))
			})

			It("Should fail a probe of the reference", func(ctx SpecContext) {
				Expect(deviceRef.rack.Get(ctx, nil, 1)).Error().
					To(MatchError(gorp.ErrIndexInvalid))
				Expect(racks.Dependents(ctx, db, 1)).Error().
					To(MatchError(gorp.ErrIndexInvalid))
			})

			It("Should fail a delete of a target", func(ctx SpecContext) {
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, db)).To(MatchError(gorp.ErrIndexInvalid))
				Expect(rackExists(ctx, 1)).To(BeTrue())
			})

			It("Should fall back to a scan in a filter", func(ctx SpecContext) {
				var racked, unlabeled []refDevice
				Expect(devices.NewRetrieve().
					Where(deviceRef.rack.Filter(1)).
					Entries(&racked).
					Exec(ctx, db)).To(Succeed())
				Expect(racked).To(HaveLen(1))
				Expect(devices.NewRetrieve().
					Where(deviceRef.labels.FilterHolding(false)).
					Entries(&unlabeled).
					Exec(ctx, db)).To(Succeed())
				Expect(unlabeled).To(HaveLen(1))
			})

			It("Should scan the table in Referenced", func(ctx SpecContext) {
				createRack(ctx, 2)
				tx := DeferClose(db.OpenTx())
				Expect(devices.NewCreate().
					Entry(&refDevice{Key: "d2", Rack: 2}).
					Exec(ctx, tx)).To(Succeed())
				all := func(string) bool { return true }
				Expect(deviceRef.rack.Referenced(ctx, tx, all, 1, 2)).
					To(Equal(set.New[uint32](1, 2)))
				d1 := func(k string) bool { return k == "d1" }
				Expect(deviceRef.rack.Referenced(ctx, tx, d1, 1, 2)).
					To(Equal(set.New[uint32](1)))
			})

			It("Should fail a filter on another entry type", func(ctx SpecContext) {
				var res []refDeviceName
				Expect(gorp.NewRetrieve[string, refDeviceName]().
					Where(deviceRef.rack.FilterAs[refDeviceName](1)).
					Entries(&res).
					Exec(ctx, db)).To(MatchError(gorp.ErrIndexInvalid))
			})
		})
	})

	Describe("Registry", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It(
			"Should refuse a second open table with the same name",
			func(ctx SpecContext) {
				Expect(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
					DB:   db,
					Name: "rack",
				})).Error().To(MatchError(ContainSubstring(`table "rack" is already open`)))
			},
		)

		It(
			"Should refuse a second open table that stores under the same key prefix",
			func(ctx SpecContext) {
				Expect(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
					DB:   db,
					Name: "other_rack",
				})).Error().To(MatchError(
					`[gorp] - table "other_rack" stores under the key prefix of open ` +
						`table "rack"`,
				))
			},
		)

		It("Should free the key prefix of a closed table", func(ctx SpecContext) {
			Expect(racks.Close()).To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "other_rack",
			}))
		})

		DescribeTable(
			"Should reject an invalid reference",
			func(
				ctx SpecContext,
				cfg gorp.KeyReferenceConfig[string, refProject, string],
				msg string,
			) {
				Expect(gorp.OpenTable(ctx, gorp.TableConfig[string, refProject]{
					DB:   db,
					Name: "project",
					References: []gorp.Reference[string, refProject]{
						gorp.NewKeyReference(
							gorp.KeyReferenceConfig[string, refProject, string]{
								Name:     "parent",
								Table:    "project",
								OnDelete: gorp.Cascade,
								Kind:     gorp.Place,
								Targets:  func(*refProject) []string { return nil },
							},
						),
						gorp.NewKeyReference(cfg),
					},
				})).Error().To(MatchError(ContainSubstring(msg)))
			},
			Entry("with no name", gorp.KeyReferenceConfig[string, refProject, string]{
				Table:    "label",
				OnDelete: gorp.Restrict,
				Kind:     gorp.Use,
				Targets:  func(*refProject) []string { return nil },
			}, "project.references.1.name: required"),
			Entry("with no table", gorp.KeyReferenceConfig[string, refProject, string]{
				Name:     "label",
				OnDelete: gorp.Restrict,
				Kind:     gorp.Use,
				Targets:  func(*refProject) []string { return nil },
			}, "project.references.1.table: must name at least one table"),
			Entry(
				"with no targets",
				gorp.KeyReferenceConfig[string, refProject, string]{
					Name:     "label",
					Table:    "label",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
				},
				"project.references.1.targets: must be non-nil",
			),
			Entry("with no policy", gorp.KeyReferenceConfig[string, refProject, string]{
				Name:    "label",
				Table:   "label",
				Kind:    gorp.Use,
				Targets: func(*refProject) []string { return nil },
			}, "project.references.1.on_delete: must be Restrict, Cascade, Detach, or Inline"),
			Entry("with no kind", gorp.KeyReferenceConfig[string, refProject, string]{
				Name:     "label",
				Table:    "label",
				OnDelete: gorp.Restrict,
				Targets:  func(*refProject) []string { return nil },
			}, "project.references.1.kind: must be Place or Use"),
			Entry(
				"detaching with no Detach",
				gorp.KeyReferenceConfig[string, refProject, string]{
					Name:     "label",
					Table:    "label",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(*refProject) []string { return nil },
				},
				"project.references.1.detach: must be set when on_delete is Detach",
			),
			Entry(
				"inlining with no Inline",
				gorp.KeyReferenceConfig[string, refProject, string]{
					Name:     "label",
					Table:    "label",
					OnDelete: gorp.Inline,
					Kind:     gorp.Use,
					Targets:  func(*refProject) []string { return nil },
				},
				"project.references.1.inline: must be set when on_delete is Inline",
			),
			Entry(
				"acyclic across tables",
				gorp.KeyReferenceConfig[string, refProject, string]{
					Name:     "label",
					Table:    "label",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Use,
					Acyclic:  true,
					Targets:  func(*refProject) []string { return nil },
				},
				"project.references.1.acyclic: requires the reference to point at its own table",
			),
		)
	})
})

var _ = Describe("Reference closure", func() {
	var (
		db         *gorp.DB
		projects   *gorp.Table[string, refProject]
		groups     *gorp.Table[string, refGroup]
		schematics *gorp.Table[string, refSchematic]
	)
	// open opens a project that groups and schematics cascade from, and schematics
	// that reference their group with policy.
	open := func(ctx SpecContext, policy gorp.OnDelete, opts ...gorp.Option) {
		GinkgoHelper()
		db = DeferClose(
			gorp.Wrap(memkv.New(), append(opts, gorp.WithCodec(msgpack.Codec))...),
		)
		StartRun(db.Run)
		projects = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refProject]{
			DB:   db,
			Name: "project",
		}))
		groups = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refGroup]{
			DB:   db,
			Name: "group",
			References: []gorp.Reference[string, refGroup]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refGroup, string]{
					Name:     "project",
					Table:    "project",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets:  func(g *refGroup) []string { return []string{g.Project} },
				}),
			},
		}))
		group := gorp.KeyReferenceConfig[string, refSchematic, string]{
			Name:     "group",
			Table:    "group",
			OnDelete: policy,
			Kind:     gorp.Place,
			Targets: func(s *refSchematic) []string {
				if s.Group == "" {
					return nil
				}
				return []string{s.Group}
			},
		}
		if policy == gorp.Detach {
			group.Detach = func(s *refSchematic, _ string) { s.Group = "" }
		}
		schematics = MustOpen(gorp.OpenTable(
			ctx,
			gorp.TableConfig[string, refSchematic]{
				DB:   db,
				Name: "schematic",
				References: []gorp.Reference[string, refSchematic]{
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, refSchematic, string]{
							Name:     "project",
							Table:    "project",
							OnDelete: gorp.Cascade,
							Kind:     gorp.Place,
							Targets: func(s *refSchematic) []string {
								return []string{s.Project}
							},
						},
					),
					gorp.NewKeyReference(group),
				},
			},
		))
		Expect(projects.NewCreate().Entry(&refProject{Key: "p"}).Exec(ctx, db)).
			To(Succeed())
		Expect(groups.NewCreate().
			Entry(&refGroup{Key: "g", Project: "p"}).
			Exec(ctx, db)).To(Succeed())
		Expect(schematics.NewCreate().
			Entry(&refSchematic{Key: "s", Project: "p", Group: "g"}).
			Exec(ctx, db)).To(Succeed())
	}
	deleteProject := func(ctx SpecContext) error {
		return projects.NewDelete().
			Where(gorp.MatchKeys[string, refProject]("p")).
			Exec(ctx, db)
	}
	schematicExists := func(ctx SpecContext) bool {
		GinkgoHelper()
		return MustSucceed(schematics.NewRetrieve().
			Where(gorp.MatchKeys[string, refSchematic]("s")).
			Exists(ctx, db))
	}

	It(
		"Should not restrict a delete on a dependent the delete removes",
		func(ctx SpecContext) {
			open(ctx, gorp.Restrict)
			Expect(groups.NewDelete().
				Where(gorp.MatchKeys[string, refGroup]("g")).
				Exec(ctx, db)).To(MatchError(ContainSubstring("through group")))
			Expect(deleteProject(ctx)).To(Succeed())
			Expect(schematicExists(ctx)).To(BeFalse())
		},
	)

	It(
		"Should not detach a target from an entry the delete removes",
		func(ctx SpecContext) {
			open(ctx, gorp.Detach, gorp.WithMaxDeleteSize(3))
			Expect(deleteProject(ctx)).To(Succeed())
			Expect(schematicExists(ctx)).To(BeFalse())
		},
	)
})

var _ = Describe("Reference migrations", func() {
	var db *gorp.DB
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
	})

	It(
		"Should let a migration write a reference to the table it migrates",
		func(ctx SpecContext) {
			devices := MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, refDevice]{
					DB:         db,
					Name:       "device",
					References: newDeviceRefs().all(),
				},
			))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "rack",
				Migrations: []gorp.Migration{
					migrate.New(
						"create_device",
						func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
							if err := gorp.WrapWriter[uint32, refRack](tx).
								Set(ctx, refRack{Key: 1}); err != nil {
								return err
							}
							return devices.NewCreate().
								Entry(&refDevice{Key: "a", Rack: 1}).
								Exec(ctx, tx)
						},
					),
				},
			}))
			Expect(devices.NewRetrieve().
				Where(gorp.MatchKeys[string, refDevice]("a")).
				Exists(ctx, db)).To(BeTrue())
		},
	)

	It(
		"Should not follow the references of a table while it migrates",
		func(ctx SpecContext) {
			racks := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "rack",
			}))
			Expect(racks.NewCreate().Entry(&refRack{Key: 1}).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, refDevice]{
					DB:         db,
					Name:       "device",
					References: newDeviceRefs().all(),
					Migrations: []gorp.Migration{
						migrate.New(
							"delete_rack",
							func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
								return racks.NewDelete().
									Where(gorp.MatchKeys[uint32, refRack](1)).
									Exec(ctx, tx)
							},
						),
					},
				},
			))
			Expect(racks.NewRetrieve().
				Where(gorp.MatchKeys[uint32, refRack](1)).
				Exists(ctx, db)).To(BeFalse())
		},
		SpecTimeout(5*time.Second),
	)

	It("Should free the name of a table whose migrations fail", func(ctx SpecContext) {
		Expect(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
			DB:   db,
			Name: "rack",
			Migrations: []gorp.Migration{migrate.New(
				"fail",
				func(context.Context, gorp.Tx, alamos.Instrumentation) error {
					return errors.New("migration failed")
				},
			)},
		})).Error().To(MatchError(ContainSubstring("migration failed")))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
			DB:   db,
			Name: "rack",
		}))
	})
})

var _ = Describe("DropDangling", func() {
	var (
		db   *gorp.DB
		refs = func(
			rack func(d *refDevice) []gorp.Target,
			rackDetach func(d *refDevice, t gorp.Target),
		) []gorp.Reference[string, refDevice] {
			return []gorp.Reference[string, refDevice]{
				gorp.NewTargetReference(gorp.TargetReferenceConfig[string, refDevice]{
					Name:     "rack",
					Tables:   []string{"rack"},
					OnDelete: gorp.Restrict,
					Kind:     gorp.Place,
					Targets:  rack,
					Detach:   rackDetach,
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refDevice, string]{
					Name:     "labels",
					Table:    "label",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(d *refDevice) []string { return d.Labels },
					Detach: func(d *refDevice, t string) {
						d.Labels = slices.DeleteFunc(d.Labels, func(l string) bool {
							return l == t
						})
					},
				}),
			}
		}
		rackTarget = func(d *refDevice) []gorp.Target {
			if d.Rack == 0 {
				return nil
			}
			return []gorp.Target{gorp.NewTarget("rack", d.Rack)}
		}
		clearRack = func(d *refDevice, _ gorp.Target) { d.Rack = 0 }
		open      = func(
			ctx SpecContext,
			refs []gorp.Reference[string, refDevice],
			migrations ...gorp.Migration,
		) *gorp.Table[string, refDevice] {
			GinkgoHelper()
			return MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         db,
				Name:       "device",
				References: refs,
				Migrations: migrations,
			}))
		}
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		racks := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
			DB:   db,
			Name: "rack",
		}))
		labels := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
			DB:   db,
			Name: "label",
		}))
		Expect(racks.NewCreate().Entry(&refRack{Key: 1}).Exec(ctx, db)).To(Succeed())
		Expect(labels.NewCreate().Entry(&refLabel{Key: "a"}).Exec(ctx, db)).
			To(Succeed())
	})

	Describe("Table", func() {
		It(
			"Should drop each missing target and keep the others",
			func(ctx SpecContext) {
				devices := open(ctx, refs(rackTarget, clearRack))
				d := refDevice{Key: "d", Rack: 2, Labels: []string{"a", "b"}}
				Expect(devices.DropDangling(ctx, db, &d)).To(BeTrue())
				Expect(d).To(Equal(refDevice{Key: "d", Labels: []string{"a"}}))
			},
		)

		It("Should report false when every target exists", func(ctx SpecContext) {
			devices := open(ctx, refs(rackTarget, clearRack))
			d := refDevice{Key: "d", Rack: 1, Labels: []string{"a"}}
			Expect(devices.DropDangling(ctx, db, &d)).To(BeFalse())
			Expect(d).To(Equal(refDevice{Key: "d", Rack: 1, Labels: []string{"a"}}))
		})

		It("Should drop a target whose key is not in text form", func(ctx SpecContext) {
			nonCanonical := func(*refDevice) []gorp.Target {
				return []gorp.Target{{Table: "rack", Key: "01"}}
			}
			devices := open(ctx, refs(nonCanonical, clearRack))
			d := refDevice{Key: "d", Rack: 1}
			Expect(devices.DropDangling(ctx, db, &d)).To(BeTrue())
			Expect(d.Rack).To(BeZero())
		})

		It(
			"Should keep a missing target whose reference cannot drop it",
			func(ctx SpecContext) {
				devices := open(ctx, refs(rackTarget, nil))
				d := refDevice{Key: "d", Rack: 2, Labels: []string{"b"}}
				Expect(devices.DropDangling(ctx, db, &d)).To(BeTrue())
				Expect(d).To(Equal(refDevice{Key: "d", Rack: 2, Labels: []string{}}))
			},
		)

		It(
			"Should skip a reference with no Detach when its table is not open",
			func(ctx SpecContext) {
				missingTable := func(*refDevice) []gorp.Target {
					return []gorp.Target{gorp.NewTarget("channel", uint32(1))}
				}
				devices := open(ctx, refs(missingTable, nil))
				d := refDevice{Key: "d"}
				Expect(devices.DropDangling(ctx, db, &d)).To(BeFalse())
			},
		)

		It("Should fail when a target's table is not open", func(ctx SpecContext) {
			missingTable := func(*refDevice) []gorp.Target {
				return []gorp.Target{gorp.NewTarget("channel", uint32(1))}
			}
			devices := open(ctx, refs(missingTable, clearRack))
			d := refDevice{Key: "d"}
			Expect(devices.DropDangling(ctx, db, &d)).
				Error().To(MatchError(`[gorp] - no open table is named "channel"`))
		})
	})

	Describe("Migration", func() {
		It(
			"Should drop the missing targets of every entry when the table opens",
			func(ctx SpecContext) {
				w := gorp.WrapWriter[string, refDevice](db)
				d1 := refDevice{Key: "d1", Rack: 2, Labels: []string{"a"}}
				d2 := refDevice{Key: "d2", Rack: 1, Labels: []string{"b"}}
				Expect(w.Set(ctx, d1)).To(Succeed())
				Expect(w.Set(ctx, d2)).To(Succeed())
				devices := open(
					ctx,
					refs(rackTarget, clearRack),
					gorp.DanglingMigration("drop_dangling"),
				)
				var entries []refDevice
				Expect(devices.NewRetrieve().Entries(&entries).Exec(ctx, db)).
					To(Succeed())
				Expect(entries).To(ConsistOf(
					refDevice{Key: "d1", Labels: []string{"a"}},
					refDevice{Key: "d2", Rack: 1, Labels: []string{}},
				))
			},
		)

		It("Should run on an open table", func(ctx SpecContext) {
			devices := open(ctx, refs(rackTarget, clearRack))
			Expect(gorp.WrapWriter[string, refDevice](db).
				Set(ctx, refDevice{Key: "d", Rack: 2})).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "dangling",
				Migrations: []gorp.Migration{devices.DanglingMigration("drop")},
			})).To(Succeed())
			var d refDevice
			Expect(devices.NewRetrieve().
				Where(gorp.MatchKeys[string, refDevice]("d")).
				Entry(&d).
				Exec(ctx, db)).To(Succeed())
			Expect(d).To(Equal(refDevice{Key: "d"}))
		})

		It("Should fail outside the config of its table", func(ctx SpecContext) {
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "dangling",
				Migrations: []gorp.Migration{gorp.DanglingMigration("drop")},
			})).To(MatchError(ContainSubstring(
				"[gorp] - dangling migration drop runs only in the config of its table",
			)))
		})
	})
})

var _ = Describe("Inline reference", func() {
	var (
		db       *gorp.DB
		groups   *gorp.Table[string, refGroup]
		swatches *gorp.Table[string, refSwatch]
		fills    *gorp.Table[string, refFill]
		// inlineErr fails each Inline when it is not nil.
		inlineErr error
		// rewritten holds each fill that the detach hook gets.
		rewritten []refFill
	)
	BeforeEach(func(ctx SpecContext) {
		inlineErr, rewritten = nil, nil
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		groups = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refGroup]{
			DB:   db,
			Name: "group",
		}))
		swatches = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refSwatch]{
			DB:   db,
			Name: "swatch",
			References: []gorp.Reference[string, refSwatch]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refSwatch, string]{
					Name:     "group",
					Table:    "group",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets:  func(s *refSwatch) []string { return []string{s.Group} },
				}),
			},
		}))
		fills = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refFill]{
			DB:   db,
			Name: "fill",
			References: []gorp.Reference[string, refFill]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refFill, string]{
					Name:     "group",
					Table:    "group",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets: func(f *refFill) []string {
						if f.Group == "" {
							return nil
						}
						return []string{f.Group}
					},
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refFill, string]{
					Name:     "swatch",
					Table:    "swatch",
					OnDelete: gorp.Inline,
					Kind:     gorp.Use,
					Targets: func(f *refFill) []string {
						if f.Swatch == "" {
							return nil
						}
						return []string{f.Swatch}
					},
					Inline: func(
						ctx context.Context,
						tx gorp.Tx,
						f *refFill,
						key string,
					) error {
						if inlineErr != nil {
							return inlineErr
						}
						var s refSwatch
						if err := gorp.NewRetrieve[string, refSwatch]().
							Where(gorp.MatchKeys[string, refSwatch](key)).
							Entry(&s).
							Exec(ctx, tx); err != nil {
							return err
						}
						f.Swatch, f.Value = "", s.Value
						return nil
					},
				}),
			},
			OnDetach: func(_ context.Context, _ gorp.Tx, f refFill) error {
				rewritten = append(rewritten, f)
				return nil
			},
		}))
		Expect(groups.NewCreate().Entry(&refGroup{Key: "g"}).Exec(ctx, db)).
			To(Succeed())
		Expect(swatches.NewCreate().
			Entries(&[]refSwatch{
				{Key: "red", Group: "g", Value: "#ff0000"},
				{Key: "blue", Group: "g", Value: "#0000ff"},
			}).
			Exec(ctx, db)).To(Succeed())
		Expect(fills.NewCreate().
			Entries(&[]refFill{
				{Key: "a", Swatch: "red"},
				{Key: "b", Swatch: "red"},
				{Key: "c", Swatch: "blue"},
				{Key: "d", Group: "g", Swatch: "red"},
			}).
			Exec(ctx, db)).To(Succeed())
	})
	retrieveFills := func(ctx SpecContext) []refFill {
		GinkgoHelper()
		var out []refFill
		Expect(fills.NewRetrieve().Entries(&out).Exec(ctx, db)).To(Succeed())
		return out
	}

	It(
		"Should replace a deleted target with the value read from it",
		func(ctx SpecContext) {
			Expect(swatches.NewDelete().
				Where(gorp.MatchKeys[string, refSwatch]("red")).
				Exec(ctx, db)).To(Succeed())
			Expect(retrieveFills(ctx)).To(ConsistOf(
				refFill{Key: "a", Value: "#ff0000"},
				refFill{Key: "b", Value: "#ff0000"},
				refFill{Key: "c", Swatch: "blue"},
				refFill{Key: "d", Group: "g", Value: "#ff0000"},
			))
			Expect(rewritten).To(ConsistOf(
				refFill{Key: "a", Value: "#ff0000"},
				refFill{Key: "b", Value: "#ff0000"},
				refFill{Key: "d", Group: "g", Value: "#ff0000"},
			))
			Expect(MustSucceed(swatches.NewRetrieve().
				Where(gorp.MatchKeys[string, refSwatch]("red")).
				Exists(ctx, db))).To(BeFalse())
		},
	)

	It(
		"Should inline the targets of a cascade in each holder the cascade keeps",
		func(ctx SpecContext) {
			Expect(groups.NewDelete().
				Where(gorp.MatchKeys[string, refGroup]("g")).
				Exec(ctx, db)).To(Succeed())
			Expect(retrieveFills(ctx)).To(ConsistOf(
				refFill{Key: "a", Value: "#ff0000"},
				refFill{Key: "b", Value: "#ff0000"},
				refFill{Key: "c", Value: "#0000ff"},
			))
			Expect(rewritten).To(HaveLen(3))
			Expect(rewritten).ToNot(ContainElement(HaveField("Key", "d")))
		},
	)

	It("Should fail the delete when an Inline fails", func(ctx SpecContext) {
		inlineErr = errors.New("inline failed")
		Expect(swatches.NewDelete().
			Where(gorp.MatchKeys[string, refSwatch]("red")).
			Exec(ctx, db)).To(MatchError(inlineErr))
		Expect(retrieveFills(ctx)).To(ContainElement(refFill{Key: "a", Swatch: "red"}))
		Expect(MustSucceed(swatches.NewRetrieve().
			Where(gorp.MatchKeys[string, refSwatch]("red")).
			Exists(ctx, db))).To(BeTrue())
	})

	It("Should refuse Inline on a target reference", func(ctx SpecContext) {
		Expect(gorp.OpenTable(ctx, gorp.TableConfig[string, refStatus]{
			DB:   db,
			Name: "status",
			References: []gorp.Reference[string, refStatus]{
				gorp.NewTargetReference(gorp.TargetReferenceConfig[string, refStatus]{
					Name:     "owner",
					Tables:   []string{"swatch"},
					OnDelete: gorp.Inline,
					Kind:     gorp.Use,
					Targets:  func(*refStatus) []gorp.Target { return nil },
				}),
			},
		})).Error().To(MatchError(ContainSubstring(
			"status.references.0.on_delete: cannot be Inline on a TargetReference",
		)))
	})
})
