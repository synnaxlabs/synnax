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
	"github.com/synnaxlabs/gorp"
	apichassis "github.com/synnaxlabs/synnax/pkg/api/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	chassisType = resource.ID{Type: resource.TypeNiChassis}
	deviceType  = resource.ID{Type: resource.TypeNiDevice}
	statusType  = resource.ID{Type: resource.TypeStatus}
	allWrites   = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	// grantWrites lets u write any NI chassis, its status, and the rack it sits on.
	grantWrites := func(ctx SpecContext, r rack.Key) {
		GinkgoHelper()
		grant(ctx, u, allWrites, chassisType, statusType)
		grant(ctx, u, []access.Action{access.ActionUpdate}, r.ResourceID())
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, chassisType)
	}
	createChassis := func(
		ctx SpecContext,
		overrides ...func(*chassis.Chassis),
	) chassis.Chassis {
		GinkgoHelper()
		d := chassis.Chassis{
			Rack:         svc.Rack.EmbeddedKey,
			Name:         "PXIe-1071",
			Serial:       uuid.New().String(),
			ResourceName: uuid.New().String(),
			Model:        "PXIe-1071",
		}
		for _, override := range overrides {
			override(&d)
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NI.Chassis.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NI.Chassis.NewWriter(tx).Delete(ctx, d.Key)
			})).To(Succeed())
		})
		return d
	}
	createModule := func(ctx SpecContext, c chassis.Chassis) device.Device {
		GinkgoHelper()
		d := device.Device{
			Rack:         c.Rack,
			Name:         "PXI-6255",
			ResourceName: uuid.New().String(),
			Model:        "PXI-6255",
			Chassis:      &c.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NI.Devices.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NI.Devices.NewWriter(tx).Delete(ctx, d.Key)
			})).To(Succeed())
		})
		return d
	}
	retrieveChassis := func(ctx SpecContext, key chassis.Key) chassis.Chassis {
		GinkgoHelper()
		var d chassis.Chassis
		Expect(svc.NI.Chassis.NewRetrieve().
			Where(chassis.MatchKeys(key)).
			Entry(&d).
			Exec(ctx, db)).To(Succeed())
		return d
	}
	deleteChassis := func(ctx SpecContext, key chassis.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NI.Chassis.NewWriter(tx).Delete(ctx, key)
		})).To(Succeed())
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apichassis.ServiceConfig)) {
				cfg := apichassis.ServiceConfig{
					Chassis: svc.NI.Chassis,
					Status:  svc.Status,
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
		)

		It("Should let a later config override an earlier one", func() {
			base := apichassis.ServiceConfig{Chassis: svc.NI.Chassis}
			Expect(apichassis.NewService(base, apichassis.ServiceConfig{
				Status: svc.Status,
			})).ToNot(BeNil())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when writes are granted", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apichassis.CreateRequest{Chassis: []chassis.Chassis{{
					Rack:         svc.Rack.EmbeddedKey,
					Name:         "Chassis",
					ResourceName: uuid.New().String(),
					Model:        "PXIe-1071",
				}}},
			))
			Expect(res.Chassis[0].Key).ToNot(Equal(uuid.Nil()))
			deleteChassis(ctx, res.Chassis[0].Key)
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apichassis.CreateRequest{Chassis: []chassis.Chassis{{
					Rack:         svc.Rack.EmbeddedKey,
					Name:         "Chassis",
					ResourceName: uuid.New().String(),
					Model:        "PXIe-1071",
					Status:       &chassis.Status{Message: "Connected"},
				}}},
			))
			Expect(res.Chassis[0].Status).To(BeNil())
			deleteChassis(ctx, res.Chassis[0].Key)
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apichassis.CreateRequest{Chassis: []chassis.Chassis{{
					Rack:         svc.Rack.EmbeddedKey,
					Name:         "Chassis",
					ResourceName: uuid.New().String(),
					Model:        "PXIe-1071",
				}}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject an invalid chassis", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apichassis.CreateRequest{Chassis: []chassis.Chassis{{
					Rack:  svc.Rack.EmbeddedKey,
					Model: "PXIe-1071",
				}}},
			)).Error().To(MatchError(ContainSubstring("name: required")))
		})

		It("Should replace an existing chassis", func(ctx SpecContext) {
			d := createChassis(ctx)
			grantWrites(ctx, d.Rack)
			d.Name = "Replaced"
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apichassis.CreateRequest{Chassis: []chassis.Chassis{d}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveChassis(ctx, d.Key).Name).To(Equal("Replaced"))
		})

		It("Should not replace a chassis without update access", func(
			ctx SpecContext,
		) {
			d := createChassis(ctx)
			d.Name = "Replaced"
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apichassis.CreateRequest{Chassis: []chassis.Chassis{d}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveChassis(ctx, d.Key).Name).To(Equal("PXIe-1071"))
		})
	})

	Describe("Retrieve", func() {
		It("Should reject a named chassis when retrieve is not granted", func(
			ctx SpecContext,
		) {
			d := createChassis(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					Where: &chassis.Where{Key: &filter.Set[chassis.Key]{Eq: &d.Key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should leave out the chassis the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			d := createChassis(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					Where: &chassis.Where{Serial: &filter.String{Eq: &d.Serial}},
				},
			))
			Expect(res.Chassis).To(BeEmpty())
		})

		It("Should filter by serial and attach the status", func(ctx SpecContext) {
			d := createChassis(ctx)
			createChassis(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					Where: &chassis.Where{
						Serial: &filter.String{Eq: &d.Serial},
					},
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
				d := createChassis(ctx)
				other := rack.Rack{Name: "Other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				createChassis(ctx, func(o *chassis.Chassis) {
					o.Rack = other.Key
					o.Name = "Other"
					o.Model = "Other"
				})
				grantRetrieve(ctx)
				var req apichassis.RetrieveRequest
				filter(&req, d)
				res := MustSucceed(retrieve(testutil.AuthedCtx(ctx, u), req))
				Expect(res.Chassis).To(HaveLen(1))
				Expect(res.Chassis[0].Key).To(Equal(d.Key))
			},
			Entry("names", func(r *apichassis.RetrieveRequest, d chassis.Chassis) {
				r.Where = &chassis.Where{Name: &filter.String{Eq: &d.Name}}
			}),
			Entry("models", func(r *apichassis.RetrieveRequest, d chassis.Chassis) {
				r.Where = &chassis.Where{Model: &filter.String{Eq: &d.Model}}
			}),
			Entry("racks", func(r *apichassis.RetrieveRequest, d chassis.Chassis) {
				r.Where = &chassis.Where{Rack: &filter.Set[rack.Key]{Eq: &d.Rack}}
			}),
			Entry(
				"search term",
				func(r *apichassis.RetrieveRequest, d chassis.Chassis) {
					r.SearchTerm = d.Name
				},
			),
		)

		It("Should select a chassis by resource name", func(ctx SpecContext) {
			d := createChassis(ctx)
			createChassis(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					Where: &chassis.Where{
						ResourceName: &filter.String{Eq: &d.ResourceName},
					},
				},
			))
			Expect(res.Chassis).To(HaveLen(1))
			Expect(res.Chassis[0].Key).To(Equal(d.Key))
		})

		It("Should cap the results at the limit", func(ctx SpecContext) {
			d := createChassis(ctx)
			createChassis(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					Where: &chassis.Where{Name: &filter.String{Eq: &d.Name}},
					Limit: new(uint32(1)),
				},
			))
			Expect(res.Chassis).To(HaveLen(1))
		})

		It("Should page past the after key", func(ctx SpecContext) {
			a := createChassis(ctx)
			b := createChassis(ctx)
			grantRetrieve(ctx)
			authed := testutil.AuthedCtx(ctx, u)
			req := apichassis.RetrieveRequest{
				Where: &chassis.Where{
					Name: &filter.String{In: []string{a.Name, b.Name}},
				},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Chassis).To(HaveLen(1))
			req.After = &apichassis.Cursor{Key: first.Chassis[0].Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Chassis).To(HaveLen(1))
			Expect(second.Chassis[0].Key).ToNot(Equal(first.Chassis[0].Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createChassis(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					SearchTerm: d.Name,
					After:      &apichassis.Cursor{Key: d.Key},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return a path error for a malformed where", func(ctx SpecContext) {
			grantRetrieve(ctx)
			Expect(retrieve(testutil.AuthedCtx(ctx, u), apichassis.RetrieveRequest{
				Where: &chassis.Where{Model: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("where.model.regex: error parsing regexp")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should return no chassis for a zero limit", func(ctx SpecContext) {
			createChassis(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{Limit: new(uint32(0))},
			))
			Expect(res.Chassis).To(BeEmpty())
		})

		It("Should return no chassis for a missing key", func(ctx SpecContext) {
			grantRetrieve(ctx)
			missing := uuid.New()
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apichassis.RetrieveRequest{
					Where: &chassis.Where{Key: &filter.Set[chassis.Key]{Eq: &missing}},
				},
			))
			Expect(res.Chassis).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		rename := func(key chassis.Key) apichassis.UpdateRequest {
			return apichassis.UpdateRequest{
				Key: key,
				Update: chassis.Update{Variant: chassis.ConfigurationUpdate{
					Configuration: chassis.Configuration{
						Name: "Renamed",
					},
				}},
			}
		}

		It("Should write only the configuration group", func(ctx SpecContext) {
			d := createChassis(ctx)
			grantWrites(ctx, d.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(d.Key))).
				Error().ToNot(HaveOccurred())
			got := retrieveChassis(ctx, d.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Serial).To(Equal(d.Serial))
		})

		It("Should write only the identity group", func(ctx SpecContext) {
			d := createChassis(ctx)
			grantWrites(ctx, d.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apichassis.UpdateRequest{
					Key: d.Key,
					Update: chassis.Update{Variant: chassis.IdentityUpdate{
						Identity: chassis.Identity{
							Rack:         d.Rack,
							Serial:       "02B",
							ResourceName: d.ResourceName,
							Model:        d.Model,
						},
					}},
				},
			)).Error().ToNot(HaveOccurred())
			got := retrieveChassis(ctx, d.Key)
			Expect(got.Serial).To(Equal("02B"))
			Expect(got.Name).To(Equal(d.Name))
		})

		It("Should reject a request with no group", func(ctx SpecContext) {
			d := createChassis(ctx)
			grantWrites(ctx, d.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apichassis.UpdateRequest{Key: d.Key},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			d := createChassis(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(d.Key))).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveChassis(ctx, d.Key).Name).To(Equal(d.Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the chassis when delete is granted", func(ctx SpecContext) {
			d := createChassis(ctx)
			grantWrites(ctx, d.Rack)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apichassis.DeleteRequest{Keys: []chassis.Key{d.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.NI.Chassis.NewRetrieve().
				Where(chassis.MatchKeys(d.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should require delete access on each module", func(ctx SpecContext) {
			c := createChassis(ctx)
			d := createModule(ctx, c)
			grantWrites(ctx, c.Rack)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apichassis.DeleteRequest{Keys: []chassis.Key{c.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(svc.NI.Devices.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, db)).To(BeTrue())
		})

		It("Should delete modules with access to them", func(ctx SpecContext) {
			c := createChassis(ctx)
			d := createModule(ctx, c)
			grantWrites(ctx, c.Rack)
			grant(
				ctx,
				u,
				[]access.Action{access.ActionUpdate, access.ActionDelete},
				deviceType,
			)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apichassis.DeleteRequest{Keys: []chassis.Key{c.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.NI.Devices.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			d := createChassis(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apichassis.DeleteRequest{Keys: []chassis.Key{d.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveChassis(ctx, d.Key).Key).To(Equal(d.Key))
		})
	})
})
