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
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apirole "github.com/synnaxlabs/synnax/pkg/api/access/role"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
	db     *gorp.DB
	svc    *service.Layer
	apiSvc *apirole.Service
	// create, del, and retrieve run their routes as the API layer binds them.
	create   freighter.UnaryHandler[apirole.CreateRequest, apirole.CreateResponse]
	del      freighter.UnaryHandler[apirole.DeleteRequest, struct{}]
	retrieve freighter.UnaryHandler[
		apirole.RetrieveRequest,
		apirole.RetrieveResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apirole.NewService(apirole.ServiceConfig{RBAC: svc.RBAC}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: svc.RBAC,
	}))
	create = bind.Write(binder, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
})

// createUser creates a user that can do nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// createRole creates a role with no policies.
func createRole(ctx SpecContext) role.Role {
	GinkgoHelper()
	r := role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.RBAC.Role.NewWriter(tx, true).Create(ctx, &r)
	})).To(Succeed())
	return r
}

// grantOn grants the action on the objects to the subject through a policy on its own
// role, and returns that role.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	action access.Action,
	objects ...resource.ID,
) role.Role {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	p := MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, key, actions, objects...),
	)
	var r role.Role
	Expect(svc.RBAC.Role.NewRetrieve().
		Where(role.MatchKeys(p.Role)).
		Entry(&r).
		Exec(ctx, nil)).To(Succeed())
	return r
}
