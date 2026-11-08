// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgorp "github.com/synnaxlabs/freighter/gorp"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIRanger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Ranger Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apiranger.Service
	// dispatch runs the Dispatch route in one transaction, as the API layer binds it.
	dispatch freighter.UnaryHandler[apiranger.DispatchRequest, struct{}]
	// renameRange and setRangeEnd run the Rename and SetEnd routes the same way.
	renameRange freighter.UnaryHandler[apiranger.RenameRequest, struct{}]
	setRangeEnd freighter.UnaryHandler[apiranger.SetEndRequest, struct{}]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apiranger.NewService(apiranger.ServiceConfig{
		RBAC:   svc.RBAC,
		Ranger: svc.Ranger,
	}))
	dispatch = fgorp.CreateWriteUnaryHandler(node.DB, apiSvc.Dispatch)
	renameRange = fgorp.CreateWriteUnaryHandler(node.DB, apiSvc.Rename)
	setRangeEnd = fgorp.CreateWriteUnaryHandler(node.DB, apiSvc.SetEnd)
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
