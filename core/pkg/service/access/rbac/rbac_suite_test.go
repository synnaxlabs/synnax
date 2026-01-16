// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db       *gorp.DB
	ctx      context.Context
	otg      *ontology.Ontology
	groupSvc *group.Service
	svc      *rbac.Service
	userSvc  *user.Service
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
	userSvc = MustSucceed(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	svc = MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
})

var _ = AfterEach(func() {
	Expect(svc.Close()).To(Succeed())
	Expect(userSvc.Close()).To(Succeed())
	Expect(groupSvc.Close()).To(Succeed())
	Expect(otg.Close()).To(Succeed())
})

func TestRBAC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RBAC Suite")
}
