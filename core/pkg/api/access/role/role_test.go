// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apirole "github.com/synnaxlabs/synnax/pkg/api/access/role"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var roleType = resource.ID{Type: resource.TypeRole}

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
		grantOn(ctx, u.ResourceID(), n.action, n.object)
	}
}

func roleExists(ctx SpecContext, key role.Key) bool {
	GinkgoHelper()
	return MustSucceed(svc.RBAC.Role.NewRetrieve().
		Where(role.MatchKeys(key)).
		Exists(ctx, db))
}

func newRole() role.Role {
	return role.Role{Key: uuid.New(), Name: "role-" + uuid.New().String()}
}

var _ = Describe("Create", func() {
	It("Should refuse a subject that cannot create the role", func(ctx SpecContext) {
		u := createUser(ctx)
		r := newRole()
		Expect(create(AuthedCtx(ctx, u), apirole.CreateRequest{
			Roles: []role.Role{r},
		})).Error().To(denied(u, access.ActionCreate, role.ResourceID(r.Key)))
		Expect(roleExists(ctx, r.Key)).To(BeFalse())
	})

	It("Should create a role for a subject that can create roles", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, roleType)
		r := newRole()
		MustSucceed(create(AuthedCtx(ctx, u), apirole.CreateRequest{
			Roles: []role.Role{r},
		}))
		Expect(roleExists(ctx, r.Key)).To(BeTrue())
	})

	It("Should refuse a role whose key exists", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRole(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, r.ResourceID())
		renamed := r
		renamed.Name = "renamed-" + r.Name
		Expect(create(AuthedCtx(ctx, u), apirole.CreateRequest{
			Roles: []role.Role{renamed},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("role %s already exists", r.Key)),
		))
		var stored role.Role
		Expect(svc.RBAC.Role.NewRetrieve().
			Where(role.MatchKeys(r.Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.Name).To(Equal(r.Name))
	})
})

var _ = Describe("Delete", func() {
	// createWithPolicy creates a role with one policy, and returns what deleting the
	// role needs.
	createWithPolicy := func(ctx SpecContext) (role.Role, policy.Policy, []need) {
		r := createRole(ctx)
		p := policy.Policy{
			Role:    r.Key,
			Name:    "policy-" + uuid.New().String(),
			Objects: []resource.ID{{Type: resource.TypeChannel}},
			Actions: []access.Action{access.ActionRetrieve},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.RBAC.Policy.NewWriter(tx, false).Create(ctx, &p)
		})).To(Succeed())
		return r, p, []need{
			{access.ActionDelete, r.ResourceID()},
			{access.ActionDelete, p.ResourceID()},
		}
	}

	DescribeTable("Should refuse a subject that lacks an access it needs", func(
		ctx SpecContext,
		refused int,
	) {
		r, p, needs := createWithPolicy(ctx)
		u := createUser(ctx)
		grantAll(ctx, u, slices.Delete(slices.Clone(needs), refused, refused+1)...)
		Expect(del(AuthedCtx(ctx, u), apirole.DeleteRequest{
			Keys: []role.Key{r.Key},
		})).Error().To(denied(u, needs[refused].action, needs[refused].object))
		Expect(roleExists(ctx, r.Key)).To(BeTrue())
		Expect(MustSucceed(svc.RBAC.Policy.NewRetrieve().
			Where(policy.MatchKeys(p.Key)).
			Exists(ctx, db))).To(BeTrue())
	},
		Entry("delete on the role", 0),
		Entry("delete on its policy", 1),
	)

	It("Should delete a role and its policies", func(ctx SpecContext) {
		r, p, needs := createWithPolicy(ctx)
		u := createUser(ctx)
		grantAll(ctx, u, needs...)
		MustSucceed(del(AuthedCtx(ctx, u), apirole.DeleteRequest{
			Keys: []role.Key{r.Key},
		}))
		Expect(roleExists(ctx, r.Key)).To(BeFalse())
		Expect(MustSucceed(svc.RBAC.Policy.NewRetrieve().
			Where(policy.MatchKeys(p.Key)).
			Exists(ctx, db))).To(BeFalse())
	})
})

var _ = Describe("Retrieve", func() {
	It("Should leave out the roles that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		granted := createRole(ctx)
		createRole(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionRetrieve, granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apirole.RetrieveRequest{}))
		Expect(res.Roles).To(HaveExactElements(HaveField("Key", granted.Key)))
	})

	It("Should return the roles the request selects when retrieve is granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		r := grantOn(ctx, u.ResourceID(), access.ActionRetrieve, roleType)
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apirole.RetrieveRequest{
			Where: &role.Where{Key: &filter.Set[role.Key]{Eq: &r.Key}},
		}))
		Expect(res.Roles).To(HaveExactElements(HaveField("Key", r.Key)))
	})

	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		Expect(retrieve(AuthedCtx(ctx, createUser(ctx)), apirole.RetrieveRequest{
			Where: &role.Where{Key: &filter.Set[role.Key]{Eq: &r.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Dispatch", func() {
	rename := func(r role.Role, name string) apirole.DispatchRequest {
		return apirole.DispatchRequest{
			Key: r.Key,
			Actions: []role.Action{role.NewSetAction(role.SetPayload{
				Path:  []string{"name"},
				Value: name,
			})},
		}
	}
	It("Should refuse a subject that cannot update the role", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRole(ctx)
		Expect(dispatch(AuthedCtx(ctx, u), rename(r, "renamed"))).Error().
			To(denied(u, access.ActionUpdate, r.ResourceID()))
	})
	It("Should rename a role the subject can update", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRole(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, r.ResourceID())
		MustSucceed(dispatch(AuthedCtx(ctx, u), rename(r, "renamed")))
		var stored role.Role
		Expect(svc.RBAC.Role.NewRetrieve().
			Where(role.MatchKeys(r.Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.Name).To(Equal("renamed"))
	})
})
