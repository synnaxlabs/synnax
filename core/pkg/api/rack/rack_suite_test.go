// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIRack(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Rack Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc       *service.Layer
	apiSvc    *apirack.Service
	statusAPI *apistatus.Service
	// create and rename run their routes as the API layer binds them.
	create freighter.UnaryHandler[apirack.CreateRequest, apirack.CreateResponse]
	rename freighter.UnaryHandler[apirack.RenameRequest, struct{}]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apirack.NewService(apirack.ServiceConfig{
		Rack:   svc.Rack,
		Task:   svc.Task,
		Status: svc.Status,
		RBAC:   svc.RBAC,
		Auth:   svc.Auth,
	}))
	statusAPI = MustSucceed(apistatus.NewService(apistatus.ServiceConfig{
		Status: svc.Status,
		RBAC:   svc.RBAC,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: svc.RBAC,
	}))
	create = bind.Write(binder, apiSvc.Create)
	rename = bind.Write(binder, apiSvc.Rename)
})

// createUser creates a user that can do nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
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
	MustSucceed(rbactestutil.Grant(ctx, svc.RBAC, svc.User, key, actions, objects...))
}
