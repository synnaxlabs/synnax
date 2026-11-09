// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIImEx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API ImEx Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db       *gorp.DB
	rbacSvc  *rbac.Service
	userSvc  *user.Service
	imexSvc  *imex.Service
	groupSvc *group.Service
	importer *recordingImporter
	root     user.User
	// ch is a channel that imported logs use.
	ch channel.Key
	// proj is the project that imported logs sit in.
	proj project.Project
	// importEnvelope and export run their routes as the API layer binds them.
	importEnvelope freighter.UnaryHandler[
		apiimex.ImportRequest,
		apiimex.ImportResponse,
	]
	export freighter.UnaryHandler[apiimex.ExportRequest, apiimex.ExportResponse]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
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
			auth.Credentials{Username: "api-imex-suite-root", Password: "p"},
			rbacSvc.Builtin.OwnerKey,
		),
	)
	imexSvc = imex.NewService()
	importer = &recordingImporter{}
	Expect(imexSvc.RegisterImporter(string(importer.Type()), importer)).To(Succeed())
	Expect(imexSvc.RegisterExporter(namingExporter{})).To(Succeed())
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
		DB:    db,
		Group: groupSvc,
		ImEx:  imexSvc,
	}))
	channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	ch = MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 1))[0]
	projectSvc := MustOpen(project.OpenService(ctx, project.ServiceConfig{DB: db}))
	proj.Name = "Imports"
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &proj)
	})).To(Succeed())
	MustOpen(log.OpenService(ctx, log.ServiceConfig{DB: db, ImEx: imexSvc}))
	apiSvc := MustSucceed(apiimex.NewService(apiimex.ServiceConfig{ImEx: imexSvc}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: rbacSvc,
		Held: apiauth.HeldTables(),
	}))
	importEnvelope = bind.Write(binder, apiSvc.Import)
	export = bind.Read(binder, apiSvc.Export)
	Expect(userSvc.NewRetrieve().
		Where(user.MatchRootUser(true)).
		Entry(&root).Exec(ctx, db)).To(Succeed())
})

// rootCtx returns a freighter.Context with the suite-level root user installed as the
// request subject. The root user holds the Owner role, so every RBAC enforcement call
// passes.
func rootCtx(ctx SpecContext) freighter.Context { return subjectCtx(ctx, root) }

// subjectCtx returns a freighter.Context with u installed as the request subject.
func subjectCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.ResourceID())
	return fctx
}
