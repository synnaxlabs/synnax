// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPITask(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Task Suite")
}

var (
	db        *gorp.DB
	otg       *ontology.Ontology
	rbacSvc   *rbac.Service
	taskSvc   *task.Service
	statusSvc *status.Service
	rackSvc   *rack.Service
	apiSvc    *Service
	author    user.User
	userSvc   *user.Service
	testRack  *rack.Rack
)

var _ = BeforeSuite(func(ctx SpecContext) {
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.Open())
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg, Search: searchIdx,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx,
	}))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx, User: userSvc,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Label: labelSvc, Search: searchIdx,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		Ontology:            otg,
		Group:               g,
		HostProvider:        mock.StaticHostKeyProvider(1),
		Status:              statusSvc,
		HealthCheckInterval: telem.Hour,
		Search:              searchIdx,
	}))
	taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Rack: rackSvc, Status: statusSvc, Search: searchIdx,
	}))
	apiSvc = &Service{
		db:     db,
		access: rbacSvc,
		task:   taskSvc,
	}
	testRack = &rack.Rack{Name: "Test Rack"}
	Expect(rackSvc.NewWriter(db).Create(ctx, testRack)).To(Succeed())
	author = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Username: "test"}))
})

func authedCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", user.OntologyID(u.Key))
	return fctx
}

// grantOn assigns a role granting the given actions on the given objects to the
// subject. Mirrors the api/status suite helper.
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
