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
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

type refRack struct{ Key uint32 }

func (r refRack) GorpKey() uint32   { return r.Key }
func (r refRack) SetOptions() []any { return nil }

type refLabel struct{ Key string }

func (l refLabel) GorpKey() string   { return l.Key }
func (l refLabel) SetOptions() []any { return nil }

type refDevice struct {
	Key    string
	Rack   uint32
	Parent string
	Labels []string
}

func (d refDevice) GorpKey() string   { return d.Key }
func (d refDevice) SetOptions() []any { return nil }

type refOwner struct {
	Table string
	Key   string
}

type refStatus struct {
	Key   string
	Owner *refOwner
}

func (s refStatus) GorpKey() string   { return s.Key }
func (s refStatus) SetOptions() []any { return nil }

type refProject struct{ Key string }

func (p refProject) GorpKey() string   { return p.Key }
func (p refProject) SetOptions() []any { return nil }

type refGroup struct {
	Key     string
	Project string
}

func (g refGroup) GorpKey() string   { return g.Key }
func (g refGroup) SetOptions() []any { return nil }

type refSchematic struct {
	Key     string
	Project string
	Group   string
}

func (s refSchematic) GorpKey() string   { return s.Key }
func (s refSchematic) SetOptions() []any { return nil }

func deviceReferences() []*gorp.Reference[string, refDevice] {
	return []*gorp.Reference[string, refDevice]{
		gorp.NewReference(gorp.ReferenceConfig[string, refDevice]{
			Name:     "rack",
			Tables:   []string{"rack"},
			OnDelete: gorp.Restrict,
			Targets: func(d *refDevice) []gorp.Target {
				return []gorp.Target{gorp.To("rack", d.Rack)}
			},
		}),
		gorp.NewReference(gorp.ReferenceConfig[string, refDevice]{
			Name:     "parent",
			Tables:   []string{"device"},
			OnDelete: gorp.Cascade,
			Acyclic:  true,
			Targets: func(d *refDevice) []gorp.Target {
				if d.Parent == "" {
					return nil
				}
				return []gorp.Target{gorp.To("device", d.Parent)}
			},
		}),
		gorp.NewReference(gorp.ReferenceConfig[string, refDevice]{
			Name:     "labels",
			Tables:   []string{"label"},
			OnDelete: gorp.Detach,
			Targets: func(d *refDevice) []gorp.Target {
				out := make([]gorp.Target, len(d.Labels))
				for i, l := range d.Labels {
					out[i] = gorp.To("label", l)
				}
				return out
			},
			Detach: func(d *refDevice, t gorp.Target) {
				d.Labels = slices.DeleteFunc(
					slices.Clone(d.Labels),
					func(l string) bool {
						return l == t.Key
					},
				)
			},
		}),
	}
}

func statusReferences() []*gorp.Reference[string, refStatus] {
	return []*gorp.Reference[string, refStatus]{
		gorp.NewReference(gorp.ReferenceConfig[string, refStatus]{
			Name:     "owner",
			Tables:   []string{"rack", "device"},
			OnDelete: gorp.Cascade,
			Unique:   true,
			Targets: func(s *refStatus) []gorp.Target {
				if s.Owner == nil {
					return nil
				}
				return []gorp.Target{{Table: s.Owner.Table, Key: s.Owner.Key}}
			},
		}),
	}
}

