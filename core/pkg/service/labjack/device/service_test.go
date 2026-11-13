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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/device"
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
	edit := func(d device.Device, change func(*device.Base)) device.Device {
		b := d.Base()
		change(&b)
		d.SetBase(b)
		return d
	}
	create := func(ctx context.Context) device.Device {
		GinkgoHelper()
		d := device.Device{Variant: device.T7Device{Base: device.Base{
			Rack:       rackKey,
			Name:       "T7-0001",
			Serial:     nextSerial(),
			Connection: device.ConnectionUsb,
		}}}
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
			Expect(d.Base().Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, d.Base().Key).Base().Serial).To(Equal(d.Base().Serial))
		})

		It("Should keep a key the caller sets", func(ctx SpecContext) {
			key := uuid.New()
			d := device.Device{Variant: device.T7Device{Base: device.Base{
				Key:        key,
				Rack:       rackKey,
				Name:       "T7-0001",
				Serial:     470010001,
				Connection: device.ConnectionUsb,
			}}}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(d.Base().Key).To(Equal(key))
		})

		DescribeTable(
			"Should reject an invalid device",
			func(ctx SpecContext, change func(*device.Device), message string) {
				d := device.Device{Variant: device.T7Device{Base: device.Base{
					Rack:       rackKey,
					Name:       "T7-0001",
					Serial:     470010001,
					Connection: device.ConnectionUsb,
				}}}
				change(&d)
				Expect(svc.NewWriter(tx).Create(ctx, &d)).
					To(MatchError(ContainSubstring(message)))
			},
			Entry(
				"no name",
				func(d *device.Device) { *d = edit(*d, func(b *device.Base) { b.Name = "" }) },
				"name: required",
			),
			Entry(
				"no serial",
				func(d *device.Device) { *d = edit(*d, func(b *device.Base) { b.Serial = 0 }) },
				"serial: must be non-zero",
			),
			Entry(
				"no model",
				func(d *device.Device) { d.Variant = nil },
				"model: required",
			),
			Entry(
				"unknown connection",
				func(d *device.Device) { *d = edit(*d, func(b *device.Base) { b.Connection = "LJM_ctWIFI" }) },
				"invalid connection",
			),
		)

		It("Should write an unknown status", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Base().Key)
			Expect(s.Name).To(Equal(d.Base().Name))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(
				s.Details,
			).To(Equal(device.StatusDetails{Rack: rackKey, Device: d.Base().Key}))
		})

		It(
			"Should move an existing status to a replaced device",
			func(ctx SpecContext) {
				d := create(ctx)
				s := retrieveStatus(ctx, d.Base().Key)
				s.Variant = status.VariantSuccess
				Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
				other := rack.Rack{Name: "Other"}
				Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
				d = edit(d, func(b *device.Base) {
					b.Rack = other.Key
					b.Name = "Replaced"
				})
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
				got := retrieveStatus(ctx, d.Base().Key)
				Expect(got.Name).To(Equal("Replaced"))
				Expect(got.Details.Rack).To(Equal(other.Key))
				Expect(got.Variant).To(Equal(status.VariantSuccess))
			},
		)

		It("Should keep a status that already exists", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Base().Key)
			s.Variant = status.VariantSuccess
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(
				retrieveStatus(ctx, d.Base().Key).Variant,
			).To(Equal(status.VariantSuccess))
		})

		It("Should refuse a device on a rack that does not exist", func(
			ctx SpecContext,
		) {
			missing := rack.NewKey(1, 999)
			d := device.Device{Variant: device.T7Device{Base: device.Base{
				Rack:       missing,
				Name:       "T7-0001",
				Serial:     470010001,
				Connection: device.ConnectionUsb,
			}}}
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
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return edit(d, func(b *device.Base) { b.Identifier = "dev1" }), nil
				},
			)).To(Succeed())
			got := retrieve(ctx, d.Base().Key)
			Expect(got.Base().Identifier).To(Equal("dev1"))
			Expect(got.Base().Serial).To(Equal(d.Base().Serial))
		})

		It("Should keep the key when change clears it", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return edit(d, func(b *device.Base) { b.Key = uuid.Nil() }), nil
				},
			)).To(Succeed())
			Expect(retrieve(ctx, d.Base().Key).Base().Key).To(Equal(d.Base().Key))
		})

		It("Should rename the status with the device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return edit(d, func(b *device.Base) { b.Name = "Renamed" }), nil
				},
			)).To(Succeed())
			Expect(retrieveStatus(ctx, d.Base().Key).Name).To(Equal("Renamed"))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Base().Key)
			s.Variant = status.VariantSuccess
			s.Message = "Connected"
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return edit(d, func(b *device.Base) { b.Rack = other.Key }), nil
				},
			)).To(Succeed())
			got := retrieveStatus(ctx, d.Base().Key)
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
				device.ResourceID(d.Base().Key),
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return edit(d, func(b *device.Base) { b.Name = "Renamed" }), nil
				},
			)).To(Succeed())
			s := retrieveStatus(ctx, d.Base().Key)
			Expect(s.Name).To(Equal("Renamed"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(
				s.Details,
			).To(Equal(device.StatusDetails{Rack: rackKey, Device: d.Base().Key}))
		})

		It("Should return the error change returns", func(ctx SpecContext) {
			d := create(ctx)
			err := errors.New("change failed")
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(device.Device) (device.Device, error) {
					return device.Device{}, err
				},
			)).To(MatchError(err))
		})

		It("Should reject an invalid result", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return edit(d, func(b *device.Base) { b.Name = "" }), nil
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
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return edit(d, func(b *device.Base) { b.Rack = missing }), nil
				},
			)).To(MatchError(fmt.Sprintf(
				"rack: rack %d does not exist: validation error",
				missing,
			)))
		})

		It("Should change the model of a device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return device.Device{Variant: device.T4Device{Base: d.Base()}}, nil
				},
			)).To(Succeed())
			got := retrieve(ctx, d.Base().Key)
			Expect(got.Model()).To(Equal(device.ModelT4))
			Expect(got.Base().Serial).To(Equal(d.Base().Serial))
		})

		It("Should reject a result with no model", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(device.Device) (device.Device, error) { return device.Device{}, nil },
			),
			).To(MatchError(ContainSubstring("model: required")))
		})

		It("Should return not found for a missing device", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Update(
				ctx,
				uuid.New(),
				func(d device.Device) (device.Device, error) { return d, nil },
			)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the device and its status", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Delete(ctx, d.Base().Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(d.Base().Key)).
				Exists(ctx, tx)).To(BeFalse())
			var s device.Status
			Expect(stat.NewRetrieve[device.StatusDetails]().
				Where(status.MatchOwners[device.StatusDetails](device.ResourceID(d.Base().Key))).
				Entry(&s).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should keep a rack that holds a device from deletion", func(
			ctx SpecContext,
		) {
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Base().Key,
				func(d device.Device) (device.Device, error) {
					return edit(d, func(b *device.Base) { b.Rack = other.Key }), nil
				},
			)).To(Succeed())
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
				Where(device.MatchSerials(d.Base().Serial)).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Base().Key).To(Equal(d.Base().Key))
		})
	})

	Describe("Resources", func() {
		It("Should name each device by its name", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{d.ResourceID()})).
				To(Equal([]resource.Resource{{ID: d.ResourceID(), Name: d.Base().Name}}))
		})

		It("Should parse the key of a device resource ID", func() {
			key := uuid.New()
			Expect(device.KeyFromResourceID(device.ResourceID(key))).To(Equal(key))
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
				search.Query[resource.ID]{Term: d.Base().Name, Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(d.ResourceID()))
		})
	})
})
