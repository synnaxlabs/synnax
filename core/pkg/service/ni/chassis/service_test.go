// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package chassis_test

import (
	"context"
	"iter"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		otg        *ontology.Ontology
		stat       *status.Service
		rackSvc    *rack.Service
		devices    *device.Service
		svc        *chassis.Service
		referenced set.Set[device.Key]
		tx         gorp.Tx
		rackKey    rack.Key
	)
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    labelSvc,
		}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        groupSvc,
			HostProvider: testutil.NewStaticHostProvider(1),
			Status:       stat,
			DriverRole:   MustSucceed(racktestutil.CreateDriverRole(ctx, otg)),
		}))
		rackKey = rackSvc.EmbeddedKey
		referenced = set.New[device.Key]()
		devices = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Status:   stat,
			Referenced: func(
				_ context.Context,
				_ gorp.Tx,
				key device.Key,
			) (bool, error) {
				return referenced.Contains(key), nil
			},
		}))
		svc = MustOpen(chassis.OpenService(ctx, chassis.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Status:   stat,
			Device:   devices,
		}))
		tx = DeferClose(db.OpenTx())
	})

	create := func(ctx context.Context) chassis.Chassis {
		GinkgoHelper()
		c := chassis.Chassis{
			Rack:   rackKey,
			Name:   "PXIe-1071",
			Serial: uuid.New().String(),
			Model:  "PXIe-1071",
			Alias:  "PXI1",
		}
		Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
		return c
	}
	createModule := func(ctx context.Context, key chassis.Key) device.Device {
		GinkgoHelper()
		d := device.Device{Rack: rackKey, Name: "PXI-6255", Chassis: &key}
		Expect(devices.NewWriter(tx).Create(ctx, &d)).To(Succeed())
		return d
	}
	retrieve := func(ctx context.Context, key chassis.Key) chassis.Chassis {
		GinkgoHelper()
		var c chassis.Chassis
		Expect(svc.NewRetrieve().
			Where(chassis.MatchKeys(key)).
			Entry(&c).
			Exec(ctx, tx)).To(Succeed())
		return c
	}
	retrieveStatus := func(ctx context.Context, key chassis.Key) chassis.Status {
		GinkgoHelper()
		var s chassis.Status
		Expect(gorp.NewRetrieve[status.Key, chassis.Status]().
			Where(gorp.MatchKeys[status.Key, chassis.Status](
				chassis.OntologyID(key).String(),
			)).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	parents := func(ctx context.Context, id ontology.ID) []ontology.ID {
		GinkgoHelper()
		var res []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(id).
			TraverseTo(ontology.ParentsTraverser).
			Entries(&res).
			Exec(ctx, tx)).To(Succeed())
		return ontology.ResourceIDs(res)
	}

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*chassis.ServiceConfig)) {
				cfg := chassis.ServiceConfig{
					DB:       db,
					Ontology: otg,
					Status:   stat,
					Device:   devices,
				}
				unset(&cfg)
				Expect(chassis.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *chassis.ServiceConfig) { c.DB = nil }),
			Entry("ontology", "ontology", func(c *chassis.ServiceConfig) {
				c.Ontology = nil
			}),
			Entry("status", "status", func(c *chassis.ServiceConfig) { c.Status = nil }),
			Entry("device", "device", func(c *chassis.ServiceConfig) { c.Device = nil }),
		)
	})

	Describe("Create", func() {
		It("Should mint a key and store the chassis", func(ctx SpecContext) {
			c := create(ctx)
			Expect(c.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, c.Key).Serial).To(Equal(c.Serial))
		})

		It("Should reject a chassis without a name", func(ctx SpecContext) {
			c := chassis.Chassis{Rack: rackKey}
			Expect(svc.NewWriter(tx).Create(ctx, &c)).
				To(MatchError(ContainSubstring("name: required")))
		})

		It("Should write an unknown status", func(ctx SpecContext) {
			c := create(ctx)
			s := retrieveStatus(ctx, c.Key)
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(s.Details).To(Equal(chassis.StatusDetails{
				Rack:    rackKey,
				Chassis: c.Key,
			}))
		})

		It("Should move an existing status to a replaced chassis", func(ctx SpecContext) {
			c := create(ctx)
			s := retrieveStatus(ctx, c.Key)
			s.Variant = status.VariantSuccess
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			c.Rack = other.Key
			c.Name = "Replaced"
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			got := retrieveStatus(ctx, c.Key)
			Expect(got.Name).To(Equal("Replaced"))
			Expect(got.Details.Rack).To(Equal(other.Key))
			Expect(got.Variant).To(Equal(status.VariantSuccess))
		})

		It("Should keep a status that already exists", func(ctx SpecContext) {
			c := create(ctx)
			s := retrieveStatus(ctx, c.Key)
			s.Variant = status.VariantSuccess
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(retrieveStatus(ctx, c.Key).Variant).To(Equal(status.VariantSuccess))
		})

		It("Should parent the chassis to its rack", func(ctx SpecContext) {
			c := create(ctx)
			Expect(parents(ctx, c.OntologyID())).To(ConsistOf(rackKey.OntologyID()))
		})

		It("Should parent a module to the chassis", func(ctx SpecContext) {
			c := create(ctx)
			d := createModule(ctx, c.Key)
			Expect(parents(ctx, d.OntologyID())).To(ConsistOf(c.OntologyID()))
		})
	})

	Describe("Update", func() {
		It("Should change one field and keep the others", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				c.Key,
				func(c chassis.Chassis) (chassis.Chassis, error) {
					c.Alias = "PXI2"
					return c, nil
				},
			)).To(Succeed())
			got := retrieve(ctx, c.Key)
			Expect(got.Alias).To(Equal("PXI2"))
			Expect(got.Serial).To(Equal(c.Serial))
		})

		It("Should rename the status with the chassis", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				c.Key,
				func(c chassis.Chassis) (chassis.Chassis, error) {
					c.Name = "Renamed"
					return c, nil
				},
			)).To(Succeed())
			Expect(retrieveStatus(ctx, c.Key).Name).To(Equal("Renamed"))
		})

		It("Should move the parent edge with the rack", func(ctx SpecContext) {
			c := create(ctx)
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				c.Key,
				func(c chassis.Chassis) (chassis.Chassis, error) {
					c.Rack = other.Key
					return c, nil
				},
			)).To(Succeed())
			Expect(parents(ctx, c.OntologyID())).To(ConsistOf(other.OntologyID()))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			c := create(ctx)
			s := retrieveStatus(ctx, c.Key)
			s.Variant = status.VariantSuccess
			s.Message = "Connected"
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				c.Key,
				func(c chassis.Chassis) (chassis.Chassis, error) {
					c.Rack = other.Key
					return c, nil
				},
			)).To(Succeed())
			got := retrieveStatus(ctx, c.Key)
			Expect(got.Details.Rack).To(Equal(other.Key))
			Expect(got.Variant).To(Equal(status.VariantSuccess))
			Expect(got.Message).To(Equal("Connected"))
		})

		It("Should keep the row when its status write fails without a tx", func(
			ctx SpecContext,
		) {
			c := chassis.Chassis{Rack: rackKey, Name: "Chassis"}
			Expect(svc.NewWriter(nil).Create(ctx, &c)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, c.Key)).To(Succeed())
			})
			Expect(stat.NewWriter(nil).Delete(
				ctx,
				chassis.OntologyID(c.Key).String(),
			)).To(Succeed())
			Expect(svc.NewWriter(nil).Update(
				ctx,
				c.Key,
				func(c chassis.Chassis) (chassis.Chassis, error) {
					c.Name = "Renamed"
					return c, nil
				},
			)).To(MatchError(query.ErrNotFound))
			var got chassis.Chassis
			Expect(svc.NewRetrieve().
				Where(chassis.MatchKeys(c.Key)).
				Entry(&got).
				Exec(ctx, nil)).To(Succeed())
			Expect(got.Name).To(Equal("Chassis"))
		})

		It("Should return the error change returns", func(ctx SpecContext) {
			c := create(ctx)
			err := errors.New("change failed")
			Expect(svc.NewWriter(tx).Update(
				ctx,
				c.Key,
				func(chassis.Chassis) (chassis.Chassis, error) {
					return chassis.Chassis{}, err
				},
			)).To(MatchError(err))
		})

		It("Should reject an invalid result", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				c.Key,
				func(c chassis.Chassis) (chassis.Chassis, error) {
					c.Name = ""
					return c, nil
				},
			)).To(MatchError(ContainSubstring("name: required")))
		})

		It("Should return not found for a missing chassis", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Update(
				ctx,
				uuid.New(),
				func(c chassis.Chassis) (chassis.Chassis, error) { return c, nil },
			)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Apply", func() {
		It("Should write identity fields and keep configuration", func(
			ctx SpecContext,
		) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, chassis.Update{
				Variant: chassis.IdentityUpdate{Identity: chassis.Identity{
					Rack:   rackKey,
					Serial: "02B",
					Model:  "PXIe-1062",
				}},
			})).To(Succeed())
			got := retrieve(ctx, c.Key)
			Expect(got.Serial).To(Equal("02B"))
			Expect(got.Model).To(Equal("PXIe-1062"))
			Expect(got.Name).To(Equal(c.Name))
		})

		It("Should write configuration fields and keep identity", func(
			ctx SpecContext,
		) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, chassis.Update{
				Variant: chassis.ConfigurationUpdate{
					Configuration: chassis.Configuration{Name: "Renamed"},
				},
			})).To(Succeed())
			got := retrieve(ctx, c.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Serial).To(Equal(c.Serial))
		})

		DescribeTable("Should reject an invalid update",
			func(ctx SpecContext, update chassis.Update, message string) {
				c := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, c.Key, update)).
					To(MatchError(ContainSubstring(message)))
			},
			Entry("no group", chassis.Update{}, "no field group"),
			Entry("identity", chassis.Update{Variant: chassis.IdentityUpdate{}}, "required"),
			Entry(
				"configuration",
				chassis.Update{Variant: chassis.ConfigurationUpdate{}},
				"required",
			),
		)
	})

	Describe("Delete", func() {
		It("Should remove the chassis and its modules", func(ctx SpecContext) {
			c := create(ctx)
			d := createModule(ctx, c.Key)
			other := createModule(ctx, create(ctx).Key)
			Expect(svc.NewWriter(tx).Delete(ctx, c.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(chassis.MatchKeys(c.Key)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(devices.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(devices.NewRetrieve().
				Where(device.MatchKeys(other.Key)).
				Exists(ctx, tx)).To(BeTrue())
			Expect(gorp.NewRetrieve[status.Key, chassis.Status]().
				Where(gorp.MatchKeys[status.Key, chassis.Status](
					chassis.OntologyID(c.Key).String(),
				)).
				Exists(ctx, tx)).To(BeFalse())
		})

		It("Should refuse a chassis with a referenced module", func(ctx SpecContext) {
			c := create(ctx)
			referenced.Add(createModule(ctx, c.Key).Key)
			Expect(svc.NewWriter(tx).Delete(ctx, c.Key)).
				To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, c.Key).Key).To(Equal(c.Key))
		})

		It("Should succeed for a missing chassis", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	Describe("Ontology", func() {
		It("Should report its resource type", func() {
			Expect(svc.Type()).To(Equal(ontology.ResourceTypeNiChassis))
		})

		It("Should return one ontology ID per key", func() {
			key := uuid.New()
			Expect(chassis.OntologyIDs([]chassis.Key{key})).
				To(ConsistOf(chassis.OntologyID(key)))
		})

		Describe("OnChange", func() {
			var (
				changes    []ontology.Change
				disconnect func()
			)
			BeforeEach(func() {
				changes = nil
				disconnect = svc.OnChange(func(
					_ context.Context,
					reader iter.Seq[ontology.Change],
				) {
					changes = append(changes, slices.Collect(reader)...)
				})
				DeferCleanup(func() { disconnect() })
			})

			It("Should translate a set and a key-only delete", func(ctx SpecContext) {
				c := chassis.Chassis{Rack: rackKey, Name: "Chassis"}
				Expect(svc.NewWriter(nil).Create(ctx, &c)).To(Succeed())
				Expect(svc.NewWriter(nil).Delete(ctx, c.Key)).To(Succeed())
				id := chassis.OntologyID(c.Key).String()
				Expect(changes).To(HaveLen(2))
				Expect(changes[0].Variant).To(Equal(change.VariantSet))
				Expect(changes[0].Key).To(Equal(id))
				Expect(changes[0].Value.Name).To(Equal(c.Name))
				Expect(changes[1].Variant).To(Equal(change.VariantDelete))
				Expect(changes[1].Key).To(Equal(id))
			})

			It("Should stop after disconnecting", func(ctx SpecContext) {
				disconnect()
				c := chassis.Chassis{Rack: rackKey, Name: "Chassis"}
				Expect(svc.NewWriter(nil).Create(ctx, &c)).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(svc.NewWriter(nil).Delete(ctx, c.Key)).To(Succeed())
				})
				Expect(changes).To(BeEmpty())
			})
		})

		It("Should retrieve a chassis as a resource", func(ctx SpecContext) {
			c := create(ctx)
			res := MustSucceed(svc.RetrieveResource(ctx, c.Key.String(), tx))
			Expect(res.ID).To(Equal(chassis.OntologyID(c.Key)))
			Expect(res.Name).To(Equal(c.Name))
		})

		It("Should find a chassis by its name", func(ctx SpecContext) {
			c := create(ctx)
			res := MustSucceed(svc.Search(ctx, tx, c.Name, 10))
			Expect(res.Keys).To(ContainElement(chassis.OntologyID(c.Key)))
		})
	})
})
