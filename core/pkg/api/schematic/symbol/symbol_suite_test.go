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
	"github.com/synnaxlabs/gorp"
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
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPISchematicSymbol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Schematic Symbol Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	rbacSvc   *rbac.Service
	groupSvc  *group.Service
	symbolSvc *symbol.Service
	userSvc   *user.Service
	apiSvc    *apisymbol.Service
	author    user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db := DeferClose(gorp.Wrap(memkv.New()))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Password: passwordSvc,
	}))
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "suite-root", Password: "p"},
		),
	)
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	rbacSvc = MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     db,
		User:   userSvc,
		Status: statusSvc,
	})).RBAC
	symbolSvc = MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
		DB:    db,
		Group: groupSvc,
		ImEx:  imex.NewService(),
	}))
	apiSvc = MustSucceed(apisymbol.NewService(apisymbol.ServiceConfig{
		Symbol: symbolSvc,
		RBAC:   rbacSvc,
	}))
	author = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Name: "test",
	}))
})

// createGroup creates a top-level symbol group. It writes outside a transaction so the
// api enforcers, which read committed state, observe the new resource.
func createGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Scope: resource.TypeSchematicSymbol}
	Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

// newSymbol returns an unsaved symbol a create request can carry.
func newSymbol(name string) symbol.Symbol {
	return symbol.Symbol{Name: name, Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"}}
}

// createSymbol creates a symbol in g, committing it for the same reason createGroup
// does.
func createSymbol(ctx SpecContext, g group.Group, name string) symbol.Symbol {
	GinkgoHelper()
	sym := newSymbol(name)
	sym.Group = &g.Key
	Expect(symbolSvc.NewWriter(nil).Create(ctx, &sym)).To(Succeed())
	return sym
}

// createUser creates a user to author a request. Specs granting on an object the whole
// suite shares, such as the symbol type, take a user of their own, so the grant cannot
// reach another spec.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Name: "test-" + uuid.New().String(),
	}))
}

// grantOn grants the action on the objects to the subject through a new role.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	action access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	actions := []access.Action{action}
	MustSucceed(rbactestutil.Grant(ctx, rbacSvc, userSvc, key, actions, objects...))
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
