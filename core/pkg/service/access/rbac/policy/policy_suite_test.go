// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	db      *gorp.DB
	g       *group.Service
	svc     *policy.Service
	roleSvc *role.Service
	// owner is the role that holds each policy a spec creates.
	owner role.Role
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	g = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	roleSvc = MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	svc = MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
		DB:   db,
		Role: roleSvc,
	}))
	owner = role.Role{Name: "owner"}
	Expect(roleSvc.NewWriter(nil, true).Create(ctx, &owner)).To(Succeed())
})

func TestPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Access RBAC Policy Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
