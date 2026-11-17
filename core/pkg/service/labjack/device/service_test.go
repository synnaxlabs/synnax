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
	"github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat    *status.Service
		rackSvc *rack.Service
		svc     *device.Service
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
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		tx = DeferClose(db.OpenTx())
	})

	var serial uint32
	nextSerial := func() uint32 {
		serial++
		return serial
	}
	edit := func(d device.Device, change func(*device.Device)) device.Device {
		change(&d)
		return d
	}
	create := func(ctx context.Context) device.Device {
		GinkgoHelper()
		d := device.Device{
			Rack:       rackKey,
			Name:       "T7-0001",
			Serial:     nextSerial(),
			Connection: device.ConnectionUSB,
			Model:      device.ModelT7,
			Ports:      device.Ports{Variant: device.T7Ports{}},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
		return d
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

		It("Should store the port maps of a device", func(ctx SpecContext) {
			d := device.Device{Variant: device.T7Device{
				Base: device.Base{
					Rack:       rackKey,
					Name:       "T7-0002",
					Serial:     nextSerial(),
					Connection: device.ConnectionUSB,
				},
				AnalogInput: map[device.T7AnalogInputPort]channel.Key{
					device.T7AnalogInputPortAin0:  1,
					device.T7AnalogInputPortAin48: 2,
				},
				AnalogOutput: map[device.AnalogOutputPort]device.CommandStatePair{
					device.AnalogOutputPortDac1: {Command: 3, State: 4},
				},
				DigitalInput: map[device.T7DigitalPort]channel.Key{
					device.T7DigitalPortDio0: 5,
				},
				DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{
					device.T7DigitalPortDio22: {Command: 6, State: 7},
				},
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			got := retrieve(ctx, d.Base().Key)
			Expect(got.Variant).To(Equal(d.Variant))
		})

		It("Should keep a key the caller sets", func(ctx SpecContext) {
			key := uuid.New()
			d := device.Device{
				Key:        key,
				Rack:       rackKey,
				Name:       "T7-0001",
				Serial:     470010001,
				Connection: device.ConnectionUSB,
				Model:      device.ModelT7,
				Ports:      device.Ports{Variant: device.T7Ports{}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(d.Key).To(Equal(key))
		})

		DescribeTable(
			"Should reject an invalid device",
			func(ctx SpecContext, change func(*device.Device), message string) {
				d := device.Device{
					Rack:       rackKey,
					Name:       "T7-0001",
					Serial:     470010001,
					Connection: device.ConnectionUSB,
					Model:      device.ModelT7,
					Ports:      device.Ports{Variant: device.T7Ports{}},
				}
				change(&d)
				Expect(svc.NewWriter(tx).Create(ctx, &d)).
					To(MatchError(ContainSubstring(message)))
			},
			Entry(
				"no name",
				func(d *device.Device) { d.Name = "" },
				"name: required",
			),
			Entry(
				"no serial",
				func(d *device.Device) { d.Serial = 0 },
				"serial: must be non-zero",
			),
			Entry(
				"no model",
				func(d *device.Device) { d.Model = "" },
				"model",
			),
			Entry(
				"no ports",
				func(d *device.Device) { d.Ports = device.Ports{} },
				"ports.model: required",
			),
			Entry(
				"ports for another model",
				func(d *device.Device) { d.Ports = device.Ports{Variant: device.T4Ports{}} },
				"ports: are for t4, not t7",
			),
			Entry(
				"unknown connection",
				func(d *device.Device) { d.Connection = "LJM_ctTCP" },
				"invalid connection",
			),
		)

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
				d = edit(d, func(b *device.Device) {
					b.Rack = other.Key
					b.Name = "Replaced"
				})
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
				got := retrieveStatus(ctx, d.Key)
				Expect(got.Name).To(Equal("Replaced"))
				Expect(got.Details.Rack).To(Equal(other.Key))
				Expect(got.Variant).To(Equal(status.VariantSuccess))
			},
		)

		It("Should move the status when it moves the device to another rack", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			s.Variant = status.VariantSuccess
			s.Message = "Connected"
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			moved := edit(d, func(b *device.Device) { b.Rack = other.Key })
			Expect(svc.NewWriter(tx).Create(ctx, &moved)).To(Succeed())
			got := retrieveStatus(ctx, d.Key)
			Expect(got.Details.Rack).To(Equal(other.Key))
			Expect(got.Variant).To(Equal(status.VariantSuccess))
			Expect(got.Message).To(Equal("Connected"))
		})

		It("Should keep a status that already exists", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			s.Variant = status.VariantSuccess
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(
				retrieveStatus(ctx, d.Key).Variant,
			).To(Equal(status.VariantSuccess))
		})

		It("Should refuse a device on a rack that does not exist", func(
			ctx SpecContext,
		) {
			missing := rack.NewKey(1, 999)
			d := device.Device{
				Rack:       missing,
				Name:       "T7-0001",
				Serial:     470010001,
				Connection: device.ConnectionUSB,
				Model:      device.ModelT7,
				Ports:      device.Ports{Variant: device.T7Ports{}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(MatchError(fmt.Sprintf(
				"rack: rack %d does not exist: validation error",
				missing,
			)))
		})
	})

	Describe("Apply", func() {
		configuration := func(ports device.PortsVariant) device.Update {
			return device.Update{Variant: device.ConfigurationUpdate{
				Configuration: device.Configuration{
					Name:       "Renamed",
					Identifier: "dev1",
					ReadIndex:  7,
					Ports:      device.Ports{Variant: ports},
				},
			}}
		}
		identity := func(d device.Device, change func(*device.Identity)) device.Update {
			i := device.Identity{
				Rack:       d.Rack,
				Serial:     d.Serial,
				Connection: d.Connection,
				Model:      d.Model,
			}
			change(&i)
			return device.Update{Variant: device.IdentityUpdate{Identity: i}}
		}

		It("Should write the configuration and keep the identity", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				d.Key,
				configuration(device.T7Ports{Mux80Attached: true}),
			)).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Identifier).To(Equal("dev1"))
			Expect(got.ReadIndex).To(BeEquivalentTo(7))
			Expect(got.Ports).To(Equal(device.Ports{
				Variant: device.T7Ports{Mux80Attached: true},
			}))
			Expect(got.Rack).To(Equal(d.Rack))
			Expect(got.Serial).To(Equal(d.Serial))
			Expect(got.Connection).To(Equal(d.Connection))
		})

		It("Should write the identity and keep the configuration", func(
			ctx SpecContext,
		) {
			d := edit(create(ctx), func(b *device.Device) { b.Identifier = "dev1" })
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, identity(
				d,
				func(i *device.Identity) {
					i.Rack = other.Key
					i.Serial = d.Serial + 1000
					i.Connection = device.ConnectionWifi
				},
			))).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Rack).To(Equal(other.Key))
			Expect(got.Serial).To(Equal(d.Serial + 1000))
			Expect(got.Connection).To(Equal(device.ConnectionWifi))
			Expect(got.Name).To(Equal(d.Name))
			Expect(got.Identifier).To(Equal("dev1"))
			Expect(retrieveStatus(ctx, d.Key).Details.Rack).To(Equal(other.Key))
		})

		It("Should refuse a move to a missing rack", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, identity(
				d,
				func(i *device.Identity) { i.Rack = rackKey + 1000 },
			))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("does not exist")),
			))
			Expect(retrieve(ctx, d.Key).Rack).To(Equal(rackKey))
		})

		It("Should rename the status with the device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				d.Key,
				configuration(device.T7Ports{}),
			)).To(Succeed())
			Expect(retrieveStatus(ctx, d.Key).Name).To(Equal("Renamed"))
		})

		It("Should write an unknown status when the status is missing", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(stat.NewWriter(tx).DeleteOwned(
				ctx,
				device.ResourceID(d.Key),
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				d.Key,
				configuration(device.T7Ports{}),
			)).To(Succeed())
			s := retrieveStatus(ctx, d.Key)
			Expect(s.Name).To(Equal("Renamed"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(
				s.Details,
			).To(Equal(device.StatusDetails{Rack: rackKey, Device: d.Key}))
		})

		It("Should reject an invalid configuration", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, device.Update{
				Variant: device.ConfigurationUpdate{Configuration: device.Configuration{
					Ports: device.Ports{Variant: device.T7Ports{}},
				}},
			})).To(MatchError(ContainSubstring("name: required")))
		})

		DescribeTable(
			"Should refuse a different model",
			func(ctx SpecContext, update func(device.Device) device.Update, message string) {
				d := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, d.Key, update(d))).
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(message)),
					))
				Expect(retrieve(ctx, d.Key).Model).To(Equal(device.ModelT7))
			},
			Entry("in the ports of a configuration", func(device.Device) device.Update {
				return configuration(device.T4Ports{})
			}, "ports: are for t4, not t7"),
			Entry("in an identity", func(d device.Device) device.Update {
				return identity(
					d,
					func(i *device.Identity) { i.Model = device.ModelT4 },
				)
			}, "model: cannot change from t7 to t4"),
		)

		It("Should reject an update with no group", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, device.Update{})).
				To(MatchError(validate.ErrValidation))
		})

		It("Should return not found for a missing device", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				uuid.New(),
				configuration(device.T7Ports{}),
			)).To(MatchError(query.ErrNotFound))
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

		It("Should keep a rack that holds a device from deletion", func(
			ctx SpecContext,
		) {
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			d := edit(create(ctx), func(b *device.Device) { b.Rack = other.Key })
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, other.Key)).
				To(MatchError(ContainSubstring("labjack_device")))
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

	Describe("ResourceID", func() {
		It("Should name a device by its key under the LabJack device type", func() {
			key := uuid.New()
			Expect(device.ResourceID(key)).To(Equal(resource.ID{
				Type: resource.TypeLabjackDevice,
				Key:  key.String(),
			}))
		})

		It("Should match the ID of a stored device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(device.ResourceID(d.Key)).To(Equal(d.ResourceID()))
		})
	})

	Describe("Resources", func() {
		It("Should name each device by its name", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{d.ResourceID()})).
				To(Equal([]resource.Resource{{ID: d.ResourceID(), Name: d.Name}}))
		})

		It("Should reject a key that is not a UUID", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeLabjackDevice,
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
