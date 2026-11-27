// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package invite_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestInvite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Auth OIDC Invite Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db      *gorp.DB
	userSvc *user.Service
	viewer  role.Key
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	StartRun(db.Run)
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{DB: db}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
	r := role.Role{Name: "viewer"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return roleSvc.NewWriter(tx, false).Create(ctx, &r)
	})).To(Succeed())
	viewer = r.Key
})

func createUser(ctx SpecContext) user.Key {
	GinkgoHelper()
	var u user.User
	Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		u, err = userSvc.NewWriter(tx).Create(
			ctx,
			user.User{Name: uuid.New().String(), Role: viewer},
		)
		return err
	})).To(Succeed())
	return u.Key
}
