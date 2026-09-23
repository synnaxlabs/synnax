// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apicfg "github.com/synnaxlabs/synnax/pkg/api/config"
	apiverification "github.com/synnaxlabs/synnax/pkg/api/verification"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	svc "github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	svcverification "github.com/synnaxlabs/synnax/pkg/service/channel/verification"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	svcmock "github.com/synnaxlabs/synnax/pkg/service/mock"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIVerification(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Verification Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db      *gorp.DB
	rbacSvc *rbac.Service
	verSvc  *svcverification.Service
	apiSvc  *apiverification.Service
	userSvc *user.Service
	keys    svcmock.Keys
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	kvDB := memkv.New()
	db = DeferClose(gorp.Wrap(kvDB))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg, Search: searchIdx,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx,
	}))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx, User: userSvc,
	}))
	keys = svcmock.NewKeys()
	verSvc = MustOpen(svcverification.OpenService(ctx, svcverification.ServiceConfig{
		DB: kvDB, Anchors: keys.Anchors,
	}))
	apiSvc = MustSucceed(apiverification.NewService(apicfg.LayerConfig{
		Distribution: &distribution.Layer{DB: db},
		Service:      &svc.Layer{RBAC: rbacSvc, Verification: verSvc},
	}))
})

// freshUser creates a user with no role assignments.
func freshUser(ctx SpecContext) user.User {
	return MustSucceed(
		userSvc.NewWriter(nil).
			Create(ctx, user.User{Username: "anon-" + uuid.New().String()}),
	)
}

// grantOn assigns a role granting the given actions on the given objects to the
// subject.
func grantOn(
	ctx SpecContext,
	subject ontology.ID,
	actions []access.Action,
	objects ...ontology.ID,
) {
	roleWriter := rbacSvc.Role.NewWriter(nil, true)
	policyWriter := rbacSvc.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: actions,
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())
}
