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
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiuser "github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPIUser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API User Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db          *gorp.DB
	passwordSvc *password.Service
	userSvc     *user.Service
	writer      user.Writer
	apiSvc      *apiuser.Service
	root        user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg,
	}))
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
		DB: db,
		Authenticators: map[auth.Method]auth.Authenticator{
			auth.MethodPassword: passwordSvc,
		},
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Password: passwordSvc,
	}))
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "api-user-suite-root", Password: "p"},
		),
	)
	writer = userSvc.NewWriter(nil)
	rbacSvc := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		User:     userSvc,
	}))
	apiSvc = MustSucceed(apiuser.NewService(apiuser.ServiceConfig{
		RBAC:     rbacSvc,
		User:     userSvc,
		Password: passwordSvc,
		Auth:     authSvc,
	}))
	root = findRoot(ctx, userSvc)
})

func login(ctx context.Context, username, pwd string) (auth.Identity, error) {
	return passwordSvc.Authenticate(ctx, auth.Proof{
		Variant: auth.PasswordProof{
			Credentials: auth.Credentials{Username: username, Password: pwd},
		},
	})
}

func findRoot(ctx SpecContext, svc *user.Service) user.User {
	GinkgoHelper()
	var u user.User
	Expect(
		svc.NewRetrieve().Where(user.MatchRootUser(true)).Entry(&u).Exec(ctx, nil),
	).To(Succeed())
	return u
}

// rootCtx returns a freighter.Context with the suite-level root user installed as the
// request subject. The root user holds the Owner role, so every RBAC enforcement call
// passes.
func rootCtx(ctx SpecContext) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", root.OntologyID())
	return fctx
}