var _ = Describe("Reference", func() {
	var (
		db        *gorp.DB
		racks     *gorp.Table[uint32, refRack]
		labels    *gorp.Table[string, refLabel]
		devices   *gorp.Table[string, refDevice]
		statuses  *gorp.Table[string, refStatus]
		deviceRef []*gorp.Reference[string, refDevice]
		open      = func(ctx SpecContext, opts ...gorp.Option) {
			db = gorp.Wrap(memkv.New(), append(opts, gorp.WithCodec(msgpack.Codec))...)
			racks = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "rack",
			}))
			labels = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
				DB:   db,
				Name: "label",
			}))
			deviceRef = deviceReferences()
			devices = MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
					DB:         db,
					Name:       "device",
					References: deviceRef,
				}),
			)
			statuses = MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, refStatus]{
					DB:         db,
					Name:       "status",
					References: statusReferences(),
				},
			))
		}
		createRack = func(ctx SpecContext, key uint32) {
			Expect(
				racks.NewCreate().Entry(&refRack{Key: key}).Exec(ctx, db),
			).To(Succeed())
		}
		createDevice = func(ctx SpecContext, d refDevice) error {
			return devices.NewCreate().Entry(&d).Exec(ctx, db)
		}
		deviceExists = func(ctx SpecContext, key string) bool {
			return MustSucceed(devices.NewRetrieve().
				Where(gorp.MatchKeys[string, refDevice](key)).
				Exists(ctx, db))
		}
	)
	JustAfterEach(func() {
		Expect(statuses.Close()).To(Succeed())
		Expect(devices.Close()).To(Succeed())
		Expect(labels.Close()).To(Succeed())
		Expect(racks.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})

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

		It(
			"Should leave writes through package-level queries unchecked",
			func(ctx SpecContext) {
				Expect(gorp.NewCreate[string, refDevice]().
					Entry(&refDevice{Key: "d1", Rack: 99}).
					Exec(ctx, db)).To(Succeed())
			},
		)
	})

	Describe("Filter", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It("Should find the entries holding a target", func(ctx SpecContext) {
			createRack(ctx, 1)
			for _, l := range []string{"l1", "l2"} {
				Expect(labels.NewCreate().Entry(&refLabel{Key: l}).Exec(ctx, db)).
					To(Succeed())
			}
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
				Where(deviceRef[2].Filter(gorp.To("label", "l2"))).
				Entries(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal("b"))
		})

		It("Should see writes staged in the same transaction", func(ctx SpecContext) {
			createRack(ctx, 1)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := devices.NewCreate().
					Entry(&refDevice{Key: "a", Rack: 1}).
					Exec(ctx, tx); err != nil {
					return err
				}
				Expect(deviceRef[0].Get(tx, gorp.To("rack", uint32(1)))).
					To(ConsistOf("a"))
				return nil
			})).To(Succeed())
		})
	})

	Describe("Delete", func() {
		BeforeEach(func(ctx SpecContext) { open(ctx) })

		It(
			"Should refuse to delete a target that an entry restricts",
			func(ctx SpecContext) {
				createRack(ctx, 1)
				Expect(createDevice(ctx, refDevice{Key: "d1", Rack: 1})).To(Succeed())
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, db)).To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"cannot delete rack 1 because device d1 references it through rack",
					)),
				))
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
				for _, l := range []string{"l1", "l2"} {
					Expect(labels.NewCreate().Entry(&refLabel{Key: l}).Exec(ctx, db)).
						To(Succeed())
				}
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
				keys := MustSucceed(deviceRef[2].Get(nil, gorp.To("label", "l1")))
				Expect(keys).To(BeEmpty())
			},
		)
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
				db = gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
				racks = MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
						DB:   db,
						Name: "rack",
					}),
				)
				labels = MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
						DB:   db,
						Name: "label",
					}),
				)
				devices = MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
						DB:         db,
						Name:       "device",
						References: deviceReferences(),
					}),
				)
				statuses = MustSucceed(gorp.OpenTable(
					ctx,
					gorp.TableConfig[string, refStatus]{
						DB:         db,
						Name:       "status",
						References: statusReferences(),
					},
				))
				Expect(racks.NewDelete().
					Where(gorp.MatchKeys[uint32, refRack](1)).
					Exec(ctx, db)).To(MatchError(ContainSubstring("through rack")))
			},
		)
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
			"Should reject a detach reference with no Detach function",
			func(ctx SpecContext) {
				_, err := gorp.OpenTable(ctx, gorp.TableConfig[string, refProject]{
					DB:   db,
					Name: "project",
					References: []*gorp.Reference[string, refProject]{
						gorp.NewReference(gorp.ReferenceConfig[string, refProject]{
							Name:     "label",
							Tables:   []string{"label"},
							OnDelete: gorp.Detach,
							Targets:  func(*refProject) []gorp.Target { return nil },
						}),
					},
				})
				Expect(
					err,
				).To(MatchError(ContainSubstring("must be set when on_delete is Detach")))
			},
		)
	})
})

var _ = Describe("Reference closure", func() {
	It(
		"Should not restrict a delete on a dependent the delete removes",
		func(ctx SpecContext) {
			db := gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec))
			projects := MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[string, refProject]{
					DB:   db,
					Name: "project",
				}),
			)
			groups := MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[string, refGroup]{
					DB:   db,
					Name: "group",
					References: []*gorp.Reference[string, refGroup]{
						gorp.NewReference(gorp.ReferenceConfig[string, refGroup]{
							Name:     "project",
							Tables:   []string{"project"},
							OnDelete: gorp.Cascade,
							Targets: func(g *refGroup) []gorp.Target {
								return []gorp.Target{gorp.To("project", g.Project)}
							},
						}),
					},
				}),
			)
			schematics := MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, refSchematic]{
					DB:   db,
					Name: "schematic",
					References: []*gorp.Reference[string, refSchematic]{
						gorp.NewReference(gorp.ReferenceConfig[string, refSchematic]{
							Name:     "project",
							Tables:   []string{"project"},
							OnDelete: gorp.Cascade,
							Targets: func(s *refSchematic) []gorp.Target {
								return []gorp.Target{gorp.To("project", s.Project)}
							},
						}),
						gorp.NewReference(gorp.ReferenceConfig[string, refSchematic]{
							Name:     "group",
							Tables:   []string{"group"},
							OnDelete: gorp.Restrict,
							Targets: func(s *refSchematic) []gorp.Target {
								return []gorp.Target{gorp.To("group", s.Group)}
							},
						}),
					},
				},
			))
			DeferCleanup(func() {
				Expect(schematics.Close()).To(Succeed())
				Expect(groups.Close()).To(Succeed())
				Expect(projects.Close()).To(Succeed())
				Expect(db.Close()).To(Succeed())
			})
			Expect(projects.NewCreate().Entry(&refProject{Key: "p"}).Exec(ctx, db)).
				To(Succeed())
			Expect(
				groups.NewCreate().
					Entry(&refGroup{Key: "g", Project: "p"}).
					Exec(ctx, db),
			).
				To(Succeed())
			Expect(schematics.NewCreate().
				Entry(&refSchematic{Key: "s", Project: "p", Group: "g"}).
				Exec(ctx, db)).To(Succeed())
			Expect(groups.NewDelete().
				Where(gorp.MatchKeys[string, refGroup]("g")).
				Exec(ctx, db)).To(MatchError(ContainSubstring("through group")))
			Expect(projects.NewDelete().
				Where(gorp.MatchKeys[string, refProject]("p")).
				Exec(ctx, db)).To(Succeed())
			Expect(schematics.NewRetrieve().
				Where(gorp.MatchKeys[string, refSchematic]("s")).
				Exists(ctx, db)).To(BeFalse())
		},
	)
})
