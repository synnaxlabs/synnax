// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apilog "github.com/synnaxlabs/synnax/pkg/api/log"
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

func TestAPILog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Log Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db     *gorp.DB
	svc    *service.Layer
	apiSvc *apilog.Service
	// create, del, dispatch, and retrieve run their routes as the API layer binds them.
	create   freighter.UnaryHandler[apilog.CreateRequest, apilog.CreateResponse]
	del      freighter.UnaryHandler[apilog.DeleteRequest, struct{}]
	dispatch freighter.UnaryHandler[apilog.DispatchRequest, apilog.DispatchResponse]
	retrieve freighter.UnaryHandler[
		apilog.RetrieveRequest,
		apilog.RetrieveResponse,
	]
	proj project.Project
	// author can do nothing until a spec grants it access.
	author user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apilog.NewService(apilog.ServiceConfig{
		Log: svc.Log,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
	proj = project.Project{Name: "test-project"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Project.NewWriter(tx).Create(ctx, &proj)
	})).To(Succeed())
	author = MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "author",
	}))
})

// newUser creates a user with no policies.
func newUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// newCreator creates a user that may create logs in the suite project.
func newCreator(ctx SpecContext) user.User {
	GinkgoHelper()
	u := newUser(ctx)
	grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{Type: resource.TypeLog})
	grant(ctx, access.ActionUpdate, u.ResourceID(), project.ResourceID(proj.Key))
	return u
}

// grant grants the action on the objects to the subject.
func grant(
	ctx SpecContext,
	action access.Action,
	subject resource.ID,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, key, actions, objects...),
	)
}
