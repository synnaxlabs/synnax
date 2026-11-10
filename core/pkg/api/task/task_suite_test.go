// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPITask(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Task Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	node   testutil.Node
	svc    *service.Layer
	binder *bind.Service
	// create, copyTask, dispatch, and retrieve run their routes as the API layer binds
	// them.
	create   freighter.UnaryHandler[apitask.CreateRequest, apitask.CreateResponse]
	copyTask freighter.UnaryHandler[apitask.CopyRequest, apitask.CopyResponse]
	dispatch freighter.UnaryHandler[apitask.DispatchRequest, apitask.DispatchResponse]
	retrieve freighter.UnaryHandler[
		apitask.RetrieveRequest,
		apitask.RetrieveResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc := MustSucceed(apitask.NewService(apitask.ServiceConfig{
		DB:      node.DB,
		Task:    svc.Task,
		Status:  svc.Status,
		RBAC:    svc.RBAC,
		Stopper: svc.TaskStopper,
	}))
	binder = MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	copyTask = bind.Write(binder, apiSvc.Copy)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
})

// createUser creates a user that can do nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(
		rbactestutil.CreateUser(ctx, node.DB, svc.RBAC, svc.User, user.User{
			Name: "user-" + uuid.New().String(),
		}),
	)
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
		rbactestutil.Grant(ctx, node.DB, svc.RBAC, svc.User, key, actions, objects...),
	)
}
