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
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apipolicy "github.com/synnaxlabs/synnax/pkg/api/access/policy"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Access Policy Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apipolicy.Service
	root   user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	creds := auth.Credentials{Username: "root", Password: "p"}
	svc = MustOpen(svctestutil.OpenLayer(
		ctx,
		node,
		service.LayerConfig{RootCredentials: creds},
	))
	root, _ = MustSucceed2(user.ProvisionRoot(ctx, svc.User, svc.Password, creds))
	apiSvc = MustSucceed(apipolicy.NewService(apipolicy.ServiceConfig{RBAC: svc.RBAC}))
})

// createUser creates a user with no role assignments.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// createRole creates a role that is not built in.
func createRole(ctx SpecContext) role.Role {
	GinkgoHelper()
	r := role.Role{Name: "role-" + uuid.New().String()}
	Expect(svc.RBAC.Role.NewWriter(nil, false).Create(ctx, &r)).To(Succeed())
	return r
}

// grantOn grants the action on the objects to the subject through a new role, and
// returns the policy that holds the grant.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	action access.Action,
	objects ...resource.ID,
) policy.Policy {
	GinkgoHelper()
	r := createRole(ctx)
	p := policy.Policy{
		Role:    r.Key,
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: []access.Action{action},
	}
	Expect(svc.RBAC.Policy.NewWriter(nil, true).Create(ctx, &p)).To(Succeed())
	key := MustSucceed(user.KeyFromResourceID(subject))
	Expect(svc.User.NewWriter(nil).AssignRole(ctx, r.Key, key)).To(Succeed())
	return p
}
