// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apipanel "github.com/synnaxlabs/synnax/pkg/api/panel"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPIPanel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Panel Suite", SuiteConfig())
}

var (
	db       *gorp.DB
	groupSvc *group.Service
	rbacSvc  *rbac.Service
	panelSvc *panel.Service
	userSvc  *user.Service
	apiSvc   *apipanel.Service
	// create, del, dispatch, and retrieve run their routes as the API layer binds them.
	create   freighter.UnaryHandler[apipanel.CreateRequest, apipanel.CreateResponse]
	del      freighter.UnaryHandler[apipanel.DeleteRequest, struct{}]
	dispatch freighter.UnaryHandler[apipanel.DispatchRequest, apipanel.DispatchResponse]
	retrieve freighter.UnaryHandler[
		apipanel.RetrieveRequest,
		apipanel.RetrieveResponse,
	]
	author     user.User
	proj       project.Project
	projectSvc *project.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Password: passwordSvc,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	rbacSvc = MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     db,
		User:   userSvc,
		Status: statusSvc,
	})).RBAC
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "suite-root", Password: "p"},
			rbacSvc.Builtin.OwnerKey,
		),
	)
	panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB: db,
	}))
	apiSvc = MustSucceed(apipanel.NewService(apipanel.ServiceConfig{
		Panel: panelSvc,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: rbacSvc,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
	author = MustSucceed(
		rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{Name: "test"}),
	)
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB: db,
	}))
	proj = project.Project{Name: "panel-project"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &proj)
	})).To(Succeed())
})

// newUser creates a fresh user. RBAC state is committed and shared across specs, so
// specs that grant a type-level permission (e.g. create on the panel type) must use a
// dedicated subject, or the grant would leak into specs asserting that permission is
// absent.
func newUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
		Name: uuid.New().String(),
	}))
}

// grant grants the action on the objects to the subject.
func grant(
	ctx SpecContext,
	subject resource.ID,
	action access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	MustSucceed(rbactestutil.Grant(ctx, db, rbacSvc, userSvc, key, actions, objects...))
}

// grantCreate lets subject create panels in the suite project.
func grantCreate(ctx SpecContext, subject resource.ID) {
	GinkgoHelper()
	grant(ctx, subject, access.ActionCreate, resource.ID{Type: resource.TypePanel})
	grant(ctx, subject, access.ActionUpdate, proj.ResourceID())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
