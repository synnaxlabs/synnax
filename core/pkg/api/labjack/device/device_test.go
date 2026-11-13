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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apidevice "github.com/synnaxlabs/synnax/pkg/api/labjack/device"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	deviceType = resource.ID{Type: resource.TypeLabjackDevice}
	statusType = resource.ID{Type: resource.TypeStatus}
	allWrites  = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var (
		u      user.User
		serial uint32
	)
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	edit := func(d device.Device, change func(*device.Base)) device.Device {
		b := d.Base()
		change(&b)
		d.SetBase(b)
		return d
	}
	newDevice := func(overrides ...func(*device.Base)) device.Device {
		serial++
		b := device.Base{
			Rack:       svc.Rack.EmbeddedKey,
			Name:       "T7-0001",
			Serial:     serial,
			Connection: device.ConnectionUsb,
		}
		for _, override := range overrides {
			override(&b)
		}
		return device.Device{Variant: device.T7Device{Base: b}}
	}
	// grantWrites lets u write any LabJack device, its status, and the rack it sits on.
	grantWrites := func(ctx SpecContext, r rack.Key) {
		GinkgoHelper()
		grant(ctx, u, allWrites, deviceType, statusType)
		grant(ctx, u, []access.Action{access.ActionUpdate}, r.ResourceID())
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, deviceType)
	}
	createDevice := func(ctx SpecContext, overrides ...func(*device.Base)) device.Device {
		GinkgoHelper()
		d := newDevice(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.LabJack.Devices.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.LabJack.Devices.NewWriter(tx).Delete(ctx, d.Base().Key)
			})).To(Succeed())
		})
		return d
	}
	retrieveDevice := func(ctx SpecContext, key device.Key) device.Device {
		GinkgoHelper()
		var d device.Device
		Expect(svc.LabJack.Devices.NewRetrieve().
			Where(device.MatchKeys(key)).
			Entry(&d).
			Exec(ctx, db)).To(Succeed())
		return d
	}
	deleteDevice := func(ctx SpecContext, key device.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.LabJack.Devices.NewWriter(tx).Delete(ctx, key)
		})).To(Succeed())
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apidevice.ServiceConfig)) {
				cfg := apidevice.ServiceConfig{
					Device: svc.LabJack.Devices,
					Status: svc.Status,
				}
				unset(&cfg)
				Expect(apidevice.NewService(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("device", "device", func(c *apidevice.ServiceConfig) {
				c.Device = nil
			}),
			Entry("status", "status", func(c *apidevice.ServiceConfig) {
				c.Status = nil
			}),
		)

		It("Should let a later config override an earlier one", func() {
			base := apidevice.ServiceConfig{Device: svc.LabJack.Devices}
			Expect(apidevice.NewService(base, apidevice.ServiceConfig{
				Status: svc.Status,
			})).ToNot(BeNil())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when writes are granted", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{newDevice()}},
			))
			Expect(res.Devices[0].Base().Key).ToNot(Equal(uuid.Nil()))
			deleteDevice(ctx, res.Devices[0].Base().Key)
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			d := newDevice(func(b *device.Base) {
				b.Status = &device.Status{Message: "Connected"}
			})
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{d}},
			))
			Expect(res.Devices[0].Base().Status).To(BeNil())
			deleteDevice(ctx, res.Devices[0].Base().Key)
		})

		It("Should reject a device with no model", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{{}}},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{newDevice()}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should replace an existing device", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Base().Rack)
			d = edit(d, func(b *device.Base) { b.Name = "Replaced" })
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{d}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveDevice(ctx, d.Base().Key).Base().Name).To(Equal("Replaced"))
		})

		It("Should not replace a device without update access", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			replaced := edit(d, func(b *device.Base) { b.Name = "Replaced" })
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{replaced}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(
				retrieveDevice(ctx, d.Base().Key).Base().Name,
			).To(Equal(d.Base().Name))
		})
	})

	Describe("Retrieve", func() {
		It("Should reject a named device when retrieve is not granted", func(
			ctx SpecContext,
		) {
			key := createDevice(ctx).Base().Key
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Key: &filter.Set[device.Key]{Eq: &key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should filter by serial and attach the status", func(ctx SpecContext) {
			d := createDevice(ctx)
			createDevice(ctx)
			grantRetrieve(ctx)
			serial := d.Base().Serial
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{
						Serial: &filter.Ordered[uint32]{Eq: &serial},
					},
					IncludeStatus: true,
				},
			))
			Expect(res.Devices).To(HaveLen(1))
			Expect(res.Devices[0].Base().Key).To(Equal(d.Base().Key))
			Expect(res.Devices[0].Base().Status.Message).To(Equal("Status unknown"))
		})

		It("Should filter by model", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantRetrieve(ctx)
			model := device.ModelT7
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{Where: &device.Where{
					Key:   &filter.Set[device.Key]{Eq: new(d.Base().Key)},
					Model: &filter.Set[device.Model]{Eq: &model},
				}},
			))
			Expect(res.Devices).To(HaveLen(1))
			Expect(res.Devices[0].Model()).To(Equal(device.ModelT7))
		})

		It("Should leave out the devices the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			serial := createDevice(ctx).Base().Serial
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Serial: &filter.Ordered[uint32]{Eq: &serial}},
				},
			))
			Expect(res.Devices).To(BeEmpty())
		})

		It("Should find a device by its name", func(ctx SpecContext) {
			d := createDevice(ctx, func(b *device.Base) { b.Name = "Pump LabJack" })
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{SearchTerm: "pump labjack"},
			))
			Expect(lo.Map(res.Devices, func(d device.Device, _ int) device.Key {
				return d.Base().Key
			})).To(ContainElement(d.Base().Key))
		})

		It("Should cap the results at the limit", func(ctx SpecContext) {
			a := createDevice(ctx)
			b := createDevice(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Key: &filter.Set[device.Key]{
						In: []device.Key{a.Base().Key, b.Base().Key},
					}},
					Limit: new(uint32(1)),
				},
			))
			Expect(res.Devices).To(HaveLen(1))
		})

		It("Should page past the after key", func(ctx SpecContext) {
			a := createDevice(ctx)
			b := createDevice(ctx)
			grantRetrieve(ctx)
			authed := testutil.AuthedCtx(ctx, u)
			req := apidevice.RetrieveRequest{
				Where: &device.Where{Key: &filter.Set[device.Key]{
					In: []device.Key{a.Base().Key, b.Base().Key},
				}},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Devices).To(HaveLen(1))
			req.After = &apidevice.Cursor{Key: first.Devices[0].Base().Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Devices).To(HaveLen(1))
			Expect(second.Devices[0].Base().Key).
				ToNot(Equal(first.Devices[0].Base().Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					SearchTerm: d.Base().Name,
					After:      &apidevice.Cursor{Key: d.Base().Key},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return a path error for a malformed where", func(ctx SpecContext) {
			grantRetrieve(ctx)
			Expect(retrieve(testutil.AuthedCtx(ctx, u), apidevice.RetrieveRequest{
				Where: &device.Where{Name: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("where.name.regex: error parsing regexp")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should return no devices for a zero limit", func(ctx SpecContext) {
			createDevice(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{Limit: new(uint32(0))},
			))
			Expect(res.Devices).To(BeEmpty())
		})

		It("Should return no devices for a missing key", func(ctx SpecContext) {
			grantRetrieve(ctx)
			missing := uuid.New()
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Key: &filter.Set[device.Key]{Eq: &missing}},
				},
			))
			Expect(res.Devices).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		It("Should write the user's fields and keep the scanner's", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Base().Rack)
			other := rack.Rack{Name: "Other"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Rack.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			next := edit(d, func(b *device.Base) {
				b.Name = "Renamed"
				b.Identifier = "lj"
				b.Rack = other.Key
				b.Connection = device.ConnectionTCP
				b.Serial = 999
			})
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apidevice.UpdateRequest{Device: next},
			)).Error().ToNot(HaveOccurred())
			got := retrieveDevice(ctx, d.Base().Key).Base()
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Identifier).To(Equal("lj"))
			Expect(got.Rack).To(Equal(d.Base().Rack))
			Expect(got.Connection).To(Equal(d.Base().Connection))
			Expect(got.Serial).To(Equal(d.Base().Serial))
		})

		It("Should refuse a different model", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Base().Rack)
			next := device.Device{Variant: device.T4Device{Base: d.Base()}}
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apidevice.UpdateRequest{Device: next},
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("model: cannot change from t7 to t4")),
			))
			Expect(retrieveDevice(ctx, d.Base().Key).Model()).To(Equal(device.ModelT7))
		})

		It("Should reject a device with no model", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apidevice.UpdateRequest{},
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("model: required")),
			))
		})

		It("Should return not found for a missing device", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			d := edit(newDevice(), func(b *device.Base) { b.Key = uuid.New() })
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apidevice.UpdateRequest{Device: d},
			)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			next := edit(d, func(b *device.Base) { b.Name = "Renamed" })
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apidevice.UpdateRequest{Device: next},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(
				retrieveDevice(ctx, d.Base().Key).Base().Name,
			).To(Equal(d.Base().Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the device when delete is granted", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Base().Rack)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apidevice.DeleteRequest{Keys: []device.Key{d.Base().Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.LabJack.Devices.NewRetrieve().
				Where(device.MatchKeys(d.Base().Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apidevice.DeleteRequest{Keys: []device.Key{d.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveDevice(ctx, d.Base().Key).Base().Key).To(Equal(d.Base().Key))
		})
	})
})
