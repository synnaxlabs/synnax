// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgorp "github.com/synnaxlabs/freighter/gorp"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apisecret "github.com/synnaxlabs/synnax/pkg/api/secret"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPISecret(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Secret Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db  *gorp.DB
	svc *service.Layer
	// The handlers run the routes as the API layer binds them.
	set      freighter.UnaryHandler[apisecret.SetRequest, apisecret.SetResponse]
	retrieve freighter.UnaryHandler[
		apisecret.RetrieveRequest,
		apisecret.RetrieveResponse,
	]
	del         freighter.UnaryHandler[apisecret.DeleteRequest, struct{}]
	dispatch    freighter.UnaryHandler[apisecret.DispatchRequest, struct{}]
	createGrant freighter.UnaryHandler[
		apisecret.GrantCreateRequest,
		apisecret.GrantCreateResponse,
	]
	resolve freighter.UnaryHandler[apisecret.ResolveRequest, apisecret.ResolveResponse]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc := MustSucceed(apisecret.NewService(apisecret.ServiceConfig{
		Secret: svc.Secret,
		RBAC:   svc.RBAC,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	set = bind.Write(binder, apiSvc.Set)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	createGrant = fgorp.CreateWriteUnaryHandler(node.DB, apiSvc.CreateGrant)
	resolve = apiSvc.Resolve
})

// createUser creates a user whose role allows nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// allow gives the subject the actions on the objects.
func allow(
	ctx SpecContext,
	subject resource.ID,
	actions []access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, key, actions, objects...),
	)
}
