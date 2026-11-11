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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat    *status.Service
		rackSvc *rack.Service
		devices *device.Service
		svc     *chassis.Service
		tx      gorp.Tx
		rackKey rack.Key
	)
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		rackKey = rackSvc.EmbeddedKey
		svc = MustOpen(chassis.OpenService(ctx, chassis.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		devices = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		tx = DeferClose(db.OpenTx())
	})

	newChassis := func() chassis.Chassis {
		return chassis.Chassis{
			Rack:         rackKey,
			Name:         "PXIe-1071",
			Serial:       uuid.New().String(),
			ResourceName: uuid.New().String(),
			Model:        "PXIe-1071",
		}
	}
	create := func(ctx context.Context) chassis.Chassis {
		GinkgoHelper()
		c := newChassis()
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
		Expect(stat.NewRetrieve[chassis.StatusDetails]().
			Where(status.MatchOwners[chassis.StatusDetails](chassis.ResourceID(key))).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}

	Describe("OpenService", func() {
		DescribeTable(
			"Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*chassis.ServiceConfig)) {
				cfg := chassis.ServiceConfig{DB: db, Status: stat}
				unset(&cfg)
				Expect(chassis.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *chassis.ServiceConfig) { c.DB = nil }),
			Entry(
				"status",
				"status",
				func(c *chassis.ServiceConfig) { c.Status = nil },
			),
		)
	})

	Describe("Create", func() {
		It("Should mint a key and store the chassis", func(ctx SpecContext) {
			c := create(ctx)
			Expect(c.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, c.Key).Serial).To(Equal(c.Serial))
		})

		DescribeTable("Should reject a chassis without a required field",
			func(ctx SpecContext, field string, clear func(*chassis.Chassis)) {
				c := newChassis()
				clear(&c)
				Expect(svc.NewWriter(tx).Create(ctx, &c)).
					To(MatchError(ContainSubstring(field + ": required")))
			},
			Entry("name", "name", func(c *chassis.Chassis) { c.Name = "" }),
			Entry("resource name", "resource_name", func(c *chassis.Chassis) {
				c.ResourceName = ""
			}),
			Entry("model", "model", func(c *chassis.Chassis) { c.Model = "" }),
		)

		It("Should store a simulated chassis without a serial", func(ctx SpecContext) {
			c := newChassis()
			c.Serial = ""
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			var got chassis.Chassis
			Expect(svc.NewRetrieve().
				Where(chassis.MatchResourceNames(c.ResourceName)).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Key).To(Equal(c.Key))
			Expect(got.Serial).To(BeEmpty())
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

		It(
			"Should move an existing status to a replaced chassis",
			func(ctx SpecContext) {
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
			},
		)

		It("Should keep a status that already exists", func(ctx SpecContext) {
			c := create(ctx)
			s := retrieveStatus(ctx, c.Key)
			s.Variant = status.VariantSuccess
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(retrieveStatus(ctx, c.Key).Variant).To(Equal(status.VariantSuccess))
		})

		It("Should refuse a chassis on a rack that does not exist", func(
			ctx SpecContext,
		) {
			missing := rack.NewKey(1, 999)
			c := newChassis()
			c.Rack = missing
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(MatchError(fmt.Sprintf(
				"rack: rack %d does not exist: validation error",
				missing,
			)))
		})
	})

	Describe("Update", func() {
		It("Should change one field and keep the others", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				c.Key,
				func(c chassis.Chassis) (chassis.Chassis, error) {
					c.Model = "PXIe-1062"
					return c, nil
				},
			)).To(Succeed())
			got := retrieve(ctx, c.Key)
			Expect(got.Model).To(Equal("PXIe-1062"))
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

		It("Should keep each module on its rack when the chassis moves", func(
			ctx SpecContext,
		) {
			c := create(ctx)
			d := createModule(ctx, c.Key)
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
			Expect(retrieve(ctx, c.Key).Rack).To(Equal(other.Key))
			var got device.Device
			Expect(devices.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Rack).To(Equal(rackKey))
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

		It("Should write an unknown status when the status is missing", func(
			ctx SpecContext,
		) {
			c := create(ctx)
			Expect(stat.NewWriter(tx).DeleteOwned(
				ctx,
				chassis.ResourceID(c.Key),
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				c.Key,
				func(c chassis.Chassis) (chassis.Chassis, error) {
					c.Name = "Renamed"
					return c, nil
				},
			)).To(Succeed())
			s := retrieveStatus(ctx, c.Key)
			Expect(s.Name).To(Equal("Renamed"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(
				s.Details,
			).To(Equal(chassis.StatusDetails{Rack: rackKey, Chassis: c.Key}))
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
					Rack:         rackKey,
					Serial:       "02B",
					ResourceName: "Mod2",
					Model:        "PXIe-1062",
				}},
			})).To(Succeed())
			got := retrieve(ctx, c.Key)
			Expect(got.Serial).To(Equal("02B"))
			Expect(got.ResourceName).To(Equal("Mod2"))
			Expect(got.Model).To(Equal("PXIe-1062"))
			Expect(got.Name).To(Equal(c.Name))
		})

		It("Should accept an identity with no serial", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, chassis.Update{
				Variant: chassis.IdentityUpdate{Identity: chassis.Identity{
					Rack:         rackKey,
					ResourceName: c.ResourceName,
					Model:        c.Model,
				}},
			})).To(Succeed())
			Expect(retrieve(ctx, c.Key).Serial).To(BeEmpty())
		})

		It("Should reject an identity without a resource name", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, chassis.Update{
				Variant: chassis.IdentityUpdate{Identity: chassis.Identity{
					Rack:   rackKey,
					Serial: c.Serial,
					Model:  c.Model,
				}},
			})).To(MatchError(ContainSubstring("resource_name: required")))
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

		It("Should return not found for a missing chassis", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(ctx, uuid.New(), chassis.Update{
				Variant: chassis.ConfigurationUpdate{
					Configuration: chassis.Configuration{Name: "Renamed"},
				},
			})).To(MatchError(query.ErrNotFound))
		})

		DescribeTable(
			"Should reject an invalid update",
			func(ctx SpecContext, update chassis.Update, message string) {
				c := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, c.Key, update)).
					To(MatchError(ContainSubstring(message)))
			},
			Entry("no group", chassis.Update{}, "no field group"),
			Entry(
				"identity",
				chassis.Update{Variant: chassis.IdentityUpdate{}},
				"required",
			),
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
			var s chassis.Status
			Expect(stat.NewRetrieve[chassis.StatusDetails]().
				Where(status.MatchOwners[chassis.StatusDetails](chassis.ResourceID(c.Key))).
				Entry(&s).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			var ds device.Status
			Expect(stat.NewRetrieve[device.StatusDetails]().
				Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
				Entry(&ds).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should succeed for a missing chassis", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	Describe("Resources", func() {
		It("Should name each chassis by its name", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{c.ResourceID()})).
				To(Equal([]resource.Resource{{ID: c.ResourceID(), Name: c.Name}}))
		})

		It("Should reject a key that is not a UUID", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeNiChassis,
				Key:  "not-a-uuid",
			}})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should find a chassis by its name", func(ctx SpecContext) {
			c := create(ctx)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: c.Name, Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(c.ResourceID()))
		})
	})
})
