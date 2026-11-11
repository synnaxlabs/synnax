// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiproject "github.com/synnaxlabs/synnax/pkg/api/project"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/project/bundle"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPIProject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Project Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	rbacSvc    *rbac.Service
	groupSvc   *group.Service
	projectSvc *project.Service
	logSvc     *log.Service
	panelSvc   *panel.Service
	userSvc    *user.Service
	apiSvc     *apiproject.Service
	// create, del, dispatch, importBundle, retrieve, and export run their routes as
	// the API layer binds them.
	create freighter.UnaryHandler[
		apiproject.CreateRequest,
		apiproject.CreateResponse,
	]
	del          freighter.UnaryHandler[apiproject.DeleteRequest, struct{}]
	dispatch     freighter.UnaryHandler[apiproject.DispatchRequest, struct{}]
	importBundle freighter.UnaryHandler[
		apiproject.ImportRequest,
		apiproject.ImportResponse,
	]
	retrieve freighter.UnaryHandler[
		apiproject.RetrieveRequest,
		apiproject.RetrieveResponse,
	]
	export freighter.UnaryHandler[apiproject.ExportRequest, apiproject.ExportResponse]
	author user.User
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
	imexSvc := imex.NewService()
	panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB: db,
	}))
	logSvc = MustOpen(log.OpenService(ctx, log.ServiceConfig{
		DB:   db,
		ImEx: imexSvc,
	}))
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB: db,
	}))
	resourceSvc := MustSucceed(resource.NewService(resource.ServiceConfig{
		Providers: map[resource.Type]resource.Provider{
			resource.TypeGroup:   groupSvc.Resources(),
			resource.TypeLog:     logSvc.Resources(),
			resource.TypePanel:   panelSvc.Resources(),
			resource.TypeProject: projectSvc.Resources(),
		},
	}))
	bundleSvc := MustSucceed(bundle.NewService(bundle.Config{
		Resource: resourceSvc,
		Project:  projectSvc,
		Group:    groupSvc,
		Panel:    panelSvc,
		ImEx:     imexSvc,
	}))
	apiSvc = MustSucceed(apiproject.NewService(apiproject.ServiceConfig{
		Project: projectSvc,
		Bundle:  bundleSvc,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: rbacSvc,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	importBundle = bind.WritePrepared(binder, apiSvc.PrepareImport, apiSvc.Import)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
	export = bind.Read(binder, apiSvc.Export)
	author = MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
		Name: "test",
	}))
})

func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
		Name: "import-" + uuid.New().String(),
	}))
}

func createProject(ctx SpecContext, name string) project.Project {
	GinkgoHelper()
	p := project.Project{Name: name}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &p)
	})).To(Succeed())
	return p
}

func createLog(ctx SpecContext, proj project.Key, name string) log.Log {
	GinkgoHelper()
	l := defaults.Fill(log.Log{Project: &proj, Name: name})
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return logSvc.NewWriter(tx).Create(ctx, &l)
	})).To(Succeed())
	return l
}

// grantOn grants the action on the objects to the subject.
func grantOn(
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

func grantRetrieveOn(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionRetrieve, objects...)
}
