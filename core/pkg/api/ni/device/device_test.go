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
	apidevice "github.com/synnaxlabs/synnax/pkg/api/ni/device"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var deviceType = ontology.ID{Type: ontology.ResourceTypeNiDevice}

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	create := func(ctx SpecContext) device.Device {
		GinkgoHelper()
		d := device.Device{
			Rack:   svc.Rack.EmbeddedKey,
			Name:   "PXI-6255",
			Serial: uuid.New().String(),
			Model:  "PXI-6255",
		}
		Expect(svc.NI.Devices.NewWriter(nil).Create(ctx, &d)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NI.Devices.NewWriter(nil).Delete(ctx, d.Key)).To(Succeed())
		})
		return d
	}
	retrieve := func(ctx SpecContext, key device.Key) device.Device {
		GinkgoHelper()
		var d device.Device
		Expect(svc.NI.Devices.NewRetrieve().
			Where(device.MatchKeys(key)).
			Entry(&d).
			Exec(ctx, nil)).To(Succeed())
		return d
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apidevice.ServiceConfig)) {
				cfg := apidevice.ServiceConfig{
					Device: svc.NI.Devices,
					Status: svc.Status,
					RBAC:   svc.RBAC,
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
			Entry("rbac", "rbac", func(c *apidevice.ServiceConfig) { c.RBAC = nil }),
		)

		It("Should let a later config override an earlier one", func() {
			base := apidevice.ServiceConfig{Device: svc.NI.Devices}
			Expect(apidevice.NewService(base, apidevice.ServiceConfig{
				Status: svc.Status,
				RBAC:   svc.RBAC,
			})).ToNot(BeNil())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when create is granted", func(ctx SpecContext) {
			grant(ctx, access.ActionCreate, user.OntologyID(u.Key), deviceType)
			res := MustSucceed(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.CreateRequest{Devices: []device.Device{{
					Rack:   svc.Rack.EmbeddedKey,
					Name:   "Module",
					Serial: uuid.New().String(),
				}}},
			))
			Expect(res.Devices[0].Key).ToNot(Equal(uuid.Nil()))
			Expect(svc.NI.Devices.NewWriter(nil).Delete(ctx, res.Devices[0].Key)).
				To(Succeed())
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.CreateRequest{Devices: []device.Device{{Name: "Module"}}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Create over an existing device", func() {
		It("Should require update access on the device", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionCreate, user.OntologyID(u.Key), deviceType)
			d.Name = "Replaced"
			Expect(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.CreateRequest{Devices: []device.Device{d}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieve(ctx, d.Key).Name).To(Equal("PXI-6255"))
		})

		It("Should replace with update access alone", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), deviceType)
			d.Name = "Replaced"
			Expect(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.CreateRequest{Devices: []device.Device{d}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieve(ctx, d.Key).Name).To(Equal("Replaced"))
		})

		It("Should require create access for a new key beside it", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), deviceType)
			Expect(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.CreateRequest{Devices: []device.Device{
					d,
					{Rack: d.Rack, Name: "Module"},
				}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Retrieve", func() {
		It("Should reject the request when retrieve is not granted", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(apiSvc.Retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{Keys: []device.Key{d.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should filter by serial and attach the status", func(ctx SpecContext) {
			d := create(ctx)
			create(ctx)
			grant(ctx, access.ActionRetrieve, user.OntologyID(u.Key), deviceType)
			res := MustSucceed(apiSvc.Retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Serials:       []string{d.Serial},
					IncludeStatus: true,
				},
			))
			Expect(res.Devices).To(HaveLen(1))
			Expect(res.Devices[0].Key).To(Equal(d.Key))
			Expect(res.Devices[0].Status.Message).To(Equal("Status unknown"))
		})

		It("Should return not found unless the request ignores it", func(
			ctx SpecContext,
		) {
			grant(ctx, access.ActionRetrieve, user.OntologyID(u.Key), deviceType)
			key := uuid.New()
			Expect(apiSvc.Retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{Keys: []device.Key{key}},
			)).Error().To(MatchError(query.ErrNotFound))
			res := MustSucceed(apiSvc.Retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Keys:           []device.Key{key},
					IgnoreNotFound: true,
				},
			))
			Expect(res.Devices).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		It("Should write only the configuration group", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), deviceType)
			Expect(apiSvc.Update(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.UpdateRequest{
					Key: d.Key,
					Update: device.Update{Variant: device.ConfigurationUpdate{
						Configuration: device.Configuration{
							Name:       "Renamed",
							Identifier: "dev1",
						},
					}},
				},
			)).Error().ToNot(HaveOccurred())
			got := retrieve(ctx, d.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Serial).To(Equal(d.Serial))
		})

		It("Should write only the identity group", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), deviceType)
			Expect(apiSvc.Update(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.UpdateRequest{
					Key: d.Key,
					Update: device.Update{Variant: device.IdentityUpdate{
						Identity: device.Identity{Rack: d.Rack, Serial: "02B"},
					}},
				},
			)).Error().ToNot(HaveOccurred())
			got := retrieve(ctx, d.Key)
			Expect(got.Serial).To(Equal("02B"))
			Expect(got.Name).To(Equal(d.Name))
		})

		It("Should reject a request with no group", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), deviceType)
			Expect(apiSvc.Update(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.UpdateRequest{Key: d.Key},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(apiSvc.Update(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.UpdateRequest{Key: d.Key},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Delete", func() {
		It("Should remove the device when delete is granted", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionDelete, user.OntologyID(u.Key), deviceType)
			Expect(apiSvc.Delete(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.DeleteRequest{Keys: []device.Key{d.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.NI.Devices.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, nil)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(apiSvc.Delete(
				testutil.AuthedCtx(ctx, u),
				nil,
				apidevice.DeleteRequest{Keys: []device.Key{d.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
})
