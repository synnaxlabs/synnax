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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
		It("Should reject a config missing the chassis service", func() {
			Expect(apichassis.NewService(apichassis.ServiceConfig{
				Status: svc.Status,
				RBAC:   svc.RBAC,
			})).Error().To(MatchError(ContainSubstring("chassis: must be non-nil")))
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

	Describe("Retrieve", func() {
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
