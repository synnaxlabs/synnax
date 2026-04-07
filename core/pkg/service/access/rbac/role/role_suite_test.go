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
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
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
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustSucceed(search.Open())
	g = MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	userSvc = MustSucceed(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
		Search:   searchIdx,
	}))
	policySvc = MustSucceed(policy.OpenService(ctx, policy.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	svc = MustSucceed(role.OpenService(ctx, role.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
		Search:   searchIdx,
	}))
})

var _ = AfterSuite(func(ctx SpecContext) {
	Expect(svc.Close()).To(Succeed())
	Expect(policySvc.Close()).To(Succeed())
	Expect(g.Close()).To(Succeed())
	Expect(otg.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
})

func TestRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Role Suite")
}
