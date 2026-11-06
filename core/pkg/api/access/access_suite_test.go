// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiaccess "github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	userSvc *user.Service
	rbacSvc *rbac.Service
	apiSvc  *apiaccess.Service
	root    user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db := DeferClose(gorp.Wrap(memkv.New()))
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db,
	}))
	root, _ = MustSucceed2(user.ProvisionRoot(
		ctx,
		userSvc,
		passwordSvc,
		auth.Credentials{Username: "root", Password: "p"},
	))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	rbacSvc = MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     db,
		User:   userSvc,
		Status: statusSvc,
	})).RBAC
	apiSvc = MustSucceed(apiaccess.NewService(apiaccess.ServiceConfig{RBAC: rbacSvc}))
})

// rootCtx returns a context whose subject is the root user, which holds the Owner role.
func rootCtx(ctx SpecContext) freighter.Context { return userCtx(ctx, root) }

// userCtx returns a context whose subject is u.
func userCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.ResourceID())
	return fctx
}

func TestAPIAccess(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Access Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
