// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPISchematicSymbol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Schematic Symbol Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db        *gorp.DB
	rbacSvc   *rbac.Service
	groupSvc  *group.Service
	symbolSvc *symbol.Service
	userSvc   *user.Service
	apiSvc    *apisymbol.Service
	// The handlers below run their routes as the API layer binds them.
	create freighter.UnaryHandler[
		apisymbol.CreateRequest,
		apisymbol.CreateResponse,
	]
	dispatch    freighter.UnaryHandler[apisymbol.DispatchRequest, struct{}]
	del         freighter.UnaryHandler[apisymbol.DeleteRequest, struct{}]
	deleteGroup freighter.UnaryHandler[apisymbol.DeleteGroupRequest, struct{}]
	importGroup freighter.UnaryHandler[
		apisymbol.ImportGroupRequest,
		apisymbol.ImportGroupResponse,
	]
	retrieve freighter.UnaryHandler[
		apisymbol.RetrieveRequest,
		apisymbol.RetrieveResponse,
	]
	exportGroup freighter.UnaryHandler[
		apisymbol.ExportGroupRequest,
		apisymbol.ExportGroupResponse,
	]
	author user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Password: passwordSvc,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	rbacSvc = MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     db,
		User:   userSvc,
		Status: statusSvc,
	})).RBAC
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "suite-root", Password: "p"},
			rbacSvc.Builtin.OwnerKey,
		),
	)
	symbolSvc = MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
		DB:    db,
		Group: groupSvc,
		ImEx:  imex.NewService(),
	}))
	apiSvc = MustSucceed(apisymbol.NewService(apisymbol.ServiceConfig{
		Symbol: symbolSvc,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: rbacSvc,
		Held: apiauth.HeldTables(),
	}))
	create = bind.Write(binder, apiSvc.Create)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	del = bind.Write(binder, apiSvc.Delete)
	deleteGroup = bind.Write(binder, apiSvc.DeleteGroup)
	importGroup = bind.Write(binder, apiSvc.ImportGroup)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
	exportGroup = bind.Read(binder, apiSvc.ExportGroup)
	author = MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
		Name: "test",
	}))
})

// createGroup creates a top-level symbol group. The write commits at once so the api
// enforcers, which read committed state, observe the new resource.
func createGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Scope: resource.TypeSchematicSymbol}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return groupSvc.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	return g
}

// groupSymbols returns the resource IDs of the symbols in the group with the given key.
func groupSymbols(ctx SpecContext, key group.Key) []resource.ID {
	GinkgoHelper()
	var symbols []symbol.Symbol
	Expect(symbolSvc.NewRetrieve().
		Where(symbol.MatchGroups(key)).
		Entries(&symbols).
		Exec(ctx, db)).To(Succeed())
	return lo.Map(symbols, func(sym symbol.Symbol, _ int) resource.ID {
		return sym.ResourceID()
	})
}

// newSymbol returns an unsaved symbol a create request can carry.
func newSymbol(name string) symbol.Symbol {
	return defaults.Fill(
		symbol.Symbol{Name: name, Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"}},
	)
}

// createSymbol creates a symbol in g, committing it for the same reason createGroup
// does.
func createSymbol(ctx SpecContext, g group.Group, name string) symbol.Symbol {
	GinkgoHelper()
	sym := newSymbol(name)
	sym.Group = &g.Key
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return symbolSvc.NewWriter(tx).Create(ctx, &sym)
	})).To(Succeed())
	return sym
}

// createUser creates a user to author a request. Specs granting on an object the whole
// suite shares, such as the symbol type, take a user of their own, so the grant cannot
// reach another spec.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
		Name: "test-" + uuid.New().String(),
	}))
}

// grantOn grants the action on the objects to the subject.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	action access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	MustSucceed(rbactestutil.Grant(ctx, db, rbacSvc, userSvc, key, actions, objects...))
}

func grantCreateOn(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionCreate, objects...)
}

func grantRetrieveOn(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionRetrieve, objects...)
}

func grantUpdateOn(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionUpdate, objects...)
}

func grantDeleteOn(ctx SpecContext, subject resource.ID, objects ...resource.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionDelete, objects...)
}
