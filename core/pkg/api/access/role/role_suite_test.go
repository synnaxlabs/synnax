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
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apirole "github.com/synnaxlabs/synnax/pkg/api/access/role"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Access Role Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apirole.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apirole.NewService(apirole.ServiceConfig{RBAC: svc.RBAC}))
})

// createUser creates a user with no role assignments.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// createRole creates a role with no policies.
func createRole(ctx SpecContext) role.Role {
	GinkgoHelper()
	r := role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(svc.RBAC.Role.NewWriter(nil, true).Create(ctx, &r)).To(Succeed())
	return r
}

// grantOn grants the action on the objects to the subject through a fresh role, and
// returns the role.
func grantOn(
	ctx SpecContext,
	subject ontology.ID,
	action access.Action,
	objects ...ontology.ID,
) role.Role {
	GinkgoHelper()
	r := createRole(ctx)
	policyWriter := svc.RBAC.Policy.NewWriter(nil, true)
	p := &policy.Policy{
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: []access.Action{action},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(svc.RBAC.Role.NewWriter(nil, true).AssignRole(ctx, subject, r.Key)).
		To(Succeed())
	return r
}
