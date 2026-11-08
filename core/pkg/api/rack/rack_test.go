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
	"context"
	"encoding/json/v2"
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var rackType = resource.ID{Type: resource.TypeRack}

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

// need is an action on an object.
type need struct {
	action access.Action
	object resource.ID
}

// grantAll grants each need to u.
func grantAll(ctx SpecContext, u user.User, needs ...need) {
	GinkgoHelper()
	for _, n := range needs {
		grant(ctx, n.action, u.ResourceID(), n.object)
	}
}

// newRackNeeds returns what a create of a new rack needs: create on the rack and its
// status, update on the rack, which its status sits in, and retrieve on the Driver role
// that the rack uses. r is the rack, or the rack type for a key the Core assigns.
func newRackNeeds(r resource.ID) []need {
	return []need{
		{access.ActionCreate, r},
		{access.ActionCreate, resource.ID{Type: resource.TypeStatus}},
		{access.ActionUpdate, r},
		{access.ActionRetrieve, role.ResourceID(svc.RBAC.Builtin.DriverKey)},
	}
}

// statusOf returns the status that r holds.
func statusOf(ctx SpecContext, r rack.Rack) resource.ID {
	GinkgoHelper()
	var st rack.Status
	Expect(svc.Status.NewRetrieve[struct{}]().
		Where(status.MatchOwners[struct{}](r.ResourceID())).
		Entry(&st).
		Exec(ctx, nil)).To(Succeed())
	return st.ResourceID()
}

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
		grantAll(ctx, u, newRackNeeds(rackType)...)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), rackType)
		res := MustSucceed(create(
			AuthedCtx(ctx, u),
			apirack.CreateRequest{Racks: []rack.Rack{{Name: "New"}}},
		))
		Expect(res.Racks).To(HaveLen(1))
		r := res.Racks[0]
		Expect(r.Status).ToNot(BeNil())
		Expect(r.Status.Owner).To(HaveValue(Equal(r.ResourceID())))
		Expect(r.Status.Variant).To(Equal(status.VariantWarning))
	})
	It(
		"Should leave out the status of a new rack the caller cannot retrieve",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grantAll(ctx, u, newRackNeeds(rackType)...)
			res := MustSucceed(create(
				AuthedCtx(ctx, u),
				apirack.CreateRequest{Racks: []rack.Rack{{Name: "New"}}},
			))
			Expect(res.Racks).To(HaveLen(1))
			Expect(res.Racks[0].Key.IsZero()).To(BeFalse())
			Expect(res.Racks[0].Status).To(BeNil())
		},
	)
	DescribeTable(
		"Should refuse a new rack and store nothing",
		func(ctx SpecContext, refused int, key rack.Key) {
			u := createUser(ctx)
			needs := newRackNeeds(key.ResourceID())
			grantAll(ctx, u, slices.Delete(slices.Clone(needs), refused, refused+1)...)
			Expect(create(
				AuthedCtx(ctx, u),
				apirack.CreateRequest{Racks: []rack.Rack{{Key: key, Name: "New"}}},
			)).Error().To(denied(u, needs[refused].action, needs[refused].object))
			Expect(svc.Rack.NewRetrieve().
				Where(rack.MatchKeys(key)).
				Exists(ctx, nil)).To(BeFalse())
		},
		Entry("without create on the rack", 0, rack.NewKey(1, 60100)),
		Entry(
			"without update on the rack, which its status sits in",
			2,
			rack.NewKey(1, 60101),
		),
		Entry("without retrieve on the Driver role", 3, rack.NewKey(1, 60102)),
	)
	It(
		"Should leave out the status of a rack the caller can only update",
		func(ctx SpecContext) {
			u := createUser(ctx)
			hidden := createRackWithStatus(ctx, "Hidden")
			grant(ctx, access.ActionUpdate, u.ResourceID(), hidden.ResourceID())
			res := MustSucceed(create(
				AuthedCtx(ctx, u),
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
		res := MustSucceed(create(
			AuthedCtx(ctx, u),
			apirack.CreateRequest{Racks: []rack.Rack{readable, hidden}},
		))
		Expect(res.Racks).To(HaveLen(2))
		Expect(res.Racks[0].Status.Message).To(Equal("Rack is running"))
		Expect(res.Racks[1].Status).To(BeNil())
	})
	It("Should reply with the live status of an existing rack", func(ctx SpecContext) {
		r := createRackWithStatus(ctx, "Existing")
		res := MustSucceed(create(
			SubjectCtx(ctx, r.ResourceID()),
			apirack.CreateRequest{Racks: []rack.Rack{r}},
		))
		Expect(res.Racks[0].Status.Message).To(Equal("Rack is running"))
	})
	It("Should reply with the new name of a renamed rack", func(ctx SpecContext) {
		r := createRackWithStatus(ctx, "Old")
		r.Name = "New"
		res := MustSucceed(create(
			SubjectCtx(ctx, r.ResourceID()),
			apirack.CreateRequest{Racks: []rack.Rack{r}},
		))
		Expect(res.Racks[0].Status.Name).To(Equal("New"))
		Expect(res.Racks[0].Status.Message).To(Equal("Rack is running"))
	})
	It("Should let a rack with the Driver role save itself", func(ctx SpecContext) {
		r := rack.Rack{Name: "Self"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		r.Integrations = []string{"ni"}
		MustSucceed(create(
			SubjectCtx(ctx, r.ResourceID()),
			apirack.CreateRequest{Racks: []rack.Rack{r}},
		))
		Expect(retrieveRack(ctx, r.Key).Integrations).To(Equal([]string{"ni"}))
	})
	It(
		"Should let a rack with the Driver role save itself twice",
		func(ctx SpecContext) {
			r := rack.Rack{Name: "Self"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			MustSucceed(create(
				SubjectCtx(ctx, r.ResourceID()),
				apirack.CreateRequest{Racks: []rack.Rack{r, r}},
			))
		},
	)
	It(
		"Should not let a rack with the Driver role create a rack",
		func(ctx SpecContext) {
			r := rack.Rack{Name: "Self"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			Expect(create(
				SubjectCtx(ctx, r.ResourceID()),
				apirack.CreateRequest{Racks: []rack.Rack{{Name: "Other"}}},
			)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("%s cannot create rack:", r.ResourceID())),
			))
		},
	)
	It("Should require update to save an existing rack", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionCreate, u.ResourceID(), rackType)
		r := rack.Rack{Name: "Existing"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		Expect(create(
			AuthedCtx(ctx, u),
			apirack.CreateRequest{Racks: []rack.Rack{r}},
		)).Error().To(denied(u, access.ActionUpdate, r.ResourceID()))
	})
})

