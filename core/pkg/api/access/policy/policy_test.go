// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apipolicy "github.com/synnaxlabs/synnax/pkg/api/access/policy"
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

var policyType = resource.ID{Type: resource.TypePolicy}

// createPolicy creates a policy of the role r that reads channels.
func createPolicy(ctx SpecContext, r role.Key) policy.Policy {
	GinkgoHelper()
	p := newPolicy(r)
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.RBAC.Policy.NewWriter(tx, false).Create(ctx, &p)
	})).To(Succeed())
	return p
}

func newPolicy(r role.Key) policy.Policy {
	return policy.Policy{
		Role:    r,
		Name:    "policy-" + uuid.New().String(),
		Objects: []resource.ID{{Type: resource.TypeChannel}},
		Actions: []access.Action{access.ActionRetrieve},
	}
}

// createEditor creates a user that may take the action on policies, and update the
// roles with the given keys.
func createEditor(ctx SpecContext, action access.Action, roles ...role.Key) user.User {
	GinkgoHelper()
	u := createUser(ctx)
	grantOn(ctx, u.ResourceID(), action, policyType)
	if len(roles) > 0 {
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, role.ResourceIDs(roles)...)
	}
	return u
}

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

func policyExists(ctx SpecContext, key policy.Key) bool {
	GinkgoHelper()
	return MustSucceed(svc.RBAC.Policy.NewRetrieve().
		Where(policy.MatchKeys(key)).
		Exists(ctx, nil))
}

// builtinPolicy returns a policy of the built-in Viewer role.
func builtinPolicy(ctx SpecContext) policy.Policy {
	GinkgoHelper()
	var p policy.Policy
	Expect(svc.RBAC.Policy.NewRetrieve().
		Where(policy.MatchRoles(svc.RBAC.Builtin.ViewerKey)).
		Limit(1).
		Entry(&p).
		Exec(ctx, nil)).To(Succeed())
	return p
}

