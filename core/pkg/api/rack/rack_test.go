// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func retrieveRack(ctx SpecContext, key rack.Key) rack.Rack {
	GinkgoHelper()
	var r rack.Rack
	Expect(svc.Rack.NewRetrieve().
		Where(rack.MatchKeys(key)).
		Entry(&r).
		Exec(ctx, nil)).To(Succeed())
	return r
}

func createRackWithStatus(ctx SpecContext, name string) rack.Rack {
	GinkgoHelper()
	r := rack.Rack{Name: name}
	Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
	Expect(svc.Status.NewWriter(nil).Set(ctx, &rack.Status{
		Owner:   new(r.ResourceID()),
		Name:    r.Name,
		Time:    telem.Now(),
		Variant: status.VariantSuccess,
		Message: "Rack is running",
	})).To(Succeed())
	return r
}

var _ = Describe("Create", func() {
	It("Should reply with the status of a new rack", func(ctx SpecContext) {
		u := createUser(ctx)
		rackType := resource.ID{Type: resource.TypeRack}
		grant(ctx, access.ActionCreate, u.ResourceID(), rackType)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), rackType)
		res := MustSucceed(apiSvc.Create(
			AuthedCtx(ctx, u), nil,
			apirack.CreateRequest{Racks: []rack.Rack{{Name: "New"}}},
		))
		Expect(res.Racks).To(HaveLen(1))
		r := res.Racks[0]
		Expect(r.Status).ToNot(BeNil())
		Expect(r.Status.Owner).To(HaveValue(Equal(r.ResourceID())))
		Expect(r.Status.Variant).To(Equal(status.VariantWarning))
	})
	It(
		"Should leave out the status when create is the only grant",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(
				ctx,
				access.ActionCreate,
				u.ResourceID(),
				resource.ID{Type: resource.TypeRack},
			)
			res := MustSucceed(apiSvc.Create(
				AuthedCtx(ctx, u), nil,
				apirack.CreateRequest{Racks: []rack.Rack{{Name: "New"}}},
			))
			Expect(res.Racks).To(HaveLen(1))
			Expect(res.Racks[0].Key.IsZero()).To(BeFalse())
			Expect(res.Racks[0].Status).To(BeNil())
		},
	)
	It(
		"Should leave out the status of a rack the caller can only update",
		func(ctx SpecContext) {
			u := createUser(ctx)
			hidden := createRackWithStatus(ctx, "Hidden")
			grant(ctx, access.ActionUpdate, u.ResourceID(), hidden.ResourceID())
			res := MustSucceed(apiSvc.Create(
				AuthedCtx(ctx, u), nil,
				apirack.CreateRequest{Racks: []rack.Rack{hidden}},
			))
			Expect(res.Racks).To(HaveLen(1))
			Expect(res.Racks[0].Status).To(BeNil())
		},
	)
	It("Should check read access on each rack in a batch", func(ctx SpecContext) {
		u := createUser(ctx)
		readable := createRackWithStatus(ctx, "Readable")
		hidden := createRackWithStatus(ctx, "Hidden")
		grant(
			ctx, access.ActionUpdate, u.ResourceID(),
			readable.ResourceID(), hidden.ResourceID(),
		)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), readable.ResourceID())
		res := MustSucceed(apiSvc.Create(
			AuthedCtx(ctx, u), nil,
			apirack.CreateRequest{Racks: []rack.Rack{readable, hidden}},
		))
		Expect(res.Racks).To(HaveLen(2))
		Expect(res.Racks[0].Status.Message).To(Equal("Rack is running"))
		Expect(res.Racks[1].Status).To(BeNil())
	})
	It("Should reply with the live status of an existing rack", func(ctx SpecContext) {
		r := createRackWithStatus(ctx, "Existing")
		res := MustSucceed(apiSvc.Create(
			SubjectCtx(ctx, r.ResourceID()), nil,
			apirack.CreateRequest{Racks: []rack.Rack{r}},
		))
		Expect(res.Racks[0].Status.Message).To(Equal("Rack is running"))
	})
	It("Should reply with the new name of a renamed rack", func(ctx SpecContext) {
		r := createRackWithStatus(ctx, "Old")
		r.Name = "New"
		res := MustSucceed(apiSvc.Create(
			SubjectCtx(ctx, r.ResourceID()), nil,
			apirack.CreateRequest{Racks: []rack.Rack{r}},
		))
		Expect(res.Racks[0].Status.Name).To(Equal("New"))
		Expect(res.Racks[0].Status.Message).To(Equal("Rack is running"))
	})
	It("Should let a rack with the Driver role save itself", func(ctx SpecContext) {
		r := rack.Rack{Name: "Self"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		r.Integrations = []string{"ni"}
		Expect(apiSvc.Create(
			SubjectCtx(ctx, r.ResourceID()), nil,
			apirack.CreateRequest{Racks: []rack.Rack{r}},
		)).Error().ToNot(HaveOccurred())
		Expect(retrieveRack(ctx, r.Key).Integrations).To(Equal([]string{"ni"}))
	})
	It(
		"Should let a rack with the Driver role save itself twice",
		func(ctx SpecContext) {
			r := rack.Rack{Name: "Self"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			Expect(apiSvc.Create(
				SubjectCtx(ctx, r.ResourceID()), nil,
				apirack.CreateRequest{Racks: []rack.Rack{r, r}},
			)).Error().ToNot(HaveOccurred())
		},
	)
	It(
		"Should not let a rack with the Driver role create a rack",
		func(ctx SpecContext) {
			r := rack.Rack{Name: "Self"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			Expect(apiSvc.Create(
				SubjectCtx(ctx, r.ResourceID()), nil,
				apirack.CreateRequest{Racks: []rack.Rack{{Name: "Other"}}},
			)).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should require update to save an existing rack", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(
			ctx,
			access.ActionCreate,
			u.ResourceID(),
			resource.ID{Type: resource.TypeRack},
		)
		r := rack.Rack{Name: "Existing"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		Expect(apiSvc.Create(
			AuthedCtx(ctx, u), nil,
			apirack.CreateRequest{Racks: []rack.Rack{r}},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Rename", func() {
	It("Should rename the rack when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Before", Integrations: []string{"ni"}}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: r.Key, Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveRack(ctx, r.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Integrations).To(Equal([]string{"ni"}))
	})
	It("Should keep the embedded flag of the embedded rack", func(ctx SpecContext) {
		u := createUser(ctx)
		key := svc.Rack.EmbeddedKey
		grant(ctx, access.ActionUpdate, u.ResourceID(), key.ResourceID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: key, Name: "Renamed Embedded"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveRack(ctx, key)
		Expect(renamed.Name).To(Equal("Renamed Embedded"))
		Expect(renamed.Embedded).To(BeTrue())
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Before"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: r.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveRack(ctx, r.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing rack", func(ctx SpecContext) {
		u := createUser(ctx)
		key := rack.NewKey(1, 60000)
		grant(ctx, access.ActionUpdate, u.ResourceID(), key.ResourceID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Before"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: r.Key, Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveRack(ctx, r.Key).Name).To(Equal("Before"))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should attach only each rack's own status", func(ctx SpecContext) {
		u := createUser(ctx)
		withStatus := rack.Rack{Name: "With Status"}
		withoutStatus := rack.Rack{Name: "Without Status"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &withStatus)).To(Succeed())
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &withoutStatus)).To(Succeed())
		w := svc.Status.NewWriter(nil)
		Expect(w.DeleteOwned(ctx, withoutStatus.ResourceID())).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.ResourceID(),
			withStatus.ResourceID(), withoutStatus.ResourceID(),
		)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apirack.RetrieveRequest{
			Where: &rack.Where{Key: &filter.Set[rack.Key]{
				In: []rack.Key{withoutStatus.Key, withStatus.Key},
			}},
			IncludeStatus: true,
		}))
		Expect(res.Racks).To(HaveLen(2))
		Expect(res.Racks[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Racks[0].Status).To(BeNil())
		Expect(res.Racks[1].Key).To(Equal(withStatus.Key))
		Expect(res.Racks[1].Status.Owner).To(HaveValue(Equal(withStatus.ResourceID())))
	})

	It("Should treat a missing key as a non-match", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Present"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), r.ResourceID())
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apirack.RetrieveRequest{
			Where: &rack.Where{Key: &filter.Set[rack.Key]{
				In: []rack.Key{r.Key, rack.NewKey(9, 9)},
			}},
		}))
		Expect(res.Racks).To(HaveLen(1))
		Expect(res.Racks[0].Key).To(Equal(r.Key))
	})

	It("Should return a path error for an invalid regex", func(ctx SpecContext) {
		u := createUser(ctx)
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apirack.RetrieveRequest{
			Where: &rack.Where{Name: &filter.String{Regex: new("(a")}},
		})).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"where.name.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})

	It(
		"Should not let the status API break the reads of a rack",
		func(ctx SpecContext) {
			u := createUser(ctx)
			r := createRackWithStatus(ctx, "Target")
			statusType := resource.ID{Type: resource.TypeStatus}
			grant(ctx, access.ActionCreate, u.ResourceID(), statusType)
			grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
			grant(ctx, access.ActionRetrieve, u.ResourceID(), r.ResourceID())
			Expect(statusAPI.Set(
				AuthedCtx(ctx, u), nil,
				apistatus.SetRequest{Statuses: []status.Status[any]{{
					Owner:   new(r.ResourceID()),
					Time:    telem.Now(),
					Variant: status.VariantError,
					Details: "not an object",
				}}},
			)).Error().To(MatchError(ContainSubstring("statuses.0.details")))
			retrieved := MustSucceed(apiSvc.Retrieve(
				AuthedCtx(ctx, u),
				apirack.RetrieveRequest{
					Where:         &rack.Where{Key: &filter.Set[rack.Key]{Eq: &r.Key}},
					IncludeStatus: true,
				},
			))
			Expect(retrieved.Racks[0].Status.Message).To(Equal("Rack is running"))
			created := MustSucceed(apiSvc.Create(
				AuthedCtx(ctx, u), nil,
				apirack.CreateRequest{Racks: []rack.Rack{r}},
			))
			Expect(created.Racks[0].Status.Message).To(Equal("Rack is running"))
		},
	)
})

