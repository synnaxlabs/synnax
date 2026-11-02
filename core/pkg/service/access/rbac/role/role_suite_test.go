// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	db        *gorp.DB
	otg       *ontology.Ontology
	g         *group.Service
	svc       *role.Service
	policySvc *policy.Service
	userSvc   *user.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	g = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	policySvc = MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	svc = MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
})

func TestRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Access RBAC Role Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
