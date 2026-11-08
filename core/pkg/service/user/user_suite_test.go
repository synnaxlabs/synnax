// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	db          *gorp.DB
	svc         *user.Service
	writer      user.Writer
	passwordSvc *password.Service
	roleSvc     *role.Service
	// member is a role for the users that specs create.
	member role.Key
	// owner is the role of the root user.
	owner role.Key
)

func TestUser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service User Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	svc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Password: passwordSvc,
	}))
	roleSvc = MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	member = createRole(ctx, nil, "member")
	owner = createRole(ctx, nil, "owner")
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			svc,
			passwordSvc,
			auth.Credentials{Username: "suite-root", Password: "p"},
			owner,
		),
	)
	writer = svc.NewWriter(nil)
})

// createRole creates a role named name in tx and returns its key.
func createRole(ctx SpecContext, tx gorp.Tx, name string) role.Key {
	GinkgoHelper()
	r := role.Role{Name: name}
	Expect(roleSvc.NewWriter(tx, false).Create(ctx, &r)).To(Succeed())
	return r.Key
}
