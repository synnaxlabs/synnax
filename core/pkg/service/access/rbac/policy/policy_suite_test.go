// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx     = context.Background()
	db      *gorp.DB
	otg     *ontology.Ontology
	g       *group.Service
	svc     *policy.Service
	roleSvc *role.Service
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
	g = MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
	svc = MustSucceed(policy.OpenService(ctx, policy.Config{
		DB:       db,
		Ontology: otg,
	}))
	roleSvc = MustSucceed(role.OpenService(ctx, role.Config{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
})

var _ = AfterSuite(func() {
	Expect(roleSvc.Close()).To(Succeed())
	Expect(svc.Close()).To(Succeed())
	Expect(g.Close()).To(Succeed())
	Expect(otg.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
})

func TestPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Policy Suite")
}
