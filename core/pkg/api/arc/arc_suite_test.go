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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	apiarcdirectory "github.com/synnaxlabs/synnax/pkg/api/arc/directory"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIArc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Arc Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	node       testutil.Node
	db         *gorp.DB
	rbacSvc    *rbac.Service
	userSvc    *user.Service
	arcSvc     *arc.Service
	channelSvc *channel.Service
	taskSvc    *task.Service
	apiSvc     *apiarc.Service
	// The routes, bound as the API layer binds them.
	create     freighter.UnaryHandler[apiarc.CreateRequest, apiarc.CreateResponse]
	del        freighter.UnaryHandler[apiarc.DeleteRequest, struct{}]
	dispatch   freighter.UnaryHandler[apiarc.DispatchRequest, struct{}]
	updateTask freighter.UnaryHandler[
		apiarc.UpdateTaskRequest,
		apiarc.UpdateTaskResponse,
	]
	createDirectory freighter.UnaryHandler[
		apiarc.CreateDirectoryRequest,
		apiarc.CreateDirectoryResponse,
	]
	dispatchDirectory freighter.UnaryHandler[
		apiarc.DispatchDirectoryRequest,
		struct{},
	]
	deleteDirectory freighter.UnaryHandler[apiarc.DeleteDirectoryRequest, struct{}]
	retrieve        freighter.UnaryHandler[
		apiarc.RetrieveRequest,
		apiarc.RetrieveResponse,
	]
	retrieveDirectories freighter.UnaryHandler[
		apiarcdirectory.RetrieveRequest,
		apiarcdirectory.RetrieveResponse,
	]
	author   user.User
	rackSvc  *rack.Service
	testRack *rack.Rack
	// root is the directory that holds every Arc.
	root directory.Key
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	db = node.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Password: passwordSvc,
	}))
	roles := MustOpen(rbac.OpenRoles(ctx, rbac.RolesConfig{DB: db}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		HostKey:             1,
		Status:              statusSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
		DriverRole:          roles.Builtin.DriverKey,
	}))
	rbacSvc = MustSucceed(rbac.NewService(ctx, rbac.ServiceConfig{
		DB:    db,
		Roles: roles,
		User:  userSvc,
		Rack:  rackSvc,
	}))
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "suite-root", Password: "p"},
			roles.Builtin.OwnerKey,
		),
	)
	imexSvc := imex.NewService()
	arcTable := MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
	arcTaskSvc := MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
	configs := MustSucceed(taskconfig.NewRegistry(arcTaskSvc.Stores()...))
	taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:           db,
		Rack:         rackSvc,
		Status:       statusSvc,
		ImEx:         imexSvc,
		Configs:      configs,
		ImExExcluded: []string{arctask.Type},
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	arcSvc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
		DB:         db,
		Table:      arcTable,
		Channel:    channelSvc,
		Task:       taskSvc,
		MatchTasks: arcTaskSvc.MatchArcs,
		Status:     statusSvc,
		ImEx:       imexSvc,
	}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	rangerSvc := MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
		DB:    db,
		Group: groupSvc,
	}))
	aliasSvc := MustOpen(alias.OpenService(ctx, alias.ServiceConfig{
		DB:              db,
		Channel:         channelSvc,
		ParentRetriever: rangerSvc,
	}))
	apiChannelSvc := MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
		DB:      db,
		RBAC:    rbacSvc,
		Channel: channelSvc,
		Status:  statusSvc,
		Alias:   aliasSvc,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: rbacSvc,
		Held: apiauth.HeldTables(),
	}))
	apiSvc = MustSucceed(apiarc.NewService(apiarc.ServiceConfig{
		DB:              db,
		Arc:             arcSvc,
		Status:          statusSvc,
		RBAC:            rbacSvc,
		DispatchChannel: bind.Write(binder, apiChannelSvc.Dispatch),
	}))
	create = bind.Write(binder, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	updateTask = bind.Write(binder, apiSvc.UpdateTask)
	createDirectory = bind.Write(binder, apiSvc.CreateDirectory)
	dispatchDirectory = bind.Write(binder, apiSvc.DispatchDirectory)
	deleteDirectory = bind.Write(binder, apiSvc.DeleteDirectory)
	retrieve = apiSvc.Compiled(bind.Read(binder, apiSvc.Retrieve))
	retrieveDirectories = bind.Read(binder, apiSvc.RetrieveDirectories)
	var r directory.Directory
	Expect(arcSvc.NewDirectoryRetrieve().
		Where(directory.MatchNested(false)).
		Entry(&r).
		Exec(ctx, db)).To(Succeed())
	root = r.Key
	author = MustSucceed(
		rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{Name: "test"}),
	)
	testRack = &rack.Rack{Name: "API Test Rack"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rackSvc.NewWriter(tx).Create(ctx, testRack)
	})).To(Succeed())
})

// grantOn grants the action on the objects to the subject through a new policy, which
// is deleted on spec cleanup so a grant cannot leak into later specs.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	action access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	p := MustSucceed(
		rbactestutil.Grant(ctx, db, rbacSvc, userSvc, key, actions, objects...),
	)
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rbacSvc.Policy.NewWriter(tx, false).Delete(ctx, p.Key)
		})).To(Succeed())
	})
}

func grantUpdateOn(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionUpdate, objects...)
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

// textOf returns the text of the text-mode Arc b, failing the spec otherwise.
func textOf(b arc.Base) text.Text {
	GinkgoHelper()
	src, ok := b.Source.Variant.(arc.TextSource)
	Expect(ok).To(BeTrue(), "expected a text source, got %T", b.Source.Variant)
	return src.Text
}

// graphOf returns the graph of the graph-mode Arc b, failing the spec otherwise.
func graphOf(b arc.Base) graph.Graph {
	GinkgoHelper()
	src, ok := b.Source.Variant.(arc.GraphSource)
	Expect(ok).To(BeTrue(), "expected a graph source, got %T", b.Source.Variant)
	return src.Graph
}
