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

	edit := func(d device.Device, change func(*device.Device)) device.Device {
		change(&d)
		return d
	}
	newDevice := func(overrides ...func(*device.Device)) device.Device {
		serial++
		b := device.Device{
			Rack:       svc.Rack.EmbeddedKey,
			Name:       "T7-0001",
			Serial:     serial,
			Connection: device.ConnectionUSB,
			Model:      device.ModelT7,
			Ports:      device.Ports{Variant: device.T7Ports{}},
		}
		for _, override := range overrides {
			override(&b)
		}
		return b
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
	createDevice := func(ctx SpecContext, overrides ...func(*device.Device)) device.Device {
		GinkgoHelper()
		d := newDevice(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.LabJack.Devices.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.LabJack.Devices.NewWriter(tx).Delete(ctx, d.Key)
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
			Expect(res.Devices[0].Key).ToNot(Equal(uuid.Nil()))
			deleteDevice(ctx, res.Devices[0].Key)
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			d := newDevice(func(b *device.Device) {
				b.Status = &device.Status{Message: "Connected"}
			})
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{d}},
			))
			Expect(res.Devices[0].Status).To(BeNil())
			deleteDevice(ctx, res.Devices[0].Key)
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
			grantWrites(ctx, d.Rack)
			d = edit(d, func(b *device.Device) { b.Name = "Replaced" })
			MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{d}},
			))
			Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Replaced"))
		})

		It("Should not replace a device without update access", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			grant(ctx, u, []access.Action{access.ActionCreate}, deviceType, statusType)
			grant(
				ctx,
				u,
				[]access.Action{access.ActionUpdate},
				statusType,
				d.Rack.ResourceID(),
			)
			replaced := edit(d, func(b *device.Device) { b.Name = "Replaced" })
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{replaced}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(
				retrieveDevice(ctx, d.Key).Name,
			).To(Equal(d.Name))
		})
	})

	Describe("Retrieve", func() {
		It("Should page by name in both directions, ignoring case", func(
			ctx SpecContext,
		) {
			prefix := uuid.New().String()
			named := func(name string) device.Key {
				GinkgoHelper()
				return createDevice(
					ctx,
					func(d *device.Device) { d.Name = prefix + name },
				).Key
			}
			beta := named("beta")
			alpha := named("Alpha")
			gamma := named("gamma")
			grantRetrieve(ctx)
			page := func(order apidevice.Order) []device.Key {
				GinkgoHelper()
				var (
					keys  []device.Key
					after *apidevice.Cursor
				)
				for {
					res := MustSucceed(retrieve(
						testutil.AuthedCtx(ctx, u),
						apidevice.RetrieveRequest{
							Where: &device.Where{Name: &filter.String{
								In: []string{
									prefix + "beta",
									prefix + "Alpha",
									prefix + "gamma",
								},
							}},
							Order: &order,
							After: after,
							Limit: new(uint32(1)),
						},
					))
					if len(res.Devices) == 0 {
						return keys
					}
					last := res.Devices[len(res.Devices)-1]
					keys = append(keys, last.Key)
					after = &apidevice.Cursor{Key: last.Key, Name: &last.Name}
				}
			}
			Expect(page(apidevice.Order{Field: apidevice.OrderFieldName})).
				To(Equal([]device.Key{alpha, beta, gamma}))
			Expect(page(apidevice.Order{
				Field:      apidevice.OrderFieldName,
				Descending: true,
			})).To(Equal([]device.Key{gamma, beta, alpha}))
		})

		It("Should reject a named device when retrieve is not granted", func(
			ctx SpecContext,
		) {
			key := createDevice(ctx).Key
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
			serial := d.Serial
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
			Expect(res.Devices[0].Key).To(Equal(d.Key))
			Expect(res.Devices[0].Status.Message).To(Equal("Status unknown"))
		})

		It("Should filter by model", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantRetrieve(ctx)
			model := device.ModelT7
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{Where: &device.Where{
					Key:   &filter.Set[device.Key]{Eq: new(d.Key)},
					Model: &filter.Set[device.Model]{Eq: &model},
				}},
			))
			Expect(res.Devices).To(HaveLen(1))
			Expect(res.Devices[0].Model).To(Equal(device.ModelT7))
		})

		It("Should leave out the devices the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			serial := createDevice(ctx).Serial
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Serial: &filter.Ordered[uint32]{Eq: &serial}},
				},
			))
			Expect(res.Devices).To(BeEmpty())
		})

		It("Should find a device by its name", func(ctx SpecContext) {
			d := createDevice(ctx, func(b *device.Device) { b.Name = "Pump LabJack" })
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{SearchTerm: "pump labjack"},
			))
			Expect(lo.Map(res.Devices, func(d device.Device, _ int) device.Key {
				return d.Key
			})).To(ContainElement(d.Key))
		})

		It("Should cap the results at the limit", func(ctx SpecContext) {
			a := createDevice(ctx)
			b := createDevice(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Key: &filter.Set[device.Key]{
						In: []device.Key{a.Key, b.Key},
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
					In: []device.Key{a.Key, b.Key},
				}},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Devices).To(HaveLen(1))
			req.After = &apidevice.Cursor{Key: first.Devices[0].Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Devices).To(HaveLen(1))
			Expect(second.Devices[0].Key).
				ToNot(Equal(first.Devices[0].Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					SearchTerm: d.Name,
					After:      &apidevice.Cursor{Key: d.Key},
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

	Describe("Dispatch", func() {
		set := func(path string, value any) device.Action {
			return device.NewSetAction(device.SetPayload{
				Path:  []string{path},
				Value: value,
			})
		}
		dispatch := func(
			key device.Key,
			acts ...device.Action,
		) apidevice.DispatchRequest {
			return apidevice.DispatchRequest{Key: key, Actions: acts}
		}

		It("Should apply the actions and keep the other fields", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				dispatch(d.Key, set("name", "Renamed"), set("identifier", "lj")),
			)).Error().ToNot(HaveOccurred())
			got := retrieveDevice(ctx, d.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Identifier).To(Equal("lj"))
			Expect(got.Serial).To(Equal(d.Serial))
		})

		It("Should refuse ports for another model", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				dispatch(d.Key, set("ports", device.Ports{Variant: device.T4Ports{}})),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("ports: are for t4, not t7")),
			))
			Expect(retrieveDevice(ctx, d.Key).Model).To(Equal(device.ModelT7))
		})

		It("Should return not found for a missing device", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				dispatch(uuid.New(), set("name", "Missing")),
			)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				dispatch(d.Key, set("name", "Renamed")),
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveDevice(ctx, d.Key).Name).To(Equal(d.Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the device when delete is granted", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Rack)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apidevice.DeleteRequest{Keys: []device.Key{d.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.LabJack.Devices.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apidevice.DeleteRequest{Keys: []device.Key{d.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveDevice(ctx, d.Key).Key).To(Equal(d.Key))
		})
	})
})
