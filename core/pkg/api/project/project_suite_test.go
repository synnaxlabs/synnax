// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIProject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Project Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	rbacSvc    *rbac.Service
	projectSvc *project.Service
	logSvc     *log.Service
	panelSvc   *panel.Service
	userSvc    *user.Service
	apiSvc     *Service
	author     user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db := DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
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
	imexSvc := imex.NewService()
	panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	logSvc = MustOpen(log.OpenService(ctx, log.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
		ImEx:     imexSvc,
	}))
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
		ImEx:     imexSvc,
		Panel:    panelSvc,
	}))
	apiSvc = &Service{internal: projectSvc, access: rbacSvc}
	author = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Username: "test",
	}))
})

// authedCtx returns a freighter.Context derived from ctx with the given user installed
// as the request subject, so auth.GetSubject succeeds inside the api service.
func authedCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.OntologyID())
	return fctx
}

// createProject creates a project. It writes outside a transaction so the api
// enforcers, which read committed state, observe the new resource.
func createProject(ctx SpecContext, name string) project.Project {
	GinkgoHelper()
	p := project.Project{Name: name}
	Expect(projectSvc.NewWriter(nil).Create(ctx, &p)).To(Succeed())
	return p
}

// createLog creates a log under the project, committing it for the same reason
// createProject does.
func createLog(ctx SpecContext, proj project.Key, name string) log.Log {
	GinkgoHelper()
	l := log.Log{Name: name}
	Expect(logSvc.NewWriter(nil).Create(ctx, proj, &l)).To(Succeed())
	return l
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

func grantRetrieveOn(ctx SpecContext, subject ontology.ID, objects ...ontology.ID) {
	GinkgoHelper()
	grantOn(ctx, subject, access.ActionRetrieve, objects...)
}
