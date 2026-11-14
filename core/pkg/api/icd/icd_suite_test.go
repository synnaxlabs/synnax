// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiicd "github.com/synnaxlabs/synnax/pkg/api/icd"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIICD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API ICD Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db  *gorp.DB
	svc *service.Layer
	// create, retrieve, check, dispatch, importFile, and del run their routes as the
	// API layer binds them.
	create     freighter.UnaryHandler[apiicd.CreateRequest, apiicd.CreateResponse]
	retrieve   freighter.UnaryHandler[apiicd.RetrieveRequest, apiicd.RetrieveResponse]
	check      freighter.UnaryHandler[apiicd.CheckRequest, apiicd.CheckResponse]
	dispatch   freighter.UnaryHandler[apiicd.DispatchRequest, struct{}]
	importFile freighter.UnaryHandler[apiicd.ImportRequest, apiicd.ImportResponse]
	del        freighter.UnaryHandler[apiicd.DeleteRequest, struct{}]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc := MustSucceed(apiicd.NewService(apiicd.ServiceConfig{ICD: svc.ICD}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
	check = bind.Read(binder, apiSvc.Check)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	importFile = bind.Write(binder, apiSvc.Import)
	del = bind.Write(binder, apiSvc.Delete)
})

func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "icd-" + uuid.New().String(),
	}))
}

func createICD(ctx SpecContext, name string) icd.ICD {
	GinkgoHelper()
	ic := icd.ICD{Name: name}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.ICD.NewWriter(tx).Create(ctx, &ic)
	})).To(Succeed())
	return ic
}

func retrieveICD(ctx SpecContext, key icd.Key) icd.ICD {
	GinkgoHelper()
	var ic icd.ICD
	Expect(svc.ICD.NewRetrieve().
		Where(icd.MatchKeys(key)).
		Entry(&ic).
		Exec(ctx, db)).To(Succeed())
	return ic
}

// grantOn grants the action on the given objects to the subject.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	action access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, key, actions, objects...),
	)
}
