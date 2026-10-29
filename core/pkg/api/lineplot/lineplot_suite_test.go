// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilineplot "github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	svcmock "github.com/synnaxlabs/synnax/pkg/service/mock"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPILinePlot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Line Plot Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apilineplot.Service
	proj   project.Project
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	svc = MustOpen(svcmock.OpenLayer(ctx, mock.NewNode(ctx)))
	apiSvc = MustSucceed(apilineplot.NewService(apilineplot.ServiceConfig{
		RBAC:     svc.RBAC,
		LinePlot: svc.LinePlot,
	}))
	proj = project.Project{Name: "test-project"}
	Expect(svc.Project.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
})

// newCreator creates a user that may create line plots.
func newCreator(ctx SpecContext) user.User {
	GinkgoHelper()
	u := MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
	roleWriter := svc.RBAC.Role.NewWriter(nil, true)
	policyWriter := svc.RBAC.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    "policy-" + uuid.New().String(),
		Objects: []ontology.ID{{Type: ontology.ResourceTypeLineplot}},
		Actions: []access.Action{access.ActionCreate},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, u.OntologyID(), r.Key)).To(Succeed())
	return u
}
