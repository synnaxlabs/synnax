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
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db       *gorp.DB
	ctx      = context.Background()
	otg      *ontology.Ontology
	groupSvc *group.Service
	roleSvc  *role.Service
	svc      *policy.Service
	tx       gorp.Tx
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
})

var _ = AfterSuite(func() {
	Expect(db.Close()).To(Succeed())
})

var _ = BeforeEach(func() {
	ctx = context.Background()
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		DB:           db,
		EnableSearch: config.False(),
	}))
	groupSvc = MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	roleSvc = MustSucceed(role.OpenService(ctx, role.Config{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	svc = MustSucceed(policy.OpenService(ctx, policy.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	tx = db.OpenTx()
})

var _ = AfterEach(func() {
	Expect(tx.Close()).To(Succeed())
	Expect(svc.Close()).To(Succeed())
	Expect(roleSvc.Close()).To(Succeed())
	Expect(groupSvc.Close()).To(Succeed())
	Expect(otg.Close()).To(Succeed())
})

func TestPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Policy Suite")
}