var _ = Describe("RetrieveRequest", func() {
	It("Should reject a request member that it does not declare", func() {
		var req apirack.RetrieveRequest
		Expect(json.Unmarshal([]byte(`{"keys":[1]}`), &req)).To(SatisfyAll(
			BeAssignableToTypeOf(&json.SemanticError{}),
			MatchError(ContainSubstring(`unknown object member name "keys"`)),
		))
	})
})

var _ = Describe("Delete", func() {
	It("Should end the sessions of the deleted rack", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Doomed"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(ctx, access.ActionDelete, u.ResourceID(), r.ResourceID())
		_, key := MustSucceed2(
			svc.APIKey.NewWriter(nil).Create(ctx, r.ResourceID(), "driver"),
		)
		_, token := MustSucceed2(svc.Auth.Login(ctx, auth.Proof{
			Variant: auth.APIKeyProof{
				APIKeyCredentials: auth.APIKeyCredentials{Key: key},
			},
		}))
		Expect(apiSvc.Delete(
			AuthedCtx(ctx, u), nil,
			apirack.DeleteRequest{Keys: []rack.Key{r.Key}},
		)).Error().ToNot(HaveOccurred())
		Expect(svc.Auth.Authenticate(ctx, token)).Error().
			To(MatchError(auth.ErrInvalidToken))
	})
})