var _ = Describe("Rename", func() {
	It("Should rename the rack and its status with update on both", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Before", Integrations: []string{"ni"}}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			r.ResourceID(),
			statusOf(ctx, r),
		)
		MustSucceed(rename(
			AuthedCtx(ctx, u),
			apirack.RenameRequest{Key: r.Key, Name: "After"},
		))
		renamed := retrieveRack(ctx, r.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Integrations).To(Equal([]string{"ni"}))
	})
	It("Should keep the embedded flag of the embedded rack", func(ctx SpecContext) {
		u := createUser(ctx)
		embedded := retrieveRack(ctx, svc.Rack.EmbeddedKey)
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			embedded.ResourceID(),
			statusOf(ctx, embedded),
		)
		MustSucceed(rename(
			AuthedCtx(ctx, u),
			apirack.RenameRequest{Key: embedded.Key, Name: "Renamed Embedded"},
		))
		renamed := retrieveRack(ctx, embedded.Key)
		Expect(renamed.Name).To(Equal("Renamed Embedded"))
		Expect(renamed.Embedded).To(BeTrue())
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Before"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			r.ResourceID(),
			statusOf(ctx, r),
		)
		Expect(rename(
			AuthedCtx(ctx, u),
			apirack.RenameRequest{Key: r.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveRack(ctx, r.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing rack", func(ctx SpecContext) {
		u := createUser(ctx)
		key := rack.NewKey(1, 60000)
		grant(ctx, access.ActionUpdate, u.ResourceID(), key.ResourceID())
		Expect(rename(
			AuthedCtx(ctx, u),
			apirack.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	rackID := func(_ SpecContext, r rack.Rack) resource.ID { return r.ResourceID() }
	DescribeTable(
		"Should refuse the rename and keep the name",
		func(
			ctx SpecContext,
			granted func(SpecContext, rack.Rack) resource.ID,
			refused func(SpecContext, rack.Rack) resource.ID,
		) {
			u := createUser(ctx)
			r := rack.Rack{Name: "Before"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			grant(ctx, access.ActionUpdate, u.ResourceID(), granted(ctx, r))
			Expect(rename(
				AuthedCtx(ctx, u),
				apirack.RenameRequest{Key: r.Key, Name: "After"},
			)).Error().To(denied(u, access.ActionUpdate, refused(ctx, r)))
			Expect(retrieveRack(ctx, r.Key).Name).To(Equal("Before"))
		},
		Entry("without update on the rack", statusOf, rackID),
		Entry("without update on its status", rackID, statusOf),
	)
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
			created := MustSucceed(create(
				AuthedCtx(ctx, u),
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
	// deleteNeeds returns what a delete of r needs: delete on r and its status, and
	// update on r, which its status, API keys, sessions, and tasks leave.
	deleteNeeds := func(ctx SpecContext, r rack.Rack) []need {
		GinkgoHelper()
		return []need{
			{access.ActionDelete, r.ResourceID()},
			{access.ActionDelete, statusOf(ctx, r)},
			{access.ActionUpdate, r.ResourceID()},
		}
	}
	// deleterOf returns a context for a new user allowed to delete r.
	deleterOf := func(ctx SpecContext, r rack.Rack, extra ...need) context.Context {
		GinkgoHelper()
		u := createUser(ctx)
		grantAll(ctx, u, append(deleteNeeds(ctx, r), extra...)...)
		return AuthedCtx(ctx, u)
	}
	// createLoggedIn creates a rack with an API key and returns the rack and the token
	// of a session that the key opened.
	createLoggedIn := func(ctx SpecContext) (rack.Rack, string) {
		GinkgoHelper()
		r := rack.Rack{Name: "Doomed"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		_, key := MustSucceed2(
			svc.APIKey.NewWriter(nil).Create(ctx, r.ResourceID(), "driver"),
		)
		_, token := MustSucceed2(svc.Auth.Login(ctx, auth.Proof{
			Variant: auth.APIKeyProof{
				APIKeyCredentials: auth.APIKeyCredentials{Key: key},
			},
		}))
		return r, token
	}
	It("Should end the sessions of the deleted rack", func(ctx SpecContext) {
		r, token := createLoggedIn(ctx)
		Expect(del(
			deleterOf(ctx, r),
			apirack.DeleteRequest{Keys: []rack.Key{r.Key}},
		)).Error().ToNot(HaveOccurred())
		Expect(svc.Auth.Authenticate(ctx, token)).Error().
			To(MatchError(auth.ErrInvalidToken))
	})
	DescribeTable(
		"Should refuse the delete and keep the rack and its sessions",
		func(ctx SpecContext, refused int) {
			u := createUser(ctx)
			r, token := createLoggedIn(ctx)
			needs := deleteNeeds(ctx, r)
			grantAll(ctx, u, slices.Delete(slices.Clone(needs), refused, refused+1)...)
			Expect(del(
				AuthedCtx(ctx, u),
				apirack.DeleteRequest{Keys: []rack.Key{r.Key}},
			)).Error().To(denied(u, needs[refused].action, needs[refused].object))
			MustSucceed(svc.Auth.Authenticate(ctx, token))
		},
		Entry("without delete on the rack", 0),
		Entry("without delete on its status", 1),
		Entry("without update on the rack, which its sessions leave", 2),
	)
	exists := func(ctx SpecContext, key rack.Key) bool {
		GinkgoHelper()
		return MustSucceed(svc.Rack.NewRetrieve().
			Where(rack.MatchKeys(key)).
			Exists(ctx, nil))
	}

	It("Should refuse to delete a rack that holds a device", func(ctx SpecContext) {
		r := rack.Rack{Name: "With Device"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		d := device.Device{
			Key:      "api-rack-device",
			Rack:     r.Key,
			Location: "loc",
			Name:     "Device",
			Make:     "make",
			Model:    "model",
		}
		Expect(svc.Device.NewWriter(nil).Create(ctx, &d)).To(Succeed())
		Expect(del(
			deleterOf(ctx, r),
			apirack.DeleteRequest{Keys: []rack.Key{r.Key}},
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(fmt.Sprintf(
				"cannot delete rack %s because device api-rack-device references "+
					"it through rack",
				r.Key,
			))),
		))
		Expect(exists(ctx, r.Key)).To(BeTrue())
	})

	It(
		"Should refuse to delete a rack that holds a task that is not internal",
		func(ctx SpecContext) {
			r := rack.Rack{Name: "With Task"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			t := task.Task{
				Rack: r.Key,
				Name: "User Task",
				Type: pagerduty.AlertTaskType,
			}
			Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			Expect(del(
				deleterOf(ctx, r),
				apirack.DeleteRequest{Keys: []rack.Key{r.Key}},
			)).Error().To(MatchError(ContainSubstring(
				"cannot delete rack With Task because it still holds tasks",
			)))
			Expect(exists(ctx, r.Key)).To(BeTrue())
		},
	)

	It("Should delete the internal tasks of a deleted rack", func(ctx SpecContext) {
		r := rack.Rack{Name: "With Internal Task"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		t := task.Task{
			Rack:     r.Key,
			Name:     "Scanner",
			Type:     pagerduty.AlertTaskType,
			Internal: true,
		}
		Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
		var st status.Status[struct{}]
		Expect(svc.Status.NewRetrieve[struct{}]().
			Where(status.MatchOwners[struct{}](t.ResourceID())).
			Entry(&st).
			Exec(ctx, nil)).To(Succeed())
		MustSucceed(del(
			deleterOf(
				ctx,
				r,
				need{access.ActionDelete, t.ResourceID()},
				need{access.ActionDelete, st.ResourceID()},
				need{access.ActionUpdate, t.ResourceID()},
			),
			apirack.DeleteRequest{Keys: []rack.Key{r.Key}},
		))
		Expect(exists(ctx, r.Key)).To(BeFalse())
		Expect(svc.Task.NewRetrieve().
			Where(task.MatchKeys(t.Key)).
			Exists(ctx, nil)).To(BeFalse())
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
	// denied matches an error that denies the subject the action on object.
	denied := func(
		subject resource.ID,
		action access.Action,
		object resource.ID,
	) types.GomegaMatcher {
		return SatisfyAll(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring("%s cannot %s %s", subject, action, object)),
		)
	}
	move := func(
		ctx SpecContext,
		u user.User,
		g group.Group,
		r rack.Rack,
	) (struct{}, error) {
		return setGroup(
			AuthedCtx(ctx, u),
			apirack.SetGroupRequest{Group: &g.Key, Keys: []rack.Key{r.Key}},
		)
	}
	createRack := func(ctx SpecContext, name string) rack.Rack {
		GinkgoHelper()
		r := rack.Rack{Name: name}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		return r
	}
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			r := createRack(ctx, "Set group denied")
			g := createDeviceGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
			Expect(move(ctx, u, g, r)).
				Error().To(denied(u.ResourceID(), access.ActionUpdate, g.ResourceID()))
			Expect(retrieveRack(ctx, r.Key).Group).To(BeNil())
		},
	)
	It(
		"Should refuse the move without update on the source group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			r := createRack(ctx, "Set group source denied")
			from := createDeviceGroup(ctx, "Set group source denied")
			Expect(svc.Rack.NewWriter(nil).SetGroup(ctx, &from.Key, r.Key)).
				To(Succeed())
			to := createDeviceGroup(ctx, "Set group destination granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				r.ResourceID(),
				to.ResourceID(),
			)
			Expect(move(ctx, u, to, r)).
				Error().
				To(denied(u.ResourceID(), access.ActionUpdate, from.ResourceID()))
			Expect(retrieveRack(ctx, r.Key).Group).To(Equal(&from.Key))
		},
	)
	It("Should refuse the move without update on the rack", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRack(ctx, "Set group rack denied")
		g := createDeviceGroup(ctx, "Set group rack denied")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		Expect(move(ctx, u, g, r)).
			Error().To(denied(u.ResourceID(), access.ActionUpdate, r.ResourceID()))
		Expect(retrieveRack(ctx, r.Key).Group).To(BeNil())
	})
	It(
		"Should move the rack with update on the rack and both groups",
		func(ctx SpecContext) {
			u := createUser(ctx)
			r := createRack(ctx, "Set group granted")
			from := createDeviceGroup(ctx, "Set group source granted")
			Expect(svc.Rack.NewWriter(nil).SetGroup(ctx, &from.Key, r.Key)).
				To(Succeed())
			to := createDeviceGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				r.ResourceID(),
				from.ResourceID(),
				to.ResourceID(),
			)
			MustSucceed(move(ctx, u, to, r))
			Expect(retrieveRack(ctx, r.Key).Group).To(Equal(&to.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createDeviceGroup(ctx, "Create denied")
			grantAll(ctx, u, newRackNeeds(rackType)...)
			Expect(create(
				AuthedCtx(ctx, u),
				apirack.CreateRequest{Racks: []rack.Rack{{Name: "New", Group: &g.Key}}},
			)).Error().To(denied(u, access.ActionUpdate, g.ResourceID()))
		},
	)
	It("Should create the rack with update on the group", func(ctx SpecContext) {
		u := createUser(ctx)
		g := createDeviceGroup(ctx, "Create granted")
		grantAll(ctx, u, newRackNeeds(rackType)...)
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(create(
			AuthedCtx(ctx, u),
			apirack.CreateRequest{Racks: []rack.Rack{{Name: "New", Group: &g.Key}}},
		))
		Expect(retrieveRack(ctx, res.Racks[0].Key).Group).To(Equal(&g.Key))
	})
})
