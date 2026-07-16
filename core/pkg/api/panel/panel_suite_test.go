// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIPanel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Panel Suite")
}

var (
	db       *gorp.DB
	otg      *ontology.Ontology
	rbacSvc  *rbac.Service
	panelSvc *panel.Service
	userSvc  *user.Service
	apiSvc   *Service
	author   user.User
	parentID ontology.ID
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
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
	panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	apiSvc = &Service{access: rbacSvc, internal: panelSvc}
	author = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Username: "test"}))
	parentID = MustSucceed(groupSvc.CreateOrRetrieve(ctx, "panel-parent", ontology.RootID)).
		OntologyID()
})

// authedCtx returns a freighter.Context derived from ctx with the given user
// installed as the request subject, so auth.GetSubject succeeds inside the
// api.Service methods.
func authedCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", user.OntologyID(u.Key))
	return fctx
}

// newUser creates a fresh user. RBAC state is committed and shared across specs, so
// specs that grant a type-level permission (e.g. create on the panel type) must use a
// dedicated subject, or the grant would leak into specs asserting that permission is
// absent.
func newUser(ctx SpecContext) user.User {
	return MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Username: uuid.NewString(),
	}))
}

// grant creates a policy permitting action on the given objects, binds it to a
// fresh role, and assigns the role to subject. Writes commit directly to the
// database (nil tx) so the api enforcer - which reads committed state - observes
// them.
func grant(
	ctx SpecContext,
	subject ontology.ID,
	action access.Action,
	objects ...ontology.ID,
) {
	roleWriter := rbacSvc.Role.NewWriter(nil, true)
	policyWriter := rbacSvc.Policy.NewWriter(nil, true)
	r := &role.Role{Name: string(action) + "-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    string(action) + "-policy-" + uuid.New().String(),
		Objects: objects,
		Actions: []access.Action{action},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
