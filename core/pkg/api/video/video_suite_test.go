// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apivideo "github.com/synnaxlabs/synnax/pkg/api/video"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIVideo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Video Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apivideo.Service
	proj   project.Project
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	svc = MustOpen(svctestutil.OpenLayer(ctx, testutil.NewNode(ctx)))
	apiSvc = MustSucceed(apivideo.NewService(apivideo.ServiceConfig{
		RBAC:  svc.RBAC,
		Video: svc.Video,
	}))
	proj = project.Project{Name: "test-project"}
	Expect(svc.Project.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
})

// newUser creates a user that may run actions on every video view.
func newUser(ctx SpecContext, actions ...access.Action) user.User {
	GinkgoHelper()
	u := MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
	if len(actions) == 0 {
		return u
	}
	roleWriter := svc.RBAC.Role.NewWriter(nil, true)
	policyWriter := svc.RBAC.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    "policy-" + uuid.New().String(),
		Objects: []ontology.ID{{Type: ontology.ResourceTypeVideo}},
		Actions: actions,
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, u.OntologyID(), r.Key)).To(Succeed())
	return u
}
