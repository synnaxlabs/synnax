// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apicfg "github.com/synnaxlabs/synnax/pkg/api/config"
	apilibrary "github.com/synnaxlabs/synnax/pkg/api/library"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPILibrary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Library Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	rbacSvc *rbac.Service
	libSvc  *library.Service
	userSvc *user.Service
	apiSvc  *apilibrary.Service
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
	imexSvc := imex.NewService()
	taskSvc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Rack:     rackSvc,
		Status:   statusSvc,
		Search:   searchIdx,
		ImEx:     imexSvc,
		Configs:  MustSucceed(taskconfig.NewRegistry()),
	}))
	libSvc = MustOpen(library.OpenService(ctx, library.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Task:     taskSvc,
		Search:   searchIdx,
		ImEx:     imexSvc,
	}))
	apiSvc = MustSucceed(apilibrary.NewService(apicfg.LayerConfig{
		Distribution: &distribution.Layer{DB: db},
		Service:      &service.Layer{Library: libSvc, RBAC: rbacSvc},
	}))
})

func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Username: "library-" + uuid.New().String(),
	}))
}

func createLibrary(ctx SpecContext, name string) library.Library {
	GinkgoHelper()
	l := library.Library{Name: name}
	Expect(libSvc.NewWriter(nil).Create(ctx, &l)).To(Succeed())
	return l
}

// grantOn grants the action on the given objects to the subject through a fresh role.
// Writes commit directly so the enforcers, which read committed state, observe them.
func grantOn(
	ctx SpecContext,
	subject ontology.ID,
	action access.Action,
	objects ...ontology.ID,
) {
	GinkgoHelper()
	roleWriter := rbacSvc.Role.NewWriter(nil, true)
	policyWriter := rbacSvc.Policy.NewWriter(nil, true)
	r := &role.Role{
		Name:        string(action) + "-" + uuid.New().String(),
		Description: "test",
	}
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
