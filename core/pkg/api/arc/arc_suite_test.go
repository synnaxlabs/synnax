// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIArc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Arc Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db      *gorp.DB
	otg     *ontology.Ontology
	rbacSvc *rbac.Service
	arcSvc  *arc.Service
	apiSvc  *Service
	author  user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := mock.NewNode(ctx)
	db = node.DB
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
		Search:   searchIdx,
	}))
	rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		Ontology:            otg,
		Group:               groupSvc,
		HostProvider:        mock.NewStaticHostProvider(1),
		Status:              statusSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
		Search:              searchIdx,
	}))
	taskSvc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Rack:     rackSvc,
		Status:   statusSvc,
		Search:   searchIdx,
	}))
	channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           db,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Status:       statusSvc,
		Search:       searchIdx,
	}))
	arcSvc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Channel:  channelSvc,
		Task:     taskSvc,
		Search:   searchIdx,
	}))
	authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
	userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
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
	apiSvc = &Service{internal: arcSvc, access: rbacSvc, status: statusSvc}
	author = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Username: "test"}))
})

// authedCtx returns a freighter.Context derived from ctx with the given user
// installed as the request subject so auth.GetSubject succeeds.
func authedCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.OntologyID())
	return fctx
}

// grantUpdateOn creates a policy granting ActionUpdate on the given objects to
// a fresh role and assigns the role to the given subject. Writes commit directly
// to the database so the api.Service.Dispatch enforcer (which reads committed
// state with no transaction) can observe them.
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