// createDeviceGroup creates a device group with no top parent, where racks can sit.
func createDeviceGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Scope: resource.TypeDevice}
	Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			r := rack.Rack{Name: "Set group denied"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			g := createDeviceGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), nil,
				apirack.SetGroupRequest{Group: &g.Key, Keys: []rack.Key{r.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveRack(ctx, r.Key).Group).To(BeNil())
		},
	)
	It(
		"Should move the rack with update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			r := rack.Rack{Name: "Set group granted"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			g := createDeviceGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				r.ResourceID(),
				g.ResourceID(),
			)
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), nil,
				apirack.SetGroupRequest{Group: &g.Key, Keys: []rack.Key{r.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveRack(ctx, r.Key).Group).To(Equal(&g.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	rackType := resource.ID{Type: resource.TypeRack}
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createDeviceGroup(ctx, "Create denied")
			grant(ctx, access.ActionCreate, u.ResourceID(), rackType)
			Expect(apiSvc.Create(
				AuthedCtx(ctx, u), nil,
				apirack.CreateRequest{Racks: []rack.Rack{{Name: "New", Group: &g.Key}}},
			)).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the rack with update on the group", func(ctx SpecContext) {
		u := createUser(ctx)
		g := createDeviceGroup(ctx, "Create granted")
		grant(ctx, access.ActionCreate, u.ResourceID(), rackType)
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(apiSvc.Create(
			AuthedCtx(ctx, u), nil,
			apirack.CreateRequest{Racks: []rack.Rack{{Name: "New", Group: &g.Key}}},
		))
		Expect(retrieveRack(ctx, res.Racks[0].Key).Group).To(Equal(&g.Key))
	})
})