var _ = Describe("Create", func() {
	It("Should refuse a subject that cannot update the role of the policy", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		u := createEditor(ctx, access.ActionCreate)
		Expect(create(AuthedCtx(ctx, u), apipolicy.CreateRequest{
			Policies: []policy.Policy{newPolicy(r.Key)},
		})).Error().To(denied(u, access.ActionUpdate, role.ResourceID(r.Key)))
		Expect(svc.RBAC.Policy.NewRetrieve().
			Where(policy.MatchRoles(r.Key)).
			Exists(ctx, nil)).To(BeFalse())
	})

	It("Should refuse a subject that cannot create the policy", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		u := createEditor(ctx, access.ActionUpdate, r.Key)
		p := newPolicy(r.Key)
		p.Key = uuid.New()
		Expect(create(AuthedCtx(ctx, u), apipolicy.CreateRequest{
			Policies: []policy.Policy{p},
		})).Error().To(denied(u, access.ActionCreate, policy.ResourceID(p.Key)))
		Expect(policyExists(ctx, p.Key)).To(BeFalse())
	})

	It("Should create a policy for a subject that can update its role", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		u := createEditor(ctx, access.ActionCreate, r.Key)
		res := MustSucceed(create(
			AuthedCtx(ctx, u),
			apipolicy.CreateRequest{Policies: []policy.Policy{newPolicy(r.Key)}},
		))
		Expect(res.Policies).To(HaveLen(1))
		Expect(policyExists(ctx, res.Policies[0].Key)).To(BeTrue())
	})

	It("Should refuse to move a policy off a role the subject cannot update", func(
		ctx SpecContext,
	) {
		from, to := createRole(ctx), createRole(ctx)
		p := createPolicy(ctx, from.Key)
		u := createEditor(ctx, access.ActionUpdate, to.Key)
		p.Role = to.Key
		Expect(create(AuthedCtx(ctx, u), apipolicy.CreateRequest{
			Policies: []policy.Policy{p},
		})).Error().To(denied(u, access.ActionUpdate, role.ResourceID(from.Key)))
		var stored policy.Policy
		Expect(svc.RBAC.Policy.NewRetrieve().
			Where(policy.MatchKeys(p.Key)).
			Entry(&stored).
			Exec(ctx, nil)).To(Succeed())
		Expect(stored.Role).To(Equal(from.Key))
	})

	It("Should overwrite a policy that keeps its role with update on the policy", func(
		ctx SpecContext,
	) {
		p := createPolicy(ctx, createRole(ctx).Key)
		u := createEditor(ctx, access.ActionUpdate)
		p.Name = "renamed-" + p.Name
		MustSucceed(create(AuthedCtx(ctx, u), apipolicy.CreateRequest{
			Policies: []policy.Policy{p},
		}))
		var stored policy.Policy
		Expect(svc.RBAC.Policy.NewRetrieve().
			Where(policy.MatchKeys(p.Key)).
			Entry(&stored).
			Exec(ctx, nil)).To(Succeed())
		Expect(stored.Name).To(Equal(p.Name))
	})

	It("Should refuse a policy of a built-in role, even for the root user", func(
		ctx SpecContext,
	) {
		Expect(create(AuthedCtx(ctx, root), apipolicy.CreateRequest{
			Policies: []policy.Policy{newPolicy(svc.RBAC.Builtin.ViewerKey)},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"cannot change the policies of built-in role Viewer",
			)),
		))
	})

	It("Should refuse to overwrite a policy of a built-in role", func(
		ctx SpecContext,
	) {
		overwrite := newPolicy(createRole(ctx).Key)
		overwrite.Key = builtinPolicy(ctx).Key
		Expect(create(AuthedCtx(ctx, root), apipolicy.CreateRequest{
			Policies: []policy.Policy{overwrite},
		})).Error().To(MatchError(ContainSubstring(
			"cannot change the policies of built-in role Viewer",
		)))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should return the policies of a subject when retrieve is granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := grantOn(ctx, u.ResourceID(), access.ActionRetrieve, policyType)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apipolicy.RetrieveRequest{
			Where: &policy.Where{Subjects: []resource.ID{u.ResourceID()}},
		}))
		Expect(res.Policies).To(HaveExactElements(HaveField("Key", p.Key)))
	})

	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := grantOn(ctx, u.ResourceID(), access.ActionUpdate, u.ResourceID())
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apipolicy.RetrieveRequest{
			Where: &policy.Where{Key: &filter.Set[policy.Key]{Eq: &p.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})

	It("Should retrieve the policies of the roles a subject holds", func(
		ctx SpecContext,
	) {
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, root),
			apipolicy.RetrieveRequest{
				Where: &policy.Where{Subjects: []resource.ID{root.ResourceID()}},
			},
		))
		Expect(res.Policies).ToNot(BeEmpty())
		Expect(res.Policies).
			To(HaveEach(HaveField("Role", svc.RBAC.Builtin.OwnerKey)))
	})

	It("Should retrieve the policies of a role", func(ctx SpecContext) {
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, root),
			apipolicy.RetrieveRequest{
				Where: &policy.Where{Roles: []role.Key{svc.RBAC.Builtin.ViewerKey}},
			},
		))
		Expect(res.Policies).ToNot(BeEmpty())
		Expect(res.Policies).
			To(HaveEach(HaveField("Role", svc.RBAC.Builtin.ViewerKey)))
	})

	It("Should retrieve no policies for a subject whose role has none", func(
		ctx SpecContext,
	) {
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, root),
			apipolicy.RetrieveRequest{
				Where: &policy.Where{
					Subjects: []resource.ID{createUser(ctx).ResourceID()},
				},
			},
		))
		Expect(res.Policies).To(BeEmpty())
	})
})

var _ = Describe("Delete", func() {
	It("Should refuse a subject that cannot update the role of the policy", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		p := createPolicy(ctx, r.Key)
		u := createEditor(ctx, access.ActionDelete)
		Expect(del(AuthedCtx(ctx, u), apipolicy.DeleteRequest{
			Keys: []policy.Key{p.Key},
		})).Error().To(denied(u, access.ActionUpdate, role.ResourceID(r.Key)))
		Expect(policyExists(ctx, p.Key)).To(BeTrue())
	})

	It("Should refuse a subject that cannot delete the policy", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		p := createPolicy(ctx, r.Key)
		u := createEditor(ctx, access.ActionUpdate, r.Key)
		Expect(del(AuthedCtx(ctx, u), apipolicy.DeleteRequest{
			Keys: []policy.Key{p.Key},
		})).Error().To(denied(u, access.ActionDelete, p.ResourceID()))
		Expect(policyExists(ctx, p.Key)).To(BeTrue())
	})

	It("Should delete a policy for a subject that can update its role", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		p := createPolicy(ctx, r.Key)
		u := createEditor(ctx, access.ActionDelete, r.Key)
		MustSucceed(del(AuthedCtx(ctx, u), apipolicy.DeleteRequest{
			Keys: []policy.Key{p.Key},
		}))
		Expect(policyExists(ctx, p.Key)).To(BeFalse())
	})

	It("Should refuse to delete a policy of a built-in role", func(ctx SpecContext) {
		builtin := builtinPolicy(ctx)
		Expect(del(AuthedCtx(ctx, root), apipolicy.DeleteRequest{
			Keys: []policy.Key{builtin.Key},
		})).Error().To(MatchError(ContainSubstring(
			"cannot change the policies of built-in role Viewer",
		)))
		Expect(policyExists(ctx, builtin.Key)).To(BeTrue())
	})
})
