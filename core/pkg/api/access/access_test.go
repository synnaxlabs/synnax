// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiaccess "github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// createRole creates a role that is not built in.
func createRole(ctx SpecContext) role.Role {
	GinkgoHelper()
	r := role.Role{Name: "role-" + uuid.New().String()}
	Expect(rbacSvc.Role.NewWriter(nil, false).Create(ctx, &r)).To(Succeed())
	return r
}

// createPolicy creates a policy of the role r that reads channels.
func createPolicy(ctx SpecContext, r role.Key) policy.Policy {
	GinkgoHelper()
	p := newPolicy(r)
	Expect(rbacSvc.Policy.NewWriter(nil, false).Create(ctx, &p)).To(Succeed())
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

// createUser creates a user that may take the action on policies, and update the roles
// with the given keys.
func createUser(ctx SpecContext, action access.Action, roles ...role.Key) user.User {
	GinkgoHelper()
	u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Name: "u"}))
	MustSucceed(rbactestutil.Grant(
		ctx,
		rbacSvc,
		userSvc,
		u.Key,
		[]access.Action{action},
		resource.ID{Type: resource.TypePolicy},
	))
	if len(roles) > 0 {
		MustSucceed(rbactestutil.Grant(
			ctx,
			rbacSvc,
			userSvc,
			u.Key,
			[]access.Action{access.ActionUpdate},
			role.ResourceIDs(roles)...,
		))
	}
	return u
}

func policyExists(ctx SpecContext, key policy.Key) bool {
	GinkgoHelper()
	return MustSucceed(rbacSvc.Policy.NewRetrieve().
		Where(policy.MatchKeys(key)).
		Exists(ctx, nil))
}

