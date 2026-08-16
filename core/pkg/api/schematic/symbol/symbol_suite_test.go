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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPISchematicSymbol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Schematic Symbol Suite")
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
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:              db,
		Ontology:        otg,
		Group:           groupSvc,
		Search:          searchIdx,
		Auth:            authSvc,
		RootCredentials: auth.Credentials{Username: "suite-root", Password: "p"},
	}))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
		User:     userSvc,
	}))
	symbolSvc = MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
		ImEx:     imex.NewService(),
	}))
	apiSvc = MustSucceed(apisymbol.NewService(config.LayerConfig{
		Distribution: &distribution.Layer{DB: db},
		Service: &service.Layer{
			Schematic: &schematic.Service{Symbol: symbolSvc},
			RBAC:      rbacSvc,
		},
	}))
	Expect(searchIdx.Initialize(ctx)).To(Succeed())
	author = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Username: "test",
	}))
})

// createGroup creates a group under the ontology root. It writes outside a transaction
// so the api enforcers, which read committed state, observe the new resource.
func createGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	return MustSucceed(groupSvc.NewWriter(nil).Create(ctx, name, ontology.RootID))
}

// newSymbol returns an unsaved symbol a create request can carry.
func newSymbol(name string) symbol.Symbol {
	return symbol.Symbol{Name: name, Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"}}
}

// createSymbol creates a symbol under g, committing it for the same reason createGroup
// does.
func createSymbol(ctx SpecContext, g group.Group, name string) symbol.Symbol {
	GinkgoHelper()
	sym := newSymbol(name)
	Expect(symbolSvc.NewWriter(nil).Create(ctx, &sym, g.OntologyID())).To(Succeed())
	return sym
}

// createUser creates a user to author a request. Specs granting on an object the whole
// suite shares — the symbol type or the permanent group — take a user of their own, so
// the grant cannot reach another spec.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Username: "test-" + uuid.NewString(),
	}))
}

// grantOn grants the action on the given objects to the subject through a fresh role.
// Writes commit directly so the api enforcers, which read committed state with no
// transaction, observe them.
func grantOn(
	ctx SpecContext,
	subject ontology.ID,
	action access.Action,
	objects ...ontology.ID,
) {
	GinkgoHelper()
	roleWriter := rbacSvc.Role.NewWriter(nil, true)
	policyWriter := rbacSvc.Policy.NewWriter(nil, true)
	r := &role.Role{Name: string(action) + "-" + uuid.NewString(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    string(action) + "-policy-" + uuid.NewString(),
		Objects: objects,
		Actions: []access.Action{action},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())
}

func grantCreateOn(ctx SpecContext, subject ontology.ID, objects ...ontology.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionCreate, objects...)
}

func grantRetrieveOn(ctx SpecContext, subject ontology.ID, objects ...ontology.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionRetrieve, objects...)
}

func grantUpdateOn(ctx SpecContext, subject ontology.ID, objects ...ontology.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionUpdate, objects...)
}

func grantDeleteOn(ctx SpecContext, subject ontology.ID, objects ...ontology.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionDelete, objects...)
}
