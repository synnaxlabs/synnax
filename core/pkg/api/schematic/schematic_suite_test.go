// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPISchematic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Schematic Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db       *gorp.DB
	otg      *ontology.Ontology
	rbacSvc  *rbac.Service
	schemSvc *schematic.Service
	apiSvc   *Service
	ws       workspace.Workspace
	author   user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.Open())
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
		Search:   searchIdx,
	}))
	workspaceSvc := MustOpen(workspace.OpenService(ctx, workspace.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
		Search:   searchIdx,
	}))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
		Search:   searchIdx,
		User:     userSvc,
	}))
	schemSvc = MustOpen(schematic.OpenService(ctx, schematic.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	apiSvc = &Service{db: db, internal: schemSvc, access: rbacSvc}
	author = user.User{Username: "test"}
	Expect(userSvc.NewWriter(nil).Create(ctx, &author)).To(Succeed())
	ws.Author = author.Key
	Expect(workspaceSvc.NewWriter(nil).Create(ctx, &ws)).To(Succeed())
})

// authedCtx returns a freighter.Context derived from ctx with the given user
// installed as the request subject. Callers must pass the returned Context as
// the ctx argument to api.Service methods so auth.GetSubject succeeds.
func authedCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", user.OntologyID(u.Key))
	return fctx
}

// grantUpdateOn creates a policy granting ActionUpdate on the given objects to
// a fresh role and assigns the role to the given subject. Writes commit
// directly to the database so the api.Service.Dispatch enforcer (which reads
// committed state with no transaction) can observe them.
func grantUpdateOn(ctx SpecContext, subject ontology.ID, objects ...ontology.ID) {
	roleWriter := rbacSvc.Role.NewWriter(nil, true)
	policyWriter := rbacSvc.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "update-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    "update-policy-" + uuid.New().String(),
		Objects: objects,
		Actions: []access.Action{access.ActionUpdate},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())
}
