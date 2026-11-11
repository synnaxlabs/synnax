// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package chassis_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichassis "github.com/synnaxlabs/synnax/pkg/api/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPINIChassis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API NI Chassis Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db     *gorp.DB
	svc    *service.Layer
	apiSvc *apichassis.Service
	// These handlers run their routes as the API layer binds them.
	create   freighter.UnaryHandler[apichassis.CreateRequest, apichassis.CreateResponse]
	update   freighter.UnaryHandler[apichassis.UpdateRequest, struct{}]
	del      freighter.UnaryHandler[apichassis.DeleteRequest, struct{}]
	retrieve freighter.UnaryHandler[
		apichassis.RetrieveRequest,
		apichassis.RetrieveResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apichassis.NewService(apichassis.ServiceConfig{
		Chassis: svc.NI.Chassis,
		Status:  svc.Status,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	update = bind.Write(binder, apiSvc.Update)
	del = bind.Write(binder, apiSvc.Delete)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
})

// createUser creates a user that can do nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// grant grants the actions on the objects to u.
func grant(
	ctx SpecContext,
	u user.User,
	actions []access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, u.Key, actions, objects...),
	)
}
