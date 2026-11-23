// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package oidc_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestOIDC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Auth OIDC Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db      *gorp.DB
	userSvc *user.Service
	// viewer is the role of the users that first logins create.
	viewer   role.Key
	provider *testutil.Provider
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	userSvc, viewer = openUsers(ctx, db)
	provider = testutil.NewProvider()
	DeferCleanup(provider.Close)
})

// openUsers opens a user service on db and creates a viewer role in it.
func openUsers(ctx SpecContext, db *gorp.DB) (*user.Service, role.Key) {
	GinkgoHelper()
	StartRun(db.Run)
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	users := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Password: passwordSvc,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
	r := role.Role{Name: "viewer"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return roleSvc.NewWriter(tx, false).Create(ctx, &r)
	})).To(Succeed())
	return users, r.Key
}

// createUser creates a user with the viewer role and returns its resource ID.
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

// link links account to subject in its own transaction.
func link(
	ctx SpecContext,
	svc *oidc.Service,
	subject user.Key,
	account oidc.Account,
) (l oidc.Link, err error) {
	err = db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		l, err = svc.NewWriter(tx).Create(ctx, subject, account)
		return err
	})
	return l, err
}
