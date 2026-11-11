// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"context"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat       *status.Service
		rackSvc    *rack.Service
		chassisSvc *chassis.Service
		svc        *device.Service
		tx         gorp.Tx
		rackKey    rack.Key
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
		chassisSvc = MustOpen(chassis.OpenService(ctx, chassis.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		tx = DeferClose(db.OpenTx())
	})

	newDevice := func() device.Device {
		return device.Device{
			Rack:         rackKey,
			Name:         "PXI-6255",
			Serial:       uuid.New().String(),
			ResourceName: uuid.New().String(),
			Model:        "PXI-6255",
		}
	}
	create := func(ctx context.Context) device.Device {
		GinkgoHelper()
		d := newDevice()
		Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
		return d
	}
	createChassis := func(ctx context.Context) chassis.Key {
		GinkgoHelper()
		c := chassis.Chassis{
			Rack:         rackKey,
			Name:         "PXIe-1071",
			ResourceName: uuid.New().String(),
			Model:        "PXIe-1071",
		}
		Expect(chassisSvc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
		return c.Key
	}
	retrieve := func(ctx context.Context, key device.Key) device.Device {
		GinkgoHelper()
		var d device.Device
		Expect(svc.NewRetrieve().
			Where(device.MatchKeys(key)).
			Entry(&d).
			Exec(ctx, tx)).To(Succeed())
		return d
	}
	retrieveStatus := func(ctx context.Context, key device.Key) device.Status {
		GinkgoHelper()
		var s device.Status
		Expect(stat.NewRetrieve[device.StatusDetails]().
			Where(status.MatchOwners[device.StatusDetails](device.ResourceID(key))).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*device.ServiceConfig)) {
				cfg := device.ServiceConfig{DB: db, Status: stat}
				unset(&cfg)
				Expect(device.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *device.ServiceConfig) { c.DB = nil }),
			Entry("status", "status", func(c *device.ServiceConfig) { c.Status = nil }),
		)

		It("Should let a later config override an earlier one", func() {
			cfg := device.ServiceConfig{DB: db}.Override(
				device.ServiceConfig{Status: stat},
			)
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Status).To(BeIdenticalTo(stat))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Create", func() {
		It("Should mint a key and store the device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(d.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, d.Key).Serial).To(Equal(d.Serial))
		})

		It("Should keep a key the caller sets", func(ctx SpecContext) {
			key := uuid.New()
			d := newDevice()
			d.Key = key
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(d.Key).To(Equal(key))
		})

		DescribeTable("Should reject a device without a required field",
			func(ctx SpecContext, field string, clear func(*device.Device)) {
				d := newDevice()
				clear(&d)
				Expect(svc.NewWriter(tx).Create(ctx, &d)).
					To(MatchError(ContainSubstring(field + ": required")))
			},
			Entry("name", "name", func(d *device.Device) { d.Name = "" }),
			Entry("resource name", "resource_name", func(d *device.Device) {
				d.ResourceName = ""
			}),
			Entry("model", "model", func(d *device.Device) { d.Model = "" }),
		)

		It("Should store a simulated device without a serial", func(ctx SpecContext) {
			d := newDevice()
			d.Serial = ""
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			var got device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchResourceNames(d.ResourceName)).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Key).To(Equal(d.Key))
			Expect(got.Serial).To(BeEmpty())
		})

		It("Should refuse a device in a chassis that does not exist", func(
			ctx SpecContext,
		) {
			missing := uuid.New()
			d := newDevice()
			d.Chassis = &missing
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(MatchError(fmt.Sprintf(
				"chassis: ni_chassis %s does not exist: validation error",
				missing,
			)))
		})

		It("Should write an unknown status", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			Expect(s.Name).To(Equal(d.Name))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(
				s.Details,
			).To(Equal(device.StatusDetails{Rack: rackKey, Device: d.Key}))
		})

		It(
			"Should move an existing status to a replaced device",
			func(ctx SpecContext) {
				d := create(ctx)
				s := retrieveStatus(ctx, d.Key)
				s.Variant = status.VariantSuccess
				Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
				other := rack.Rack{Name: "Other"}
				Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
				d.Rack = other.Key
				d.Name = "Replaced"
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
				got := retrieveStatus(ctx, d.Key)
				Expect(got.Name).To(Equal("Replaced"))
				Expect(got.Details.Rack).To(Equal(other.Key))
				Expect(got.Variant).To(Equal(status.VariantSuccess))
			},
		)

		It("Should keep a status that already exists", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			s.Variant = status.VariantSuccess
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(retrieveStatus(ctx, d.Key).Variant).To(Equal(status.VariantSuccess))
		})

		It("Should refuse a device on a rack that does not exist", func(
			ctx SpecContext,
		) {
			missing := rack.NewKey(1, 999)
			d := newDevice()
			d.Rack = missing
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(MatchError(fmt.Sprintf(
				"rack: rack %d does not exist: validation error",
				missing,
			)))
		})
	})

	Describe("Update", func() {
		It("Should change one field and keep the others", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Identifier = "dev1"
					return d, nil
				},
			)).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Identifier).To(Equal("dev1"))
			Expect(got.Serial).To(Equal(d.Serial))
		})

		It("Should keep the key when change clears it", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Key = uuid.Nil()
					return d, nil
				},
			)).To(Succeed())
			Expect(retrieve(ctx, d.Key).Key).To(Equal(d.Key))
		})

		It("Should rename the status with the device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Name = "Renamed"
					return d, nil
				},
			)).To(Succeed())
			Expect(retrieveStatus(ctx, d.Key).Name).To(Equal("Renamed"))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			s.Variant = status.VariantSuccess
			s.Message = "Connected"
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Rack = other.Key
					return d, nil
				},
			)).To(Succeed())
			got := retrieveStatus(ctx, d.Key)
			Expect(got.Details.Rack).To(Equal(other.Key))
			Expect(got.Variant).To(Equal(status.VariantSuccess))
			Expect(got.Message).To(Equal("Connected"))
		})

		It("Should write an unknown status when the status is missing", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(stat.NewWriter(tx).DeleteOwned(
				ctx,
				device.ResourceID(d.Key),
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Name = "Renamed"
					return d, nil
				},
			)).To(Succeed())
			s := retrieveStatus(ctx, d.Key)
			Expect(s.Name).To(Equal("Renamed"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(
				s.Details,
			).To(Equal(device.StatusDetails{Rack: rackKey, Device: d.Key}))
		})

		It("Should return the error change returns", func(ctx SpecContext) {
			d := create(ctx)
			err := errors.New("change failed")
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(device.Device) (device.Device, error) {
					return device.Device{}, err
				},
			)).To(MatchError(err))
		})

		It("Should reject an invalid result", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Name = ""
					return d, nil
				},
			)).To(MatchError(ContainSubstring("name: required")))
		})

		It("Should refuse a move to a rack that does not exist", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			missing := rack.NewKey(1, 999)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Rack = missing
					return d, nil
				},
			)).To(MatchError(fmt.Sprintf(
				"rack: rack %d does not exist: validation error",
				missing,
			)))
		})

		It("Should return not found for a missing device", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Update(
				ctx,
				uuid.New(),
				func(d device.Device) (device.Device, error) { return d, nil },
			)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Apply", func() {
		It("Should write identity fields and keep configuration", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Identifier = "dev1"
					return d, nil
				},
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, device.Update{
				Variant: device.IdentityUpdate{Identity: device.Identity{
					Rack:         rackKey,
					Serial:       "02B",
					ResourceName: "Mod2",
					Model:        "PXI-6259",
				}},
			})).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Serial).To(Equal("02B"))
			Expect(got.ResourceName).To(Equal("Mod2"))
			Expect(got.Model).To(Equal("PXI-6259"))
			Expect(got.Name).To(Equal(d.Name))
			Expect(got.Identifier).To(Equal("dev1"))
		})

		It("Should write configuration fields and keep identity", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, device.Update{
				Variant: device.ConfigurationUpdate{
					Configuration: device.Configuration{
						Name:       "Renamed",
						Identifier: "dev1",
					},
				},
			})).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Identifier).To(Equal("dev1"))
			Expect(got.Serial).To(Equal(d.Serial))
			Expect(retrieveStatus(ctx, d.Key).Name).To(Equal("Renamed"))
		})

		It("Should clear the ports that a configuration update leaves out", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.AnalogInput = device.InputPorts{
						Index:    1,
						Channels: map[string]channel.Key{"ai0": 2},
					}
					return d, nil
				},
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, device.Update{
				Variant: device.ConfigurationUpdate{
					Configuration: device.Configuration{
						Name:       "Renamed",
						Identifier: "dev1",
					},
				},
			})).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.AnalogInput.Index).To(BeZero())
			Expect(got.AnalogInput.Channels).To(BeEmpty())
		})

		It("Should return not found for a missing device", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(ctx, uuid.New(), device.Update{
				Variant: device.ConfigurationUpdate{
					Configuration: device.Configuration{Name: "Renamed"},
				},
			})).To(MatchError(query.ErrNotFound))
		})

		DescribeTable(
			"Should reject an invalid update",
			func(ctx SpecContext, update device.Update, message string) {
				d := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, d.Key, update)).
					To(MatchError(ContainSubstring(message)))
			},
			Entry("no group", device.Update{}, "no field group"),
			Entry(
				"identity",
				device.Update{Variant: device.IdentityUpdate{}},
				"required",
			),
			Entry(
				"configuration",
				device.Update{Variant: device.ConfigurationUpdate{}},
				"required",
			),
		)

		newIdentity := func() device.Identity {
			return device.Identity{
				Rack:         rackKey,
				Serial:       "02B",
				ResourceName: "Mod2",
				Model:        "PXI-6259",
			}
		}

		DescribeTable("Should reject an identity without a required field",
			func(ctx SpecContext, field string, clear func(*device.Identity)) {
				d := create(ctx)
				identity := newIdentity()
				clear(&identity)
				Expect(svc.NewWriter(tx).Apply(ctx, d.Key, device.Update{
					Variant: device.IdentityUpdate{Identity: identity},
				})).To(MatchError(ContainSubstring(field + ": required")))
			},
			Entry("resource name", "resource_name", func(i *device.Identity) {
				i.ResourceName = ""
			}),
			Entry("model", "model", func(i *device.Identity) { i.Model = "" }),
		)

		It("Should accept an identity without a serial", func(ctx SpecContext) {
			d := create(ctx)
			identity := newIdentity()
			identity.Serial = ""
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, device.Update{
				Variant: device.IdentityUpdate{Identity: identity},
			})).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Serial).To(BeEmpty())
			Expect(got.ResourceName).To(Equal("Mod2"))
		})
	})

	Describe("Migration", func() {
		It("Should require an identity update before a migrated physical device is "+
			"configured", func(ctx SpecContext) {
			mDB := DeferClose(gorp.Wrap(memkv.New()))
			mStat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: mDB}))
			MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: mDB}))
			MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: mDB}))
			mRole := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: mDB}))
			mRack := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:         mDB,
				HostKey:    1,
				Status:     mStat,
				DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, mDB, mRole)),
			}))
			MustOpen(chassis.OpenService(ctx, chassis.ServiceConfig{
				DB:     mDB,
				Status: mStat,
			}))
			old := v0.Device{
				Key:    uuid.New(),
				Rack:   mRack.EmbeddedKey,
				Name:   "PXI-6255",
				Serial: "01A",
				Model:  "PXI-6255",
			}
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Device]{DB: mDB}))
			Expect(gorp.NewCreate[v0.Key, v0.Device]().Entry(&old).Exec(ctx, mDB)).
				To(Succeed())
			mSvc := MustOpen(device.OpenService(ctx, device.ServiceConfig{
				DB:     mDB,
				Status: mStat,
			}))
			apply := func(update device.Update) error {
				return mDB.WithTx(ctx, func(tx gorp.Tx) error {
					return mSvc.NewWriter(tx).Apply(ctx, old.Key, update)
				})
			}
			configure := device.Update{Variant: device.ConfigurationUpdate{
				Configuration: device.Configuration{Name: "Renamed"},
			}}
			Expect(apply(configure)).
				To(MatchError(ContainSubstring("resource_name: required")))
			Expect(apply(device.Update{Variant: device.IdentityUpdate{
				Identity: device.Identity{
					Rack:         old.Rack,
					Serial:       old.Serial,
					ResourceName: "Mod1",
					Model:        old.Model,
				},
			}})).To(Succeed())
			Expect(apply(configure)).To(Succeed())
		})
	})

	Describe("Delete", func() {
		It("Should remove the device and its status", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Delete(ctx, d.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, tx)).To(BeFalse())
			var s device.Status
			Expect(stat.NewRetrieve[device.StatusDetails]().
				Where(status.MatchOwners[device.StatusDetails](device.ResourceID(d.Key))).
				Entry(&s).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should succeed for a missing device", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	Describe("Retrieve", func() {
		It("Should match devices by serial", func(ctx SpecContext) {
			d := create(ctx)
			create(ctx)
			var got []device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchSerials(d.Serial)).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Key).To(Equal(d.Key))
		})
	})

	Describe("Chassis", func() {
		It("Should match devices by chassis", func(ctx SpecContext) {
			key := createChassis(ctx)
			d := newDevice()
			d.Chassis = &key
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			create(ctx)
			var got []device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchChassisList(key)).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Key).To(Equal(d.Key))
		})

		It("Should delete a device and its status with its chassis", func(
			ctx SpecContext,
		) {
			key := createChassis(ctx)
			d := newDevice()
			d.Chassis = &key
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(chassisSvc.NewWriter(tx).Delete(ctx, key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, tx)).To(BeFalse())
			var s device.Status
			Expect(stat.NewRetrieve[device.StatusDetails]().
				Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
				Entry(&s).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Resources", func() {
		It("Should name each device by its name", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{d.ResourceID()})).
				To(Equal([]resource.Resource{{ID: d.ResourceID(), Name: d.Name}}))
		})

		It("Should parse the key of a device resource ID", func() {
			key := uuid.New()
			Expect(device.KeyFromResourceID(device.ResourceID(key))).To(Equal(key))
		})

		It("Should reject a key that is not a UUID", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeNiDevice,
				Key:  "not-a-uuid",
			}})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should find a device by its name", func(ctx SpecContext) {
			d := create(ctx)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: d.Name, Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(d.ResourceID()))
		})
	})
})
