// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Channel Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	channelSvc *channel.Service
	groupSvc   *group.Service
	statusSvc  *status.Service
	rbacSvc    *rbac.Service
	userSvc    *user.Service
	apiSvc     *apichannel.Service
	dist       testutil.Node
	// The handlers run their routes as the API layer binds them.
	createChannels freighter.UnaryHandler[
		apichannel.CreateRequest,
		apichannel.CreateResponse,
	]
	deleteChannels freighter.UnaryHandler[apichannel.DeleteRequest, struct{}]
	dispatch       freighter.UnaryHandler[
		apichannel.DispatchRequest,
		apichannel.DispatchResponse,
	]
	retrieveChannels freighter.UnaryHandler[
		apichannel.RetrieveRequest,
		apichannel.RetrieveResponse,
	]
	author user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	dist = testutil.NewNode(ctx)
	db = dist.DB
	svc := MustOpen(svctestutil.OpenLayer(ctx, dist))
	channelSvc, groupSvc, statusSvc, rbacSvc = svc.Channel, svc.Group, svc.Status, svc.RBAC
	userSvc = svc.User
	apiSvc = MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
		DB:      db,
		RBAC:    rbacSvc,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	binder := MustSucceed(
		bind.NewService(bind.ServiceConfig{
			DB:   db,
			RBAC: rbacSvc,
			Held: apiauth.HeldTables(),
		}),
	)
	createChannels = bind.Write(binder, apiSvc.Create)
	deleteChannels = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	retrieveChannels = bind.Read(binder, apiSvc.Retrieve)
	author = MustSucceed(
		rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{Name: "test"}),
	)
})

// grantOn grants the actions on the objects to the subject.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	actions []access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	MustSucceed(rbactestutil.Grant(ctx, db, rbacSvc, userSvc, key, actions, objects...))
}
