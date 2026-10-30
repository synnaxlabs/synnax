// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apidevice "github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIDevice(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Device Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apidevice.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apidevice.NewService(apidevice.ServiceConfig{
		Device:   svc.Device,
		Status:   svc.Status,
		RBAC:     svc.RBAC,
		Ontology: svc.Ontology,
	}))
})

// createUser creates a user with no role assignments.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// grant grants the action on the given objects to the subject through a fresh role.
func grant(
	ctx SpecContext,
	action access.Action,
	subject ontology.ID,
	objects ...ontology.ID,
) {
	GinkgoHelper()
	roleWriter := svc.RBAC.Role.NewWriter(nil, true)
	policyWriter := svc.RBAC.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: []access.Action{action},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())
}
