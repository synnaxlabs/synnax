// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPIStatus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Status Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db        *gorp.DB
	otg       *ontology.Ontology
	rbacSvc   *rbac.Service
	statusSvc *status.Service
	labelSvc  *label.Service
	apiSvc    *apistatus.Service
	author    user.User
	userSvc   *user.Service
	rackSvc   *rack.Service
	devices   *gorp.Table[device.Key, device.Device]
	tasks     *gorp.Table[task.Key, task.Task]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg,
	}))
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Password: passwordSvc,
	}))
	labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db, Ontology: otg, Group: g,
	}))
	// A status owner must exist, so the owner tables must be open.
	devices = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[device.Key, device.Device]{
		DB: db, Name: string(resource.TypeDevice),
	}))
	tasks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[task.Key, task.Task]{
		DB: db, Name: string(resource.TypeTask),
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db, Ontology: otg,
	}))
	svcs := MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:       db,
		Ontology: otg,
		User:     userSvc,
		Status:   statusSvc,
	}))
	rbacSvc, rackSvc = svcs.RBAC, svcs.Rack
	apiSvc = MustSucceed(apistatus.NewService(apistatus.ServiceConfig{
		Status: statusSvc,
		RBAC:   rbacSvc,
	}))
	author = MustSucceed(
		userSvc.NewWriter(nil).Create(ctx, user.User{Name: "test"}),
	)
})

// freshUser creates a user with no role assignments. Use this for "unauthorized"
// specs so accumulated grants on the shared author don't leak in.
func freshUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(
		userSvc.NewWriter(nil).
			Create(ctx, user.User{Name: "anon-" + uuid.New().String()}),
	)
}

// grantOn grants the actions on the objects to the subject through a new role.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	actions []access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	MustSucceed(rbactestutil.Grant(ctx, rbacSvc, userSvc, key, actions, objects...))
}
