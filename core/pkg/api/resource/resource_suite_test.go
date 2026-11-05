// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiresource "github.com/synnaxlabs/synnax/pkg/api/resource"
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

func TestAPIResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Resource Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apiresource.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apiresource.NewService(apiresource.ServiceConfig{
		Resource: svc.Resource,
		RBAC:     svc.RBAC,
	}))
})

func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

func grant(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	roleWriter := svc.RBAC.Role.NewWriter(nil, true)
	policyWriter := svc.RBAC.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Role:    r.Key,
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: []access.Action{access.ActionRetrieve},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	key := MustSucceed(user.KeyFromResourceID(subject))
	Expect(svc.User.NewWriter(nil).AssignRole(ctx, r.Key, key)).To(Succeed())
}
