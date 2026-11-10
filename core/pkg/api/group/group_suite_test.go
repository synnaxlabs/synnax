// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apigroup "github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Group Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db     *gorp.DB
	svc    *service.Layer
	apiSvc *apigroup.Service
	// create, del, and dispatch run their routes as the API layer binds them.
	create   freighter.UnaryHandler[apigroup.CreateRequest, apigroup.CreateResponse]
	del      freighter.UnaryHandler[apigroup.DeleteRequest, struct{}]
	dispatch freighter.UnaryHandler[apigroup.DispatchRequest, struct{}]
	// retrieveGroups and retrieveAncestors run their routes as the API layer binds
	// them.
	retrieveGroups freighter.UnaryHandler[
		apigroup.RetrieveRequest,
		apigroup.RetrieveResponse,
	]
	retrieveAncestors freighter.UnaryHandler[
		apigroup.RetrieveAncestorsRequest,
		apigroup.RetrieveAncestorsResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apigroup.NewService(apigroup.ServiceConfig{
		Group: svc.Group,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	retrieveGroups = bind.Read(binder, apiSvc.Retrieve)
	retrieveAncestors = bind.Read(binder, apiSvc.RetrieveAncestors)
})

// grant returns a policy that allows actions on objects.
func grant(actions []access.Action, objects ...resource.ID) policy.Policy {
	return policy.Policy{
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: actions,
	}
}

// newUser creates a user that holds policies.
func newUser(ctx SpecContext, policies ...policy.Policy) user.User {
	GinkgoHelper()
	u := MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
	for _, p := range policies {
		MustSucceed(
			rbactestutil.Grant(
				ctx,
				db,
				svc.RBAC,
				svc.User,
				u.Key,
				p.Actions,
				p.Objects...,
			),
		)
	}
	return u
}
