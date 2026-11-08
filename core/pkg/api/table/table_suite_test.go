// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apitable "github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPITable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Table Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db     *gorp.DB
	svc    *service.Layer
	apiSvc *apitable.Service
	proj   project.Project
	// author has no role assignments.
	author user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apitable.NewService(apitable.ServiceConfig{
		RBAC:  svc.RBAC,
		Table: svc.Table,
	}))
	proj = project.Project{Name: "test-project"}
	Expect(svc.Project.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
	author = MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "author",
	}))
})

// newUser creates a user with no policies.
func newUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// newCreator creates a user that may create tables in the suite project.
func newCreator(ctx SpecContext) user.User {
	GinkgoHelper()
	u := newUser(ctx)
	grant(
		ctx,
		access.ActionCreate,
		u.ResourceID(),
		resource.ID{Type: resource.TypeTable},
	)
	grant(ctx, access.ActionUpdate, u.ResourceID(), project.ResourceID(proj.Key))
	return u
}

// grant grants the action on the objects to the subject through a new role.
func grant(
	ctx SpecContext,
	action access.Action,
	subject resource.ID,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	MustSucceed(rbactestutil.Grant(ctx, svc.RBAC, svc.User, key, actions, objects...))
}
