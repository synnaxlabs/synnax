// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apicalculation "github.com/synnaxlabs/synnax/pkg/api/calculation"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPICalculation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Calculation Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db  *gorp.DB
	svc *service.Layer
	// create, del, dispatch, and retrieve run their routes as the API layer binds them.
	create freighter.UnaryHandler[
		apicalculation.CreateRequest,
		apicalculation.CreateResponse,
	]
	del      freighter.UnaryHandler[apicalculation.DeleteRequest, struct{}]
	dispatch freighter.UnaryHandler[apicalculation.DispatchRequest, struct{}]
	retrieve freighter.UnaryHandler[
		apicalculation.RetrieveRequest,
		apicalculation.RetrieveResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc := MustSucceed(apicalculation.NewService(apicalculation.ServiceConfig{
		Calculation: svc.Calculation,
		Status:      svc.Status,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
})

// createUser creates a user that can do nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// grant grants the actions on the objects to the subject.
func grant(
	ctx SpecContext,
	actions []access.Action,
	subject resource.ID,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, key, actions, objects...),
	)
}

// createVirtual creates a virtual float32 channel with a unique name.
func createVirtual(ctx SpecContext) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     "calc_" + uuid.New().String()[:8],
		DataType: telem.Float32T,
		Virtual:  true,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Channel.NewWriter(tx).Create(ctx, &ch)
	})).To(Succeed())
	return ch
}