var _ = Describe("CreatePolicy", func() {
	It("Should refuse a subject that cannot update the role of the policy", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		u := createUser(ctx, access.ActionCreate)
		p := newPolicy(r.Key)
		Expect(apiSvc.CreatePolicy(userCtx(ctx, u), nil, apiaccess.CreatePolicyRequest{
			Policies: []policy.Policy{p},
		})).Error().To(MatchError(access.ErrDenied))
		var held []policy.Policy
		Expect(rbacSvc.Policy.NewRetrieve().
			Where(policy.MatchRoles(r.Key)).
			Entries(&held).
			Exec(ctx, nil)).To(Succeed())
		Expect(held).To(BeEmpty())
	})

	It("Should create a policy for a subject that can update its role", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		u := createUser(ctx, access.ActionCreate, r.Key)
		res := MustSucceed(apiSvc.CreatePolicy(
			userCtx(ctx, u),
			nil,
			apiaccess.CreatePolicyRequest{Policies: []policy.Policy{newPolicy(r.Key)}},
		))
		Expect(res.Policies).To(HaveLen(1))
		Expect(policyExists(ctx, res.Policies[0].Key)).To(BeTrue())
	})

	It("Should refuse to move a policy off a role the subject cannot update", func(
		ctx SpecContext,
	) {
		from, to := createRole(ctx), createRole(ctx)
		p := createPolicy(ctx, from.Key)
		u := createUser(ctx, access.ActionCreate, to.Key)
		p.Role = to.Key
		Expect(apiSvc.CreatePolicy(userCtx(ctx, u), nil, apiaccess.CreatePolicyRequest{
			Policies: []policy.Policy{p},
		})).Error().To(MatchError(access.ErrDenied))
		var stored policy.Policy
		Expect(rbacSvc.Policy.NewRetrieve().
			Where(policy.MatchKeys(p.Key)).
			Entry(&stored).
			Exec(ctx, nil)).To(Succeed())
		Expect(stored.Role).To(Equal(from.Key))
	})

	It("Should refuse a policy of a built-in role, even for the root user", func(
		ctx SpecContext,
	) {
		Expect(apiSvc.CreatePolicy(rootCtx(ctx), nil, apiaccess.CreatePolicyRequest{
			Policies: []policy.Policy{newPolicy(rbacSvc.Builtin.ViewerKey)},
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
		var builtin policy.Policy
		Expect(rbacSvc.Policy.NewRetrieve().
			Where(policy.MatchRoles(rbacSvc.Builtin.ViewerKey)).
			Limit(1).
			Entry(&builtin).
			Exec(ctx, nil)).To(Succeed())
		overwrite := newPolicy(createRole(ctx).Key)
		overwrite.Key = builtin.Key
		Expect(apiSvc.CreatePolicy(rootCtx(ctx), nil, apiaccess.CreatePolicyRequest{
			Policies: []policy.Policy{overwrite},
		})).Error().To(MatchError(ContainSubstring(
			"cannot change the policies of built-in role Viewer",
		)))
	})
})

var _ = Describe("DeletePolicy", func() {
	It("Should refuse a subject that cannot update the role of the policy", func(
		ctx SpecContext,
	) {
		p := createPolicy(ctx, createRole(ctx).Key)
		u := createUser(ctx, access.ActionDelete)
		Expect(apiSvc.DeletePolicy(userCtx(ctx, u), nil, apiaccess.DeletePolicyRequest{
			Keys: []policy.Key{p.Key},
		})).Error().To(MatchError(access.ErrDenied))
		Expect(policyExists(ctx, p.Key)).To(BeTrue())
	})

	It("Should delete a policy for a subject that can update its role", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		p := createPolicy(ctx, r.Key)
		u := createUser(ctx, access.ActionDelete, r.Key)
		MustSucceed(
			apiSvc.DeletePolicy(userCtx(ctx, u), nil, apiaccess.DeletePolicyRequest{
				Keys: []policy.Key{p.Key},
			}),
		)
		Expect(policyExists(ctx, p.Key)).To(BeFalse())
	})

	It("Should refuse to delete a policy of a built-in role", func(ctx SpecContext) {
		var builtin policy.Policy
		Expect(rbacSvc.Policy.NewRetrieve().
			Where(policy.MatchRoles(rbacSvc.Builtin.ViewerKey)).
			Limit(1).
			Entry(&builtin).
			Exec(ctx, nil)).To(Succeed())
		Expect(apiSvc.DeletePolicy(rootCtx(ctx), nil, apiaccess.DeletePolicyRequest{
			Keys: []policy.Key{builtin.Key},
		})).Error().To(MatchError(ContainSubstring(
			"cannot change the policies of built-in role Viewer",
		)))
		Expect(policyExists(ctx, builtin.Key)).To(BeTrue())
	})
})

var _ = Describe("RetrievePolicy", func() {
	It("Should retrieve the policies of the roles a subject holds", func(
		ctx SpecContext,
	) {
		res := MustSucceed(apiSvc.RetrievePolicy(
			rootCtx(ctx),
			apiaccess.RetrievePolicyRequest{Subjects: []resource.ID{root.ResourceID()}},
		))
		Expect(res.Policies).ToNot(BeEmpty())
		Expect(res.Policies).To(HaveEach(HaveField("Role", rbacSvc.Builtin.OwnerKey)))
	})

	It("Should retrieve the policies of a role", func(ctx SpecContext) {
		res := MustSucceed(apiSvc.RetrievePolicy(
			rootCtx(ctx),
			apiaccess.RetrievePolicyRequest{
				Roles: []role.Key{rbacSvc.Builtin.ViewerKey},
			},
		))
		Expect(res.Policies).ToNot(BeEmpty())
		Expect(res.Policies).To(HaveEach(HaveField("Role", rbacSvc.Builtin.ViewerKey)))
	})

	It("Should retrieve no policies for a subject with no roles", func(
		ctx SpecContext,
	) {
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Name: "none"}))
		res := MustSucceed(apiSvc.RetrievePolicy(
			rootCtx(ctx),
			apiaccess.RetrievePolicyRequest{Subjects: []resource.ID{u.ResourceID()}},
		))
		Expect(res.Policies).To(BeEmpty())
	})
})
