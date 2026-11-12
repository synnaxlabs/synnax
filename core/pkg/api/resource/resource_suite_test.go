// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiresource "github.com/synnaxlabs/synnax/pkg/api/resource"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Resource Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db  *gorp.DB
	svc *service.Layer
	// search and retrieve run their routes as the API layer binds them.
	search freighter.UnaryHandler[
		apiresource.SearchRequest,
		apiresource.SearchResponse,
	]
	retrieve freighter.UnaryHandler[
		apiresource.RetrieveRequest,
		apiresource.RetrieveResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc := MustSucceed(apiresource.NewService(apiresource.ServiceConfig{
		Resource: svc.Resource,
	}))
	binder := MustSucceed(
		bind.NewService(bind.ServiceConfig{
			DB:   node.DB,
			RBAC: svc.RBAC,
			Held: apiauth.HeldTables(),
		}),
	)
	search = bind.Read(binder, apiSvc.Search)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
})

func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

func grant(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	MustSucceed(rbactestutil.Grant(
		ctx,
		db,
		svc.RBAC,
		svc.User,
		MustSucceed(user.KeyFromResourceID(subject)),
		[]access.Action{access.ActionRetrieve},
		objects...,
	))
}
