// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIArc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Arc Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc      *service.Layer
	db       *gorp.DB
	otg      *ontology.Ontology
	rbacSvc  *rbac.Service
	arcSvc   *arc.Service
	apiSvc   *apiarc.Service
	author   user.User
	rackSvc  *rack.Service
	testRack *rack.Rack
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	otg = svc.Ontology
	rbacSvc = svc.RBAC
	arcSvc = svc.Arc
	rackSvc = svc.Rack
	apiSvc = MustSucceed(apiarc.NewService(apiarc.ServiceConfig{
		RBAC:    rbacSvc,
		Arc:     arcSvc,
		Status:  svc.Status,
		Stopper: svc.TaskStopper,
	}))
	author = MustSucceed(
		svc.User.NewWriter(nil).Create(ctx, user.User{Name: "test"}),
	)
	testRack = &rack.Rack{Name: "API Test Rack"}
	Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())
})

// grantOn creates a policy granting the given action on the given objects to a
// fresh role and assigns the role to the given subject. Writes commit directly
// to the database so the api enforcers (which read committed state with no
// transaction) can observe them. The assignment is unassigned on spec cleanup so
// a grant cannot leak into later randomized specs.
func grantOn(
	ctx SpecContext,
	subject ontology.ID,
	action access.Action,
	objects ...ontology.ID,
) {
	GinkgoHelper()
	roleWriter := rbacSvc.Role.NewWriter(nil, true)
	policyWriter := rbacSvc.Policy.NewWriter(nil, true)
	r := &role.Role{
		Name:        string(action) + "-" + uuid.New().String(),
		Description: "test",
	}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    string(action) + "-policy-" + uuid.New().String(),
		Objects: objects,
		Actions: []access.Action{action},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(roleWriter.UnassignRole(ctx, subject, r.Key)).To(Succeed())
	})
}

func grantUpdateOn(ctx SpecContext, subject ontology.ID, objects ...ontology.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionUpdate, objects...)
}
