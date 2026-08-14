// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apicfg "github.com/synnaxlabs/synnax/pkg/api/config"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	svc "github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIStatus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Status Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db        *gorp.DB
	otg       *ontology.Ontology
	rbacSvc   *rbac.Service
	statusSvc *status.Service
	labelSvc  *label.Service
	apiSvc    *apistatus.Service
	author    user.User
	userSvc   *user.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg, Search: searchIdx,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx,
	}))
	labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx,
	}))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx, User: userSvc,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Label: labelSvc, Search: searchIdx,
	}))
	apiSvc = MustSucceed(apistatus.NewService(apicfg.LayerConfig{
		Distribution: &distribution.Layer{DB: db},
		Service: &svc.Layer{
			Status: statusSvc,
			Label:  labelSvc,
			RBAC:   rbacSvc,
		},
	}))
	author = MustSucceed(
		userSvc.NewWriter(nil).Create(ctx, user.User{Username: "test"}),
	)
})

func authedCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.OntologyID())
	return fctx
}

// freshUser creates a user with no role assignments. Use this for "unauthorized"
// specs so accumulated grants on the shared author don't leak in.
func freshUser(ctx SpecContext) user.User {
	return MustSucceed(
		userSvc.NewWriter(nil).
			Create(ctx, user.User{Username: "anon-" + uuid.New().String()}),
	)
}

// grantOn assigns a role granting the given actions on the given objects to the
// subject. Mirrors the schematic suite helper.
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
