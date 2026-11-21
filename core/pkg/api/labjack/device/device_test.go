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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var deviceType = resource.ID{Type: resource.TypeLabjackDevice}

var _ = Describe("Service", func() {
	var (
		u      user.User
		serial uint32
	)
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	newDevice := func(overrides ...func(*device.Base)) device.Device {
		serial++
		b := device.Base{
			Key:        uuid.New(),
			Rack:       svc.Rack.EmbeddedKey,
			Name:       "T7-0001",
			Serial:     serial,
			Connection: device.ConnectionUSB,
		}
		for _, override := range overrides {
			override(&b)
		}
		return device.Device{Variant: device.T7Device{Base: b}}
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, deviceType)
	}
	createDevice := func(ctx SpecContext, overrides ...func(*device.Base)) device.Device {
		GinkgoHelper()
		d := newDevice(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewCreate[device.Key, device.Device]().Entry(&d).Exec(ctx, tx)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewDelete[device.Key, device.Device]().
					Where(gorp.MatchKeys[device.Key, device.Device](d.Base().Key)).
					Exec(ctx, tx)
			})).To(Succeed())
		})
		return d
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
			st := device.Status{
				Owner:   new(d.ResourceID()),
				Name:    d.Base().Name,
				Time:    telem.Now(),
				Variant: status.VariantSuccess,
				Message: "Running",
				Details: device.StatusDetails{Device: d.Base().Key},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
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
			Expect(res.Devices[0].Base().Status.Message).To(Equal("Running"))
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
})
