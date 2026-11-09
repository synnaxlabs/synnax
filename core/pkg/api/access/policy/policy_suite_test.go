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
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apipolicy "github.com/synnaxlabs/synnax/pkg/api/access/policy"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
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
	db     *gorp.DB
	svc    *service.Layer
	apiSvc *apipolicy.Service
	root   user.User
	// create, del, and retrieve run their routes as the API layer binds them.
	create   freighter.UnaryHandler[apipolicy.CreateRequest, apipolicy.CreateResponse]
	del      freighter.UnaryHandler[apipolicy.DeleteRequest, struct{}]
	dispatch freighter.UnaryHandler[apipolicy.DispatchRequest, struct{}]
	retrieve freighter.UnaryHandler[
		apipolicy.RetrieveRequest,
		apipolicy.RetrieveResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	creds := auth.Credentials{Username: "root", Password: "p"}
	svc = MustOpen(svctestutil.OpenLayer(
		ctx,
		node,
		service.LayerConfig{RootCredentials: creds},
	))
	root, _ = MustSucceed2(user.ProvisionRoot(
		ctx,
		svc.User,
		svc.Password,
		creds,
		svc.RBAC.Builtin.OwnerKey,
	))
	apiSvc = MustSucceed(apipolicy.NewService(apipolicy.ServiceConfig{RBAC: svc.RBAC}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
})

// createUser creates a user that can do nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// createRole creates a role that is not built in.
func createRole(ctx SpecContext) role.Role {
	GinkgoHelper()
	r := role.Role{Name: "role-" + uuid.New().String()}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.RBAC.Role.NewWriter(tx, false).Create(ctx, &r)
	})).To(Succeed())
	return r
}

// grantOn grants the action on the objects to the subject through a new policy on its
// own role, and returns the policy.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	action access.Action,
	objects ...resource.ID,
) policy.Policy {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	return MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, key, actions, objects...),
	)
}
