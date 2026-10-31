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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apichassis "github.com/synnaxlabs/synnax/pkg/api/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var chassisType = ontology.ID{Type: ontology.ResourceTypeNiChassis}

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	create := func(ctx SpecContext) chassis.Chassis {
		GinkgoHelper()
		d := chassis.Chassis{
			Rack:   svc.Rack.EmbeddedKey,
			Name:   "PXIe-1071",
			Serial: uuid.New().String(),
			Model:  "PXIe-1071",
		}
		Expect(svc.NI.Chassis.NewWriter(nil).Create(ctx, &d)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NI.Chassis.NewWriter(nil).Delete(ctx, d.Key)).To(Succeed())
		})
		return d
	}
	retrieve := func(ctx SpecContext, key chassis.Key) chassis.Chassis {
		GinkgoHelper()
		var d chassis.Chassis
		Expect(svc.NI.Chassis.NewRetrieve().
			Where(chassis.MatchKeys(key)).
			Entry(&d).
			Exec(ctx, nil)).To(Succeed())
		return d
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apichassis.ServiceConfig)) {
				cfg := apichassis.ServiceConfig{
					Chassis: svc.NI.Chassis,
					Status:  svc.Status,
					Device:  svc.NI.Devices,
					RBAC:    svc.RBAC,
				}
				unset(&cfg)
				Expect(apichassis.NewService(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("chassis", "chassis", func(c *apichassis.ServiceConfig) {
				c.Chassis = nil
			}),
			Entry("status", "status", func(c *apichassis.ServiceConfig) {
				c.Status = nil
			}),
			Entry("device", "device", func(c *apichassis.ServiceConfig) {
				c.Device = nil
			}),
			Entry("rbac", "rbac", func(c *apichassis.ServiceConfig) { c.RBAC = nil }),
		)

		It("Should let a later config override an earlier one", func() {
			base := apichassis.ServiceConfig{Chassis: svc.NI.Chassis}
			Expect(apichassis.NewService(base, apichassis.ServiceConfig{
				Status: svc.Status,
				Device: svc.NI.Devices,
				RBAC:   svc.RBAC,
			})).ToNot(BeNil())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when create is granted", func(ctx SpecContext) {
			grant(ctx, access.ActionCreate, user.OntologyID(u.Key), chassisType)
			res := MustSucceed(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.CreateRequest{Chassis: []chassis.Chassis{{
					Rack:   svc.Rack.EmbeddedKey,
					Name:   "Module",
					Serial: uuid.New().String(),
				}}},
			))
			Expect(res.Chassis[0].Key).ToNot(Equal(uuid.Nil()))
			Expect(svc.NI.Chassis.NewWriter(nil).Delete(ctx, res.Chassis[0].Key)).
				To(Succeed())
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.CreateRequest{Chassis: []chassis.Chassis{{Name: "Module"}}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Create over an existing chassis", func() {
		It("Should require update access on the chassis", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionCreate, user.OntologyID(u.Key), chassisType)
			d.Name = "Replaced"
			Expect(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.CreateRequest{Chassis: []chassis.Chassis{d}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieve(ctx, d.Key).Name).To(Equal("PXIe-1071"))
		})

		It("Should replace with update access alone", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), chassisType)
			d.Name = "Replaced"
			Expect(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.CreateRequest{Chassis: []chassis.Chassis{d}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieve(ctx, d.Key).Name).To(Equal("Replaced"))
		})

		It("Should require create access for a new key beside it", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), chassisType)
			Expect(apiSvc.Create(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.CreateRequest{Chassis: []chassis.Chassis{
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
				apichassis.RetrieveRequest{Keys: []chassis.Key{d.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should filter by serial and attach the status", func(ctx SpecContext) {
			d := create(ctx)
			create(ctx)
			grant(ctx, access.ActionRetrieve, user.OntologyID(u.Key), chassisType)
			res := MustSucceed(apiSvc.Retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					Serials:       []string{d.Serial},
					IncludeStatus: true,
				},
			))
			Expect(res.Chassis).To(HaveLen(1))
			Expect(res.Chassis[0].Key).To(Equal(d.Key))
			Expect(res.Chassis[0].Status.Message).To(Equal("Status unknown"))
		})

		DescribeTable(
			"Should filter by each request field",
			func(ctx SpecContext, filter func(*apichassis.RetrieveRequest, chassis.Chassis)) {
				d := create(ctx)
				create(ctx)
				grant(ctx, access.ActionRetrieve, user.OntologyID(u.Key), chassisType)
				req := apichassis.RetrieveRequest{Keys: []chassis.Key{d.Key}}
				filter(&req, d)
				res := MustSucceed(apiSvc.Retrieve(testutil.AuthedCtx(ctx, u), req))
				Expect(res.Chassis).To(HaveLen(1))
				Expect(res.Chassis[0].Key).To(Equal(d.Key))
			},
			Entry("names", func(r *apichassis.RetrieveRequest, d chassis.Chassis) {
				r.Names = []string{d.Name}
			}),
			Entry("models", func(r *apichassis.RetrieveRequest, d chassis.Chassis) {
				r.Models = []string{d.Model}
			}),
			Entry("racks", func(r *apichassis.RetrieveRequest, d chassis.Chassis) {
				r.Racks = []rack.Key{d.Rack}
			}),
			Entry(
				"search term",
				func(r *apichassis.RetrieveRequest, d chassis.Chassis) {
					r.SearchTerm = d.Name
				},
			),
			Entry("limit", func(r *apichassis.RetrieveRequest, _ chassis.Chassis) {
				r.Limit = 1
			}),
		)

		It("Should skip chassis before the offset", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionRetrieve, user.OntologyID(u.Key), chassisType)
			res := MustSucceed(apiSvc.Retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{Keys: []chassis.Key{d.Key}, Offset: 1},
			))
			Expect(res.Chassis).To(BeEmpty())
		})

		It("Should return not found unless the request ignores it", func(
			ctx SpecContext,
		) {
			grant(ctx, access.ActionRetrieve, user.OntologyID(u.Key), chassisType)
			key := uuid.New()
			Expect(apiSvc.Retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{Keys: []chassis.Key{key}},
			)).Error().To(MatchError(query.ErrNotFound))
			res := MustSucceed(apiSvc.Retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					Keys:           []chassis.Key{key},
					IgnoreNotFound: true,
				},
			))
			Expect(res.Chassis).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		It("Should write only the configuration group", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), chassisType)
			Expect(apiSvc.Update(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.UpdateRequest{
					Key: d.Key,
					Update: chassis.Update{Variant: chassis.ConfigurationUpdate{
						Configuration: chassis.Configuration{
							Name: "Renamed",
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
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), chassisType)
			Expect(apiSvc.Update(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.UpdateRequest{
					Key: d.Key,
					Update: chassis.Update{Variant: chassis.IdentityUpdate{
						Identity: chassis.Identity{Rack: d.Rack, Serial: "02B"},
					}},
				},
			)).Error().ToNot(HaveOccurred())
			got := retrieve(ctx, d.Key)
			Expect(got.Serial).To(Equal("02B"))
			Expect(got.Name).To(Equal(d.Name))
		})

		It("Should reject a request with no group", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionUpdate, user.OntologyID(u.Key), chassisType)
			Expect(apiSvc.Update(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.UpdateRequest{Key: d.Key},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(apiSvc.Update(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.UpdateRequest{Key: d.Key},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Delete", func() {
		It("Should remove the chassis when delete is granted", func(ctx SpecContext) {
			d := create(ctx)
			grant(ctx, access.ActionDelete, user.OntologyID(u.Key), chassisType)
			Expect(apiSvc.Delete(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.DeleteRequest{Keys: []chassis.Key{d.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.NI.Chassis.NewRetrieve().
				Where(chassis.MatchKeys(d.Key)).
				Exists(ctx, nil)).To(BeFalse())
		})

		It("Should require delete access on each module", func(ctx SpecContext) {
			c := create(ctx)
			d := device.Device{Rack: c.Rack, Name: "Module", Chassis: &c.Key}
			Expect(svc.NI.Devices.NewWriter(nil).Create(ctx, &d)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NI.Devices.NewWriter(nil).Delete(ctx, d.Key)).To(Succeed())
			})
			grant(ctx, access.ActionDelete, user.OntologyID(u.Key), chassisType)
			Expect(apiSvc.Delete(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.DeleteRequest{Keys: []chassis.Key{c.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(svc.NI.Devices.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, nil)).To(BeTrue())
		})

		It("Should delete modules with access to them", func(ctx SpecContext) {
			c := create(ctx)
			d := device.Device{Rack: c.Rack, Name: "Module", Chassis: &c.Key}
			Expect(svc.NI.Devices.NewWriter(nil).Create(ctx, &d)).To(Succeed())
			grant(
				ctx,
				access.ActionDelete,
				user.OntologyID(u.Key),
				chassisType,
				ontology.ID{Type: ontology.ResourceTypeNiDevice},
			)
			Expect(apiSvc.Delete(
				testutil.AuthedCtx(ctx, u),
				nil,
				apichassis.DeleteRequest{Keys: []chassis.Key{c.Key}},
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
				apichassis.DeleteRequest{Keys: []chassis.Key{d.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
})
