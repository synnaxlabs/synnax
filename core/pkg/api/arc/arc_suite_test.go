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
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
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
	db       *gorp.DB
	otg      *ontology.Ontology
	rbacSvc  *rbac.Service
	userSvc  *user.Service
	arcSvc   *arc.Service
	apiSvc   *apiarc.Service
	author   user.User
	rackSvc  *rack.Service
	testRack *rack.Rack
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "suite-root", Password: "p"},
		),
	)
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		User:     userSvc,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		Ontology:            otg,
		HostProvider:        testutil.NewStaticHostProvider(1),
		Status:              statusSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
		DriverRole:          rbacSvc.Builtin.DriverKey,
	}))
	imexSvc := imex.NewService()
	arcTable := MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
	arcTaskSvc := MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
	configs := MustSucceed(taskconfig.NewRegistry(arcTaskSvc.Stores()...))
	taskSvc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:           db,
		Ontology:     otg,
		Rack:         rackSvc,
		Status:       statusSvc,
		ImEx:         imexSvc,
		Configs:      configs,
		ImExExcluded: []string{arctask.Type},
	}))
	channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           db,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Status:       statusSvc,
	}))
	arcSvc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
		DB:         db,
		Table:      arcTable,
		Ontology:   otg,
		Channel:    channelSvc,
		Task:       taskSvc,
		MatchTasks: arcTaskSvc.MatchArcs,
		Status:     statusSvc,
		ImEx:       imexSvc,
	}))
	apiSvc = MustSucceed(apiarc.NewService(apiarc.ServiceConfig{
		RBAC:   rbacSvc,
		Arc:    arcSvc,
		Status: statusSvc,
	}))
	author = MustSucceed(
		userSvc.NewWriter(nil).Create(ctx, user.User{Name: "test"}),
	)
	testRack = &rack.Rack{Name: "API Test Rack"}
	Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())
})

// grantOn grants the action on the objects to the subject through a new role, which
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
	r := MustSucceed(
		rbactestutil.Grant(ctx, rbacSvc, userSvc, key, actions, objects...),
	)
	DeferCleanup(func(ctx SpecContext) {
		Expect(rbacSvc.Role.NewWriter(nil, false).Delete(ctx, r)).To(Succeed())
	})
}

func grantUpdateOn(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionUpdate, objects...)
}
