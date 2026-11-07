// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apigroup "github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Group Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apigroup.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	svc = MustOpen(svctestutil.OpenLayer(ctx, testutil.NewNode(ctx)))
	apiSvc = MustSucceed(apigroup.NewService(apigroup.ServiceConfig{
		RBAC:  svc.RBAC,
		Group: svc.Group,
	}))
})

// grant returns a policy that allows actions on objects.
func grant(actions []access.Action, objects ...resource.ID) policy.Policy {
	return policy.Policy{
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: actions,
	}
}

// newUser creates a user that holds policies.
func newUser(ctx SpecContext, policies ...policy.Policy) user.User {
	GinkgoHelper()
	u := MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
	roleWriter := svc.RBAC.Role.NewWriter(nil, true)
	policyWriter := svc.RBAC.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	for _, p := range policies {
		Expect(policyWriter.Create(ctx, &p)).To(Succeed())
		Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	}
	Expect(roleWriter.AssignRole(ctx, u.ResourceID(), r.Key)).To(Succeed())
	return u
}
