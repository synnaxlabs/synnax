// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apialias "github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIAlias(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Alias Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db  *gorp.DB
	svc *service.Layer
	// set, dispatch, del, resolve, list, and retrieve run their routes as the API
	// layer binds them.
	set      freighter.UnaryHandler[apialias.SetRequest, struct{}]
	dispatch freighter.UnaryHandler[apialias.DispatchRequest, struct{}]
	del      freighter.UnaryHandler[apialias.DeleteRequest, struct{}]
	resolve  freighter.UnaryHandler[apialias.ResolveRequest, apialias.ResolveResponse]
	list     freighter.UnaryHandler[apialias.ListRequest, apialias.ListResponse]
	retrieve freighter.UnaryHandler[
		apialias.RetrieveRequest,
		apialias.RetrieveResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc := MustSucceed(apialias.NewService(apialias.ServiceConfig{
		Alias: svc.Alias,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	set = bind.Write(binder, apiSvc.Set)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	del = bind.Write(binder, apiSvc.Delete)
	resolve = bind.Read(binder, apiSvc.Resolve)
	list = bind.Read(binder, apiSvc.List)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
})

// createUser creates a user that can do nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// grant grants u the action on the objects.
func grant(ctx SpecContext, u user.User, action access.Action, objects ...resource.ID) {
	GinkgoHelper()
	actions := []access.Action{action}
	MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, u.Key, actions, objects...),
	)
}

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}
