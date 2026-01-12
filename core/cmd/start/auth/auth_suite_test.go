// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx = context.Background()
	db  *gorp.DB
	otg *ontology.Ontology
	g   *group.Service
	svc *service.Layer
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Suite")
}

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
	g = MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg}))
	userSvc := MustSucceed(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	rbacSvc := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	svc = &service.Layer{
		User: userSvc,
		RBAC: rbacSvc,
		Auth: &auth.KV{DB: db},
	}
})

var _ = AfterSuite(func() {
	Expect(svc.RBAC.Close()).To(Succeed())
	Expect(g.Close()).To(Succeed())
	Expect(otg.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
})
