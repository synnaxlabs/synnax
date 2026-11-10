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
	// The handlers run their routes as the API layer binds them.
	create   freighter.UnaryHandler[apiicd.CreateRequest, apiicd.CreateResponse]
	retrieve freighter.UnaryHandler[apiicd.RetrieveRequest, apiicd.RetrieveResponse]
	check    freighter.UnaryHandler[apiicd.CheckRequest, apiicd.CheckResponse]
	rename   freighter.UnaryHandler[apiicd.RenameRequest, struct{}]
	imp      freighter.UnaryHandler[apiicd.ImportRequest, apiicd.ImportResponse]
	del      freighter.UnaryHandler[apiicd.DeleteRequest, struct{}]
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
	rename = bind.Write(binder, apiSvc.Rename)
	imp = bind.Write(binder, apiSvc.Import)
	del = bind.Write(binder, apiSvc.Delete)
})

// newUser creates a user with no policies.
func newUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// grant grants the action on the objects to u.
func grant(
	ctx SpecContext,
	u user.User,
	action access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	actions := []access.Action{action}
	MustSucceed(
		rbactestutil.Grant(ctx, db, svc.RBAC, svc.User, u.Key, actions, objects...),
	)
}

func createICD(ctx SpecContext, name string) icd.ICD {
	GinkgoHelper()
	ic := icd.ICD{Name: name}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.ICD.NewWriter(tx).Create(ctx, &ic)
	})).To(Succeed())
	return ic
}

func retrieveICD(ctx SpecContext, key icd.Key) (icd.ICD, error) {
	var ic icd.ICD
	err := svc.ICD.NewRetrieve().
		Where(icd.MatchKeys(key)).
		Entry(&ic).
		Exec(ctx, db)
	return ic, err
}
